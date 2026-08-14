package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
