package orbitfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkSearchStream1K(b *testing.B)  { benchmarkSearchStream(b, 1000) }
func BenchmarkSearchStream10K(b *testing.B) { benchmarkSearchStream(b, 10000) }

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
