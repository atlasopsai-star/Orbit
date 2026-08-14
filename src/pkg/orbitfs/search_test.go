package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestSearchFilenameAndContentSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.go"), []byte("authentication middleware\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "auth.go"), []byte("authentication"), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := Search(context.Background(), root, "auth", FilenameSearch)
	if err != nil || len(files) != 1 {
		t.Fatalf("filename results = %#v, %v", files, err)
	}
	content, err := Search(context.Background(), root, "middleware", ContentSearch)
	if err != nil || len(content) != 1 || content[0].Line != 1 {
		t.Fatalf("content results = %#v, %v", content, err)
	}
}

func TestSearchHonorsOrderedGitignoreNegation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\n!important.log\nimportant.log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ignored.log", "important.log"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("log"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), root, "log", FilenameSearch)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("ordered rules returned %#v, want no results", results)
	}
}

func TestSearchAllowsNestedGitignoreNegation(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, ".gitignore"), []byte("!keep.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"keep.tmp", "drop.tmp"} {
		if err := os.WriteFile(filepath.Join(nested, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), root, "tmp", FilenameSearch)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != filepath.Join(nested, "keep.tmp") {
		t.Fatalf("nested negation returned %#v, want keep.tmp only", results)
	}
}

func TestSearchHandlesGitignoreCommentsCRLFAndEscapedBang(t *testing.T) {
	root := t.TempDir()
	contents := "# comment\r\n*\r\n!keep.log\r\n!!literal.txt\r\n\\!literal-ignored.txt\r\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ignored.log", "keep.log", "!literal.txt", "!literal-ignored.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	logs, err := Search(context.Background(), root, "log", FilenameSearch)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || filepath.Base(logs[0].Path) != "keep.log" {
		t.Fatalf("CRLF rules returned %#v, want keep.log only", logs)
	}
	literal, err := Search(context.Background(), root, "literal", FilenameSearch)
	if err != nil {
		t.Fatal(err)
	}
	if len(literal) != 1 || filepath.Base(literal[0].Path) != "!literal.txt" {
		t.Fatalf("escaped bang rule returned %#v, want literal.txt", literal)
	}
}

func TestSearchStreamCancelsDuringTraversal(t *testing.T) {
	root := t.TempDir()
	const fileCount = 1000
	for index := 0; index < fileCount; index++ {
		name := filepath.Join(root, "match-"+strconv.Itoa(index)+".txt")
		if err := os.WriteFile(name, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := SearchStream(ctx, root, "match", FilenameSearch, 10_000)
	finished := make(chan int, 1)
	go func() {
		count := 0
		for batch := range stream {
			count += len(batch.Results)
			if count > 0 {
				cancel()
			}
		}
		finished <- count
	}()
	select {
	case count := <-finished:
		if count == 0 {
			t.Fatal("cancellation test did not receive an initial result batch")
		}
		if count >= fileCount {
			t.Fatalf("cancellation arrived too late: received %d results", count)
		}
	case <-time.After(time.Second):
		t.Fatal("search stream did not close after cancellation")
	}
}
