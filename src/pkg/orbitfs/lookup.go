package orbitfs

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// LookupScope selects which roots Orbit Lookup searches.
type LookupScope int

const (
	ScopeCurrent LookupScope = iota
	ScopeHome
	ScopeProjects
	ScopeExternal
	ScopeEverywhere
)

// String returns the human readable scope label shown in the Lookup header.
func (s LookupScope) String() string {
	switch s {
	case ScopeCurrent:
		return "CURRENT"
	case ScopeHome:
		return "HOME"
	case ScopeProjects:
		return "PROJECTS"
	case ScopeExternal:
		return "EXTERNAL"
	default:
		return "EVERYWHERE"
	}
}

const (
	MaxLookupResults = 200
	lookupFlushEvery = 24
)

// lookupIgnoredDirectories prunes noisy or recursive directories from global
// traversal so Lookup stays bounded and useful. `.Trash` must never be walked.
var lookupIgnoredDirectories = map[string]bool{
	".git": true, "node_modules": true, "target": true, ".next": true,
	"dist": true, "build": true, "DerivedData": true, ".cache": true,
	".npm": true, ".Trash": true, "Library": true,
}

// LookupOptions controls how a Lookup stream behaves.
type LookupOptions struct {
	IncludeHidden bool // include dotfiles (default: skip them)
	UseSpotlight  bool // query the macOS Spotlight index alongside the walker
	Limit         int  // max results; 0 means MaxLookupResults
}

// LookupResult is one matched file or folder. Metadata is deliberately cheap:
// expensive enrichment (git, sizes) happens on the selected result only.
type LookupResult struct {
	Path     string
	Name     string
	IsDir    bool
	Ext      string
	Parent   string
	ModTime  time.Time
	Symlink  bool
	Provider string // "fs" or "spotlight"
	Score    int
}

// LookupBatch is one incremental slice of ranked results.
type LookupBatch struct {
	Results   []LookupResult
	Done      bool
	Truncated bool
	Err       error
}

// ResolveLookupRoots expands a scope into the concrete existing directories to
// search. Extra configured roots (lookup_roots) always join EVERYWHERE, and
// only roots that exist on disk are returned.
func ResolveLookupRoots(scope LookupScope, cwd string, extra []string) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = cwd
	}
	var candidates []string
	switch scope {
	case ScopeCurrent:
		candidates = []string{cwd}
	case ScopeHome:
		candidates = []string{home}
	case ScopeProjects:
		for _, dir := range []string{"Developer", "Projects", "Documents", "Desktop", "Downloads", "Archive"} {
			candidates = append(candidates, filepath.Join(home, dir))
		}
	case ScopeExternal:
		for _, root := range extra {
			if isExternalRoot(root) {
				candidates = append(candidates, root)
			}
		}
		// No configured external roots: fall back to mounted volumes so an
		// SSD plugged in for the first time is still reachable on demand.
		if len(candidates) == 0 {
			if entries, err := os.ReadDir("/Volumes"); err == nil {
				for _, entry := range entries {
					candidates = append(candidates, filepath.Join("/Volumes", entry.Name()))
				}
			}
		}
	case ScopeEverywhere:
		candidates = append(candidates, home)
		candidates = append(candidates, extra...)
	}
	return dedupeExisting(candidates)
}

func isExternalRoot(path string) bool {
	return strings.HasPrefix(filepath.Clean(path), "/Volumes"+string(filepath.Separator))
}

func dedupeExisting(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		if info, err := os.Stat(clean); err == nil && info.IsDir() {
			out = append(out, clean)
		}
	}
	return out
}

// rankLookup scores a basename against the query. Lower is better, matching
// the Lookup ranking priority: exact, case-insensitive exact, prefix, token
// prefix, substring, fuzzy subsequence. A weak path match never outranks a
// strong basename match because only basenames are ranked here.
func rankLookup(query, name string) (int, bool) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return 0, false
	}
	if name == trimmed {
		return 0, true
	}
	q := strings.ToLower(trimmed)
	n := strings.ToLower(name)
	if n == q {
		return 1, true
	}
	if strings.HasPrefix(n, q) {
		return 2, true
	}
	for _, token := range strings.FieldsFunc(n, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if strings.HasPrefix(token, q) {
			return 3, true
		}
	}
	if strings.Contains(n, q) {
		return 4, true
	}
	if fuzzyMatch(q, n) {
		return 5, true
	}
	return 0, false
}

func fuzzyMatch(query, name string) bool {
	i := 0
	for j := 0; i < len(query) && j < len(name); j++ {
		if query[i] == name[j] {
			i++
		}
	}
	return i == len(query)
}

func sortLookupResults(results []LookupResult) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score < results[j].Score
		}
		if results[i].IsDir != results[j].IsDir {
			return results[i].IsDir
		}
		return results[i].Name < results[j].Name
	})
}

