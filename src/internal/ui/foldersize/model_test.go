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
