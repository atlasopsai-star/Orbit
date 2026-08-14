package preview

import (
	"archive/tar"
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchivePreviewListsZipEntriesWithoutExtracting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for _, name := range []string{"README.md", "src/main.go"} {
		entry, createErr := archive.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte("orbit")); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	m := New()
	render, _ := m.RenderWithPath(path, 40, 8, 40)
	for _, name := range []string{"ZIP", "2 entries", "README.md", "src/main.go"} {
		if !strings.Contains(render, name) {
			t.Fatalf("archive preview does not contain %q: %q", name, render)
		}
	}
}

func TestArchivePreviewListsTarEntriesAndHandlesCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.tar")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := tar.NewWriter(file)
	if err := archive.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o644, Size: int64(len("orbit"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte("orbit")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	m := New()
	render, _ := m.RenderWithPath(path, 40, 8, 40)
	for _, name := range []string{"TAR", "1 entries", "README.md"} {
		if !strings.Contains(render, name) {
			t.Fatalf("archive preview does not contain %q: %q", name, render)
		}
	}

	corrupt := filepath.Join(t.TempDir(), "broken.zip")
	if err := os.WriteFile(corrupt, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	render, _ = m.RenderWithPath(corrupt, 40, 8, 40)
	if !strings.Contains(render, "Archive preview unavailable") {
		t.Fatalf("corrupt archive did not produce a useful error: %q", render)
	}
}
