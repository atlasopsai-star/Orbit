package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectorySize(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "b"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := DirectorySize(context.Background(), root)
	if err != nil || got.Bytes != 8 || got.Files != 2 || got.Directories != 1 {
		t.Fatalf("size = %#v, %v", got, err)
	}
}

func TestDirectorySizeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DirectorySize(ctx, t.TempDir()); err == nil {
		t.Fatal("expected cancellation")
	}
}
