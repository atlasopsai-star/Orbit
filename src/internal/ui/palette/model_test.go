package palette

import (
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/actions"
)

func testActions() []actions.Action {
	return []actions.Action{{ID: "finder", Label: "Reveal in Finder", Description: "Show in Finder", Category: "Mac"}, {ID: "size", Label: "Calculate Folder Size", Description: "Scan folder", Category: "Utility"}}
}

func TestPaletteFiltersAndRenders(t *testing.T) {
	m := New()
	m.SetActions(testActions())
	m.Open()
	m.SetQuery("finder")
	items := m.Filtered()
	if len(items) != 1 || items[0].ID != "finder" {
		t.Fatalf("filtered = %#v", items)
	}
	if !strings.Contains(m.View(), "Reveal in Finder") {
		t.Fatalf("view = %q", m.View())
	}
}

func TestPaletteSelectionAndClose(t *testing.T) {
	m := New()
	m.SetActions(testActions())
	m.Open()
	cmd := m.HandleKey("enter")
	if cmd == nil || m.IsOpen() {
		t.Fatal("enter did not select/close")
	}
	msg := cmd().(ActionSelectedMsg)
	if msg.ID != "finder" {
		t.Fatalf("selected = %#v", msg)
	}
}

func TestPaletteFitsNarrowDimensions(t *testing.T) {
	m := New()
	m.SetActions(testActions())
	m.SetDimensions(20, 10)
	m.Open()
	if width := lipgloss.Width(m.View()); width > 20 {
		t.Fatalf("palette width=%d", width)
	}
}

func TestPaletteEscAndUpdate(t *testing.T) {
	m := New()
	m.SetActions(testActions())
	m.Open()
	m.HandleKey("esc")
	if m.IsOpen() {
		t.Fatal("escape did not close")
	}
	m.Open()
	_ = m.Update(tea.KeyPressMsg{Code: 'x'})
}
