package search

import (
	"strings"
	"testing"

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
func TestSearchModalIgnoresStaleResults(t *testing.T) {
	m := New()
	m.Open("/tmp/Orbit", orbitfs.FilenameSearch)
	m.request = 2
	m.Apply(ResultMsg{Request: 1, Results: []orbitfs.SearchResult{{Path: "/tmp/stale"}}})
	if len(m.results) != 0 {
		t.Fatal("stale results applied")
	}
	m.Apply(ResultMsg{Request: 2, Results: []orbitfs.SearchResult{{Path: "/tmp/fresh"}}})
	if len(m.results) != 1 {
		t.Fatal("fresh results missing")
	}
}
func TestSearchModalRendersContent(t *testing.T) {
	m := New()
	m.Open("/tmp/Orbit", orbitfs.ContentSearch)
	m.Apply(ResultMsg{Request: m.request, Results: []orbitfs.SearchResult{{Path: "/tmp/Orbit/src/auth.go", Line: 42, Text: "authentication"}}})
	view := m.View()
	if !strings.Contains(view, "auth.go:42") || !strings.Contains(view, "authentication") {
		t.Fatalf("view = %q", view)
	}
}
