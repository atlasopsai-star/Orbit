package orbitfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyNoOverwriteFilesAndUnicodePaths(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source file é.txt")
	destination := filepath.Join(root, "copy file é.txt")
	if err := os.WriteFile(source, []byte("orbit"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyNoOverwrite(source, destination); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(destination)
	if err != nil || string(content) != "orbit" {
		t.Fatalf("copy = %q, %v", content, err)
	}
	if err := CopyNoOverwrite(source, destination); err == nil {
		t.Fatal("expected collision error")
	}
}

func TestCopyNoOverwriteDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "child"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyNoOverwrite(source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "child")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(destination, "destination") {
		t.Fatal("unexpected destination")
	}
}
