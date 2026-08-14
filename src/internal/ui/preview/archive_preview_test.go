package preview

import (
	"archive/tar"
	"archive/zip"
	"encoding/binary"
	"fmt"
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

// TestZipEntryCountReadsEndOfCentralDirectory verifies that the entry count is
// read from the EOCD tail without parsing the central directory, including the
// zip64 path where the classic EOCD carries the 0xffff sentinel.
func TestZipEntryCountReadsEndOfCentralDirectory(t *testing.T) {
	// Build a small real zip and confirm the count matches the writer's.
	path := filepath.Join(t.TempDir(), "real.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for _, name := range []string{"a.txt", "b.txt", "c/d.txt"} {
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
	if count, err := zipEntryCount(path); err != nil || count != 3 {
		t.Fatalf("zipEntryCount = %d, %v; want 3", count, err)
	}

	// Craft a zip64-style tail: a 56-byte zip64 EOCD record declaring 70000
	// entries, a 20-byte locator, then a classic EOCD with 0xffff sentinels.
	// No central directory is present, which would make zip.OpenReader fail;
	// zipEntryCount must still report the declared count from the records.
	bomb := filepath.Join(t.TempDir(), "bomb.zip")
	buf := make([]byte, 0, 98)
	eocd64 := make([]byte, 56)
	binary.LittleEndian.PutUint32(eocd64[0:], 0x06064b50)
	binary.LittleEndian.PutUint64(eocd64[32:], 70000)
	buf = append(buf, eocd64...)
	locator := make([]byte, 20)
	binary.LittleEndian.PutUint32(locator[0:], 0x07064b50)
	binary.LittleEndian.PutUint64(locator[8:], 0) // zip64 EOCD offset
	buf = append(buf, locator...)
	eocd := make([]byte, 22)
	binary.LittleEndian.PutUint32(eocd[0:], 0x06054b50)
	binary.LittleEndian.PutUint16(eocd[8:], 0xffff)  // entries on this disk
	binary.LittleEndian.PutUint16(eocd[10:], 0xffff) // total entries
	binary.LittleEndian.PutUint32(eocd[16:], 0xffffffff)
	buf = append(buf, eocd...)
	if err := os.WriteFile(bomb, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	if count, err := zipEntryCount(bomb); err != nil || count != 70000 {
		t.Fatalf("zipEntryCount(zip64) = %d, %v; want 70000", count, err)
	}
}

// TestArchivePreviewCapsHugeEntryCounts verifies that a zip declaring more
// entries than maxArchivePreviewEntries renders the real count from the EOCD
// and skips the per-entry listing instead of allocating a header per entry.
func TestArchivePreviewCapsHugeEntryCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "many.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for index := 0; index < maxArchivePreviewEntries+5; index++ {
		entry, createErr := archive.Create(fmt.Sprintf("entry-%05d.txt", index))
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte("x")); writeErr != nil {
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
	want := fmt.Sprintf("%d entries", maxArchivePreviewEntries+5)
	if !strings.Contains(render, want) {
		t.Fatalf("capped preview should report the real count %q: %q", want, render)
	}
	if !strings.Contains(render, "listing capped") {
		t.Fatalf("capped preview should mention the cap: %q", render)
	}
	if strings.Contains(render, "entry-00000.txt") {
		t.Fatalf("capped preview must not list entries: %q", render)
	}
}
