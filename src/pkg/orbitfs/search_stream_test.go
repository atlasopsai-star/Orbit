package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchStreamRespectsGitignoreAndEmitsMultipleContentMatches(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "ignored"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored", "auth.go"), []byte("authentication\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.go"), []byte("authentication one\nother\nauthentication two\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var results []SearchResult
	for batch := range SearchStream(context.Background(), root, "authentication", ContentSearch, MaxSearchResults) {
		if batch.Err != nil {
			t.Fatal(batch.Err)
		}
		results = append(results, batch.Results...)
	}
	if len(results) != 2 || results[0].Line != 1 || results[1].Line != 3 {
		t.Fatalf("results = %#v", results)
	}
}

func TestSearchStreamAppliesNestedGitignoreNegation(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("nested/*.go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, ".gitignore"), []byte("!keep.go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "keep.go"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "skip.go"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	results, err := Search(context.Background(), root, "keep", FilenameSearch)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || filepath.Base(results[0].Path) != "keep.go" {
		t.Fatalf("results = %#v", results)
	}
}

func TestSearchStreamCapsResults(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < 5; index++ {
		if err := os.WriteFile(filepath.Join(root, "match"+string(rune('a'+index))+".go"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}
	count, truncated := 0, false
	for batch := range SearchStream(context.Background(), root, "match", FilenameSearch, 2) {
		count += len(batch.Results)
		truncated = truncated || batch.Truncated
	}
	if count != 2 || !truncated {
		t.Fatalf("count=%d truncated=%v", count, truncated)
	}
}

func TestSearchStreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for batch := range SearchStream(ctx, t.TempDir(), "match", FilenameSearch, MaxSearchResults) {
		if batch.Err != nil {
			t.Fatal(batch.Err)
		}
	}
}
