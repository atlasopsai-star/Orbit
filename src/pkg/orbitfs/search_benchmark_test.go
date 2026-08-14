package orbitfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkSearchStream1K(b *testing.B)   { benchmarkSearchStream(b, 1000) }
func BenchmarkSearchStream10K(b *testing.B)  { benchmarkSearchStream(b, 10000) }
func BenchmarkSearchStream100K(b *testing.B) { benchmarkSearchStreamTree(b, 100000, 1000) }

func benchmarkSearchStream(b *testing.B, entries int) {
	root := b.TempDir()
	for index := 0; index < entries; index++ {
		path := filepath.Join(root, fmt.Sprintf("entry-%05d.txt", index))
		if index%10 == 0 {
			path = filepath.Join(root, fmt.Sprintf("match-%05d.txt", index))
		}
		if err := os.WriteFile(path, []byte("orbit benchmark"), 0644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for range b.N {
		results, err := Search(context.Background(), root, "match", FilenameSearch)
		if err != nil || len(results) == 0 {
			b.Fatalf("results=%d err=%v", len(results), err)
		}
	}
}

func benchmarkSearchStreamTree(b *testing.B, entries, directories int) {
	root := b.TempDir()
	entriesPerDirectory := (entries + directories - 1) / directories
	for index := 0; index < entries; index++ {
		directory := filepath.Join(root, fmt.Sprintf("dir-%04d", index/entriesPerDirectory))
		if err := os.MkdirAll(directory, 0755); err != nil {
			b.Fatal(err)
		}
		name := fmt.Sprintf("entry-%05d.txt", index)
		if index%10 == 0 {
			name = fmt.Sprintf("match-%05d.txt", index)
		}
		if err := os.WriteFile(filepath.Join(directory, name), []byte("orbit benchmark"), 0644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for range b.N {
		results, err := Search(context.Background(), root, "match", FilenameSearch)
		if err != nil || len(results) == 0 {
			b.Fatalf("results=%d err=%v", len(results), err)
		}
	}
}