// LookupStream streams ranked matches for a query across the given roots. The
// filesystem walker always runs; on macOS the Spotlight provider (mdfind) runs
// in parallel to surface indexed results fast. Results are deduplicated by
// path, capped at opts.Limit, and emitted in ranked order. Cancelling ctx stops
// every provider promptly.
func LookupStream(ctx context.Context, query string, roots []string, opts LookupOptions) <-chan LookupBatch {
	out := make(chan LookupBatch, 1)
	go func() {
		defer close(out)
		query = strings.TrimSpace(query)
		if query == "" || len(roots) == 0 {
			out <- LookupBatch{Done: true}
			return
		}
		limit := opts.Limit
		if limit <= 0 {
			limit = MaxLookupResults
		}

		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		feed := make(chan LookupResult, 64)
		var wg sync.WaitGroup
		for _, root := range roots {
			wg.Add(1)
			go func(root string) {
				defer wg.Done()
				walkLookupRoot(childCtx, root, query, opts.IncludeHidden, feed)
			}(root)
		}
		if opts.UseSpotlight && runtime.GOOS == "darwin" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				spotlightLookup(childCtx, query, roots, opts.IncludeHidden, feed)
			}()
		}
		go func() {
			wg.Wait()
			close(feed)
		}()

		results := make([]LookupResult, 0, limit)
		seen := make(map[string]struct{}, limit)
		count := 0
		truncated := false
		emit := func(done bool, err error) bool {
			if count == 0 && !done {
				return true
			}
			sortLookupResults(results)
			batch := append([]LookupResult(nil), results...)
			select {
			case out <- LookupBatch{Results: batch, Done: done, Truncated: truncated, Err: err}:
				return true
			case <-ctx.Done():
				return false
			}
		}
		for result := range feed {
			if truncated {
				continue
			}
			if _, dup := seen[result.Path]; dup {
				continue
			}
			seen[result.Path] = struct{}{}
			results = append(results, result)
			count++
			if count >= limit {
				truncated = true
				cancel() // stop providers: the cap is already full
			} else if count%lookupFlushEvery == 0 {
				if !emit(false, nil) {
					return
				}
			}
		}
		if ctx.Err() != nil {
			return
		}
		if truncated {
			_ = emit(true, nil)
			return
		}
		if !emit(true, nil) {
			return
		}
	}()
	return out
}

// Lookup is the non-streaming convenience wrapper over LookupStream.
func Lookup(ctx context.Context, query string, roots []string, opts LookupOptions) ([]LookupResult, bool, error) {
	results := make([]LookupResult, 0)
	truncated := false
	for batch := range LookupStream(ctx, query, roots, opts) {
		if batch.Err != nil {
			return results, truncated, batch.Err
		}
		results = batch.Results
		truncated = batch.Truncated
		if batch.Done {
			break
		}
	}
	return results, truncated, nil
}

func walkLookupRoot(ctx context.Context, root, query string, includeHidden bool, feed chan<- LookupResult) {
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Permission errors are skipped silently; global search must not
			// stop at the first inaccessible directory.
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		name := entry.Name()
		if path != root && !includeHidden && isHiddenName(name) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() && path != root && lookupIgnoredDirectories[name] {
			return filepath.SkipDir
		}
		score, ok := rankLookup(query, name)
		if !ok {
			return nil
		}
		result := lookupResultFromEntry(path, entry, score, "fs")
		select {
		case feed <- result:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func spotlightLookup(ctx context.Context, query string, roots []string, includeHidden bool, feed chan<- LookupResult) {
	if _, err := exec.LookPath("mdfind"); err != nil {
		return
	}
	cmd := exec.CommandContext(ctx, "mdfind", "-name", query)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	defer func() { _ = cmd.Wait() }()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path == "" || !withinLookupRoots(path, roots) {
			continue
		}
		if !includeHidden && containsHiddenComponent(path) {
			continue
		}
		if containsIgnoredLookupComponent(path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		score, ok := rankLookup(query, name)
		if !ok {
			continue
		}
		result := LookupResult{
			Path: path, Name: name, IsDir: info.IsDir(),
			Ext: strings.TrimPrefix(filepath.Ext(name), "."), Parent: filepath.Dir(path),
			ModTime: info.ModTime(), Symlink: info.Mode()&os.ModeSymlink != 0,
			Provider: "spotlight", Score: score,
		}
		select {
		case feed <- result:
		case <-ctx.Done():
			return
		}
	}
}

func withinLookupRoots(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func containsHiddenComponent(path string) bool {
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		if isHiddenName(part) {
			return true
		}
	}
	return false
}

// containsIgnoredLookupComponent reports whether any path component is in the
// ignored set, keeping the Spotlight provider consistent with the filesystem
// walker (which prunes those directories entirely).
func containsIgnoredLookupComponent(path string) bool {
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		if lookupIgnoredDirectories[part] {
			return true
		}
	}
	return false
}

func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".")
}

func lookupResultFromEntry(path string, entry fs.DirEntry, score int, provider string) LookupResult {
	name := entry.Name()
	result := LookupResult{
		Path: path, Name: name, IsDir: entry.IsDir(),
		Ext: strings.TrimPrefix(filepath.Ext(name), "."), Parent: filepath.Dir(path),
		Symlink:  entry.Type()&fs.ModeSymlink != 0,
		Provider: provider, Score: score,
	}
	if info, err := entry.Info(); err == nil {
		result.ModTime = info.ModTime()
	}
	return result
}
