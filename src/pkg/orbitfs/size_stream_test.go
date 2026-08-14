package orbitfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectorySizeStreamEmitsFinalCounts(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}
	var last SizeProgress
	batches := 0
	for progress := range DirectorySizeStream(context.Background(), root) {
		batches++
		last = progress
	}
	if batches == 0 || !last.Done || last.Bytes != 3 || last.Files != 1 || last.Directories != 1 {
		t.Fatalf("batches=%d last=%#v", batches, last)
	}
}
func TestDirectorySizeStreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for progress := range DirectorySizeStream(ctx, t.TempDir()) {
		if progress.Err != nil {
			t.Fatal(progress.Err)
		}
	}
}
