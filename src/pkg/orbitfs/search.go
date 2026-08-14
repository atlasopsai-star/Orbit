package orbitfs

import (
	"bufio"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type SearchMode int

const (
	FilenameSearch SearchMode = iota
	ContentSearch
)

type SearchResult struct {
	Path string
	Line int
	Text string
}

var ignoredDirectories = map[string]bool{".git": true, "node_modules": true, "target": true, ".next": true, "dist": true, "build": true, "DerivedData": true}

func Search(ctx context.Context, root, query string, mode SearchMode) ([]SearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, nil
	}
	results := make([]SearchResult, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if entry.IsDir() {
			if path != root && ignoredDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if mode == FilenameSearch {
			if strings.Contains(strings.ToLower(entry.Name()), query) {
				results = append(results, SearchResult{Path: path})
			}
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		reader := bufio.NewScanner(io.LimitReader(file, 1024*1024))
		reader.Buffer(make([]byte, 4096), 1024*1024)
		lineNumber := 0
		for reader.Scan() {
			lineNumber++
			line := reader.Text()
			if strings.IndexByte(line, 0) >= 0 {
				return nil
			}
			if strings.Contains(strings.ToLower(line), query) {
				results = append(results, SearchResult{Path: path, Line: lineNumber, Text: strings.TrimSpace(line)})
				break
			}
		}
		if reader.Err() != nil {
			return nil
		}
		return nil
	})
	return results, err
}
