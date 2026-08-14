package orbitfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// --- True streaming benchmarks -------------------------------------------------
//
// These measure the streaming contract directly instead of the collect-all
// wrapper: how fast the first result arrives, how long a full traversal takes,
// and how quickly cancellation stops the stream.

const (
	streamBenchFiles        = 50_000
	streamBenchDirectories  = 500
	streamBenchFirstResults = 10_000
)

// buildStreamBenchTree creates a tree with streamBenchFiles files spread over
// streamBenchDirectories directories. Every 10th file matches the query so the
// first match appears early in the traversal.
func buildStreamBenchTree(b *testing.B) string {
	b.Helper()
	root := b.TempDir()
	entriesPerDirectory := (streamBenchFiles + streamBenchDirectories - 1) / streamBenchDirectories
	for index := 0; index < streamBenchFiles; index++ {
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
	return root
}

// firstStreamResult drains the stream until the first batch with results
// arrives and returns the elapsed time.
func firstStreamResult(root, query string) (time.Duration, int, error) {
	start := time.Now()
	for batch := range SearchStream(context.Background(), root, query, FilenameSearch, streamBenchFirstResults) {
		if batch.Err != nil {
			return time.Since(start), 0, batch.Err
		}
		if len(batch.Results) > 0 {
			return time.Since(start), len(batch.Results), nil
		}
	}
	return time.Since(start), 0, fmt.Errorf("no results")
}

// BenchmarkSearchStreamTimeToFirstResult measures how long it takes for the
// first result batch to arrive while a 50k-file traversal is still running.
func BenchmarkSearchStreamTimeToFirstResult(b *testing.B) {
	root := buildStreamBenchTree(b)
	b.ResetTimer()
	var elapsed time.Duration
	for range b.N {
		d, count, err := firstStreamResult(root, "match")
		if err != nil || count == 0 {
			b.Fatalf("count=%d err=%v", count, err)
		}
		elapsed += d
	}
	b.ReportMetric(float64(elapsed)/float64(b.N)/float64(time.Millisecond), "ms-to-first-result")
}

// BenchmarkSearchStreamTotalTraversal measures the full traversal wall time.
func BenchmarkSearchStreamTotalTraversal(b *testing.B) {
	root := buildStreamBenchTree(b)
	b.ResetTimer()
	var elapsed time.Duration
	for range b.N {
		start := time.Now()
		count := 0
		for batch := range SearchStream(context.Background(), root, "match", FilenameSearch, streamBenchFirstResults) {
			if batch.Err != nil {
				b.Fatal(batch.Err)
			}
			count += len(batch.Results)
		}
		if count == 0 {
			b.Fatal("no results")
		}
		elapsed += time.Since(start)
	}
	b.ReportMetric(float64(elapsed)/float64(b.N)/float64(time.Millisecond), "ms-total-traversal")
}

// BenchmarkSearchStreamCancelLatency measures how quickly the stream stops
// after the context is cancelled mid-traversal. Cancellation must be
// near-instant regardless of how many files remain.
func BenchmarkSearchStreamCancelLatency(b *testing.B) {
	root := buildStreamBenchTree(b)
	b.ResetTimer()
	var elapsed time.Duration
	for range b.N {
		ctx, cancel := context.WithCancel(context.Background())
		start := time.Now()
		for batch := range SearchStream(ctx, root, "match", FilenameSearch, streamBenchFirstResults) {
			if len(batch.Results) > 0 {
				cancel()
			}
		}
		// Cover the degenerate case where the traversal ends before any match
		// (keeps the context released and vet quiet without deferring inside
		// the benchmark loop, which would accumulate b.N deferred calls).
		cancel()
		elapsed += time.Since(start)
	}
	b.ReportMetric(float64(elapsed)/float64(b.N)/float64(time.Microsecond), "us-cancel-latency")
}
