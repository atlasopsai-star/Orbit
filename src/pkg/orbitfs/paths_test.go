package orbitfs

import (
	"path/filepath"
	"testing"
)

func TestCopyRelativePath(t *testing.T) {
	got, err := CopyRelativePath(filepath.Join("/tmp", "Orbit", "src", "main.go"), filepath.Join("/tmp", "Orbit"))
	if err != nil || got != "src/main.go" {
		t.Fatalf("relative path = %q, %v", got, err)
	}
}

func TestDuplicateName(t *testing.T) {
	existing := map[string]bool{}
	exists := func(path string) bool { return existing[path] }
	dir := t.TempDir()
	first := DuplicateName(dir, "photo.png", exists)
	if filepath.Base(first) != "photo copy.png" {
		t.Fatalf("first = %q", first)
	}
	existing[first] = true
	second := DuplicateName(dir, "photo.png", exists)
	if filepath.Base(second) != "photo copy 2.png" {
		t.Fatalf("second = %q", second)
	}
}
