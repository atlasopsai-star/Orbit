package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestRankLookupPriority(t *testing.T) {
	cases := []struct {
		query string
		name  string
		want  int
		ok    bool
	}{{"OliCode", "OliCode", 0, true}, // exact
		{"olicode", "OliCode", 1, true},      // case-insensitive exact
		{"oli", "OliCode", 2, true},          // basename prefix
		{"proj", "my-project", 3, true},      // token prefix
		{"code", "OliCode", 4, true},         // substring
		{"oicd", "OliCode", 5, true},         // fuzzy subsequence
		{"xyz", "OliCode", 0, false},         // no match
		{"", "OliCode", 0, false},            // empty query never matches
		{"resume", "Resume", 1, true},        // exact ignoring case
		{"resume", "Resume Assets", 2, true}, // prefix beats nothing else
	}
	for _, tc := range cases {
		got, ok := rankLookup(tc.query, tc.name)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("rankLookup(%q, %q) = (%d, %v), want (%d, %v)", tc.query, tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveLookupRootsFiltersAndDedupes(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(tmp, "missing")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	roots := ResolveLookupRoots(ScopeEverywhere, tmp, []string{tmp, sub, missing, ""})
	// EVERYWHERE is home plus the configured roots, deduplicated and filtered.
	if len(roots) != 3 {
		t.Fatalf("expected home+tmp+sub after dedupe, got %v", roots)
	}
	seenHome, seenSub := false, false
	for _, root := range roots {
		if root == missing {
			t.Fatal("missing root must be filtered out")
		}
		if root == home {
			seenHome = true
		}
		if root == sub {
			seenSub = true
		}
	}
	if !seenHome || !seenSub {
		t.Fatalf("expected home and sub among roots, got %v", roots)
	}
	if got := ResolveLookupRoots(ScopeCurrent, sub, nil); len(got) != 1 || got[0] != sub {
		t.Fatalf("ScopeCurrent should resolve to the cwd, got %v", got)
	}
}

func TestLookupFindsFilesAndFolders(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "AlphaDir"))
	mustWrite(t, filepath.Join(root, "alpha.txt"), "x")
	mustMkdir(t, filepath.Join(root, "sub"))
	mustWrite(t, filepath.Join(root, "sub", "beta.txt"), "x")
	mustWrite(t, filepath.Join(root, "other.log"), "x")

	results, _, err := Lookup(context.Background(), "alp", []string{root}, LookupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]LookupResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	dir, ok := byName["AlphaDir"]
	if !ok || !dir.IsDir {
		t.Fatalf("expected AlphaDir folder result, got %v", results)
	}
	file, ok := byName["alpha.txt"]
	if !ok || file.IsDir {
		t.Fatalf("expected alpha.txt file result, got %v", results)
	}
	if file.Parent != root {
		t.Fatalf("parent should be %s, got %s", root, file.Parent)
	}
	if len(results) != 2 {
		t.Fatalf("expected exactly 2 matches, got %v", results)
	}
}

func TestLookupHonorsHiddenFilter(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".hidden.txt"), "x")
	mustMkdir(t, filepath.Join(root, ".hiddendir"))
	mustWrite(t, filepath.Join(root, "visible.txt"), "x")

	hiddenOff, _, err := Lookup(context.Background(), "hid", []string{root}, LookupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hiddenOff) != 0 {
		t.Fatalf("hidden entries must be excluded by default, got %v", hiddenOff)
	}

	hiddenOn, _, err := Lookup(context.Background(), "hid", []string{root}, LookupOptions{IncludeHidden: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hiddenOn) != 2 {
		t.Fatalf("hidden entries must be included when requested, got %v", hiddenOn)
	}
}

func TestLookupDeduplicatesOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "report.txt"), "x")
	results, _, err := Lookup(context.Background(), "report", []string{root, filepath.Join(root, "sub-missing")}, LookupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range results {
		if seen[r.Path] {
			t.Fatalf("duplicate path %s", r.Path)
		}
		seen[r.Path] = true
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", results)
	}
}

func TestLookupCapsResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		mustWrite(t, filepath.Join(root, "match-"+strconv.Itoa(i)+".txt"), "x")
	}
	results, truncated, err := Lookup(context.Background(), "match", []string{root}, LookupOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	if !truncated {
		t.Fatal("expected truncated flag")
	}
}

func TestLookupCancellation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 200; i++ {
		mustWrite(t, filepath.Join(root, "file-"+strconv.Itoa(i)+".txt"), "x")
	}
	ctx, cancel := context.WithCancel(context.Background())
	stream := LookupStream(ctx, "file", []string{root}, LookupOptions{})
	// Read one batch, then cancel mid-stream: the stream must end promptly.
	for batch := range stream {
		if batch.Done {
			break
		}
		cancel()
	}
	cancel()
	start := time.Now()
	for range LookupStream(context.Background(), "file", []string{root}, LookupOptions{Limit: 5}) {
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("LookupStream should not hang after completion")
	}
}

func TestWithinLookupRootsBounds(t *testing.T) {
	roots := []string{"/a/b"}
	within := map[string]bool{
		"/a/b":       true,
		"/a/b/c/d":   true,
		"/a/bcd":     false, // prefix trap
		"/a/c":       false,
		"/a":         false,
		"/other/b/c": false,
	}
	for path, want := range within {
		if got := withinLookupRoots(path, roots); got != want {
			t.Errorf("withinLookupRoots(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestLookupSortsDirsFirstAtEqualScore(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "zebra-dir"))
	mustWrite(t, filepath.Join(root, "zebra-file.txt"), "x")
	results, _, err := Lookup(context.Background(), "zebra", []string{root}, LookupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %v", results)
	}
	if !results[0].IsDir {
		t.Fatalf("folders should rank before files at equal score, got %v", results)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
