package search

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

func TestSearchModalOpenClose(t *testing.T) {
	m := New()
	m.Open("/tmp/Orbit", orbitfs.FilenameSearch)
	if !m.IsOpen() {
		t.Fatal("not open")
	}
	m.Close()
	if m.IsOpen() {
		t.Fatal("not closed")
	}
}
func TestSearchModalStreamsBatchesAndSelects(t *testing.T) {
	m := New()
	m.Open("/tmp/Orbit", orbitfs.FilenameSearch)
	m.request = 2
	m.Update(searchBatchMsg{Request: 1, Batch: orbitfs.SearchBatch{Results: []orbitfs.SearchResult{{Path: "/tmp/stale"}}}})
	if len(m.results) != 0 {
		t.Fatal("stale batch applied")
	}
	m.Update(searchBatchMsg{Request: 2, Batch: orbitfs.SearchBatch{Results: []orbitfs.SearchResult{{Path: "/tmp/fresh"}}, Done: true}})
	cmd := m.HandleKey("enter")
	if cmd == nil || m.IsOpen() {
		t.Fatal("enter did not select")
	}
	if msg := cmd().(SelectedMsg); msg.Path != "/tmp/fresh" {
		t.Fatalf("selected=%#v", msg)
	}
}
func TestSearchModalFitsNarrowDimensions(t *testing.T) {
	m := New()
	m.SetDimensions(20, 10)
	m.Open("/tmp/Orbit", orbitfs.FilenameSearch)
	if width := lipgloss.Width(m.View()); width > 20 {
		t.Fatalf("search width=%d", width)
	}
}

func TestSearchModalRendersContentAndTruncation(t *testing.T) {
	m := New()
	m.Open("/tmp/Orbit", orbitfs.ContentSearch)
	m.Apply(ResultMsg{Request: m.request, Results: []orbitfs.SearchResult{{Path: "/tmp/Orbit/src/auth.go", Line: 42, Text: "authentication"}}})
	m.truncated = true
	view := m.View()
	if !strings.Contains(view, "auth.go:42") || !strings.Contains(view, "authentication") || !strings.Contains(view, "200+ matches") {
		t.Fatalf("view=%q", view)
	}
}
