package foldersize

import (
	"errors"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

func TestOpenUsesSessionCacheUnlessRefreshed(t *testing.T) {
	m := New()
	at := time.Now().Add(-time.Second)
	m.cache["/tmp/example"] = cacheEntry{Result: orbitfs.SizeResult{Bytes: 42, Files: 2, Directories: 1}, At: at}

	if cmd := m.Open("/tmp/example", false); cmd != nil {
		t.Fatal("cache hit started a scan")
	}
	if m.loading || m.current.Bytes != 42 || !m.cachedAt.Equal(at) {
		t.Fatalf("cache state = %#v loading=%v cachedAt=%v", m.current, m.loading, m.cachedAt)
	}

	if cmd := m.Open("/tmp/example", true); cmd == nil {
		t.Fatal("refresh did not start a scan")
	}
	if !m.loading || m.request != 2 {
		t.Fatalf("refresh state loading=%v request=%d", m.loading, m.request)
	}
}

func TestInvalidateEvictsOverlappingCachedPaths(t *testing.T) {
	m := New()
	at := time.Now()
	m.cache["/tmp/project"] = cacheEntry{At: at}
	m.cache["/tmp/project/src"] = cacheEntry{At: at}
	m.cache["/tmp/project/src/deeper"] = cacheEntry{At: at}
	m.cache["/tmp/project/docs"] = cacheEntry{At: at}
	m.cache["/tmp/projects"] = cacheEntry{At: at}

	if cmd := m.Invalidate("/tmp/project/src/new.go"); cmd != nil {
		t.Fatal("invalidate unexpectedly started a scan")
	}

	for _, path := range []string{"/tmp/project", "/tmp/project/src"} {
		if _, ok := m.cache[path]; ok {
			t.Fatalf("cache entry %q was not invalidated", path)
		}
	}
	if _, ok := m.cache["/tmp/project/src/deeper"]; !ok {
		t.Fatal("unrelated descendant cache entry was invalidated")
	}
	m.Invalidate("/tmp/project/src")
	if _, ok := m.cache["/tmp/project/src/deeper"]; ok {
		t.Fatal("descendant cache entry was not invalidated")
	}
	for _, path := range []string{"/tmp/project/docs", "/tmp/projects"} {
		if _, ok := m.cache[path]; !ok {
			t.Fatalf("unrelated cache entry %q was invalidated", path)
		}
	}
}

func TestFolderSizeFitsNarrowDimensions(t *testing.T) {
	m := New()
	m.SetDimensions(20, 10)
	m.Open("/tmp/example", false)
	if width := lipgloss.Width(m.View()); width > 20 {
		t.Fatalf("folder size width=%d", width)
	}
	m.Close()
}

func TestScanErrorDoesNotCacheZeroResult(t *testing.T) {
	m := New()
	m.Open("/tmp/example", true)
	request := m.request
	m.Update(progressMsg{Request: request, Progress: orbitfs.SizeProgress{Err: errors.New("stopped")}})
	if m.loading {
		t.Fatal("scan remained loading after error")
	}
	if _, ok := m.cache["/tmp/example"]; ok {
		t.Fatal("error result was cached")
	}
}
