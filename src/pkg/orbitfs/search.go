package orbitfs

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

type SearchMode int

const (
	FilenameSearch SearchMode = iota
	ContentSearch
)

const (
	MaxSearchResults = 200
	searchBatchSize  = 16
	searchReadLimit  = 1024 * 1024
)

type SearchResult struct {
	Path string
	Line int
	Text string
}

type SearchBatch struct {
	Results   []SearchResult
	Done      bool
	Truncated bool
	Err       error
}

var ignoredDirectories = map[string]bool{".git": true, "node_modules": true, "target": true, ".next": true, "dist": true, "build": true, "DerivedData": true}
var errSearchLimit = errors.New("search result limit reached")

type ignoreRule struct {
	matcher *ignore.GitIgnore
	negate  bool
}

type ignoreMatcher struct {
	root    string
	rules   map[string][]ignoreRule
	checked map[string]bool
}

func newIgnoreMatcher(root string) *ignoreMatcher {
	return &ignoreMatcher{
		root:    root,
		rules:   make(map[string][]ignoreRule),
		checked: make(map[string]bool),
	}
}

func (m *ignoreMatcher) ignored(path string) bool {
	if path == m.root {
		return false
	}
	ancestors := make([]string, 0, 4)
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		ancestors = append(ancestors, dir)
		if dir == m.root || dir == filepath.Dir(dir) {
			break
		}
	}
	ignored := false
	// Apply ignore files from the search root toward the selected path. A
	// matching negated rule re-includes the path, so child .gitignore files can
	// override a parent rule instead of being treated as a second OR condition.
	for index := len(ancestors) - 1; index >= 0; index-- {
		dir := ancestors[index]
		if !m.checked[dir] {
			m.checked[dir] = true
			ignorePath := filepath.Join(dir, ".gitignore")
			if contents, err := os.ReadFile(ignorePath); err == nil {
				for _, line := range strings.Split(string(contents), "\n") {
					line = strings.TrimSuffix(line, "\r")
					negate := strings.HasPrefix(line, "!") && !strings.HasPrefix(line, `\\!`)
					pattern := strings.TrimPrefix(line, "!")
					if !negate {
						pattern = line
					} else if strings.HasPrefix(pattern, "!") {
						// A second leading bang is literal after the
						// rule's negation marker.
						pattern = "\\" + pattern
					}
					compiled := ignore.CompileIgnoreLines(pattern)
					if compiled != nil {
						m.rules[dir] = append(m.rules[dir], ignoreRule{matcher: compiled, negate: negate})
					}
				}
			}
		}
		rules := m.rules[dir]
		if len(rules) == 0 {
			continue
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			continue
		}
		relative = filepath.Clean(filepath.ToSlash(relative))
		for _, rule := range rules {
			if rule.matcher.MatchesPath(relative) {
				// Git ignore rules are ordered: the last matching rule wins.
				// Keep the negation bit outside go-gitignore because its
				// MatchesPathHow API discards a final negated match.
				ignored = !rule.negate
			}
		}
	}
	return ignored
}

func Search(ctx context.Context, root, query string, mode SearchMode) ([]SearchResult, error) {
	results := make([]SearchResult, 0)
	for batch := range SearchStream(ctx, root, query, mode, MaxSearchResults) {
		if batch.Err != nil {
			return results, batch.Err
		}
		results = append(results, batch.Results...)
	}
	return results, nil
}

func SearchStream(ctx context.Context, root, query string, mode SearchMode, limit int) <-chan SearchBatch {
	out := make(chan SearchBatch, 1)
	go func() {
		defer close(out)
		query = strings.ToLower(strings.TrimSpace(query))
		if query == "" {
			out <- SearchBatch{Done: true}
			return
		}
		if limit <= 0 {
			limit = MaxSearchResults
		}
		matcher := newIgnoreMatcher(root)
		batch := make([]SearchResult, 0, searchBatchSize)
		count := 0
		truncated := false
		emit := func(done bool, err error) bool {
			if len(batch) == 0 && !done && err == nil {
				return true
			}
			message := SearchBatch{Results: batch, Done: done, Truncated: truncated, Err: err}
			select {
			case out <- message:
				batch = make([]SearchResult, 0, searchBatchSize)
				return true
			case <-ctx.Done():
				return false
			}
		}
		appendResult := func(result SearchResult) error {
			if count >= limit {
				truncated = true
				return errSearchLimit
			}
			batch = append(batch, result)
			count++
			if len(batch) >= searchBatchSize && !emit(false, nil) {
				return ctx.Err()
			}
			return nil
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if entry.IsDir() {
				if path != root && (ignoredDirectories[entry.Name()] || matcher.ignored(path)) {
					// Git cannot re-include a descendant once its parent
					// directory is ignored, so pruning keeps searches bounded.
					return filepath.SkipDir
				}
				return nil
			}
			if matcher.ignored(path) {
				return nil
			}
			if mode == FilenameSearch {
				if strings.Contains(strings.ToLower(entry.Name()), query) {
					return appendResult(SearchResult{Path: path})
				}
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			scanner := bufio.NewScanner(io.LimitReader(file, searchReadLimit))
			scanner.Buffer(make([]byte, 4096), searchReadLimit)
			lineNumber := 0
			for scanner.Scan() {
				lineNumber++
				line := scanner.Text()
				if strings.IndexByte(line, 0) >= 0 {
					break
				}
				if strings.Contains(strings.ToLower(line), query) {
					if err := appendResult(SearchResult{Path: path, Line: lineNumber, Text: strings.TrimSpace(line)}); err != nil {
						_ = file.Close()
						return err
					}
				}
			}
			_ = file.Close()
			return nil
		})
		if errors.Is(err, errSearchLimit) {
			err = nil
		} else if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			_ = emit(true, err)
			return
		}
		if !emit(true, nil) {
			return
		}
	}()
	return out
}
