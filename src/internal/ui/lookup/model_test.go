package lookup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

func testResults(count int) []orbitfs.LookupResult {
	results := make([]orbitfs.LookupResult, 0, count)
	for i := 0; i < count; i++ {
		results = append(results, orbitfs.LookupResult{
			Path:   filepath.Join("/tmp", "item-"+string(rune('a'+i))),
			Name:   "item-" + string(rune('a'+i)),
			IsDir:  false,
			Parent: "/tmp",
		})
	}
	return results
}

func TestSelectionClampsWithoutWrap(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.results = testResults(3)

	m.moveCursor(-1)
	if m.cursor != 0 {
		t.Fatalf("up at first result must stay, got %d", m.cursor)
	}
	m.moveCursor(1)
	m.moveCursor(1)
	if m.cursor != 2 {
		t.Fatalf("down twice should reach last, got %d", m.cursor)
	}
	m.moveCursor(1)
	if m.cursor != 2 {
		t.Fatalf("down at last result must stay, got %d", m.cursor)
	}
}

func TestArrowKeysMoveSelection(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.results = testResults(4)

	if cmd := m.HandleKey("down"); cmd == nil {
		t.Fatal("down with results should schedule a preview")
	}
	if m.cursor != 1 {
		t.Fatalf("down should select index 1, got %d", m.cursor)
	}
	if cmd := m.HandleKey("up"); cmd == nil {
		t.Fatal("up should schedule a preview")
	}
	if m.cursor != 0 {
		t.Fatalf("up should select index 0, got %d", m.cursor)
	}
}

func TestEnterEmitsNavigateMsg(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	m.results = testResults(1)
	m.cursor = 0

	cmd := m.HandleKey("enter")
	if cmd == nil {
		t.Fatal("enter with results must return a command")
	}
	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.Path != "/tmp/item-a" {
		t.Fatalf("wrong path %q", nav.Path)
	}
	if m.IsOpen() {
		t.Fatal("enter must close the modal")
	}
}

func TestEscClosesAndCancels(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	_, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	defer cancel()

	m.HandleKey("esc")
	if m.IsOpen() {
		t.Fatal("esc must close the modal")
	}
}

func TestQuickActionsEmitSelectedPath(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.results = testResults(2)
	m.cursor = 1

	cmd := m.HandleKey("F")
	if cmd == nil {
		t.Fatal("F should emit a quick action")
	}
	msg := cmd().(QuickActionMsg)
	if msg.Action != QuickFinder || msg.Path != m.results[1].Path {
		t.Fatalf("unexpected quick action %+v", msg)
	}

	cmd = m.HandleKey("C")
	msg = cmd().(QuickActionMsg)
	if msg.Action != QuickCopyPath {
		t.Fatalf("expected CopyPath action, got %v", msg.Action)
	}
}

func TestQueryChangeSupersedesOldRequests(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.roots = []string{t.TempDir()}

	// A new query bumps the request id and starts a fresh stream.
	m.input.SetValue("a")
	before := m.request
	if cmd := m.restartQuery(); cmd == nil {
		t.Fatal("restartQuery with a query should return a command")
	}
	if m.request != before+1 {
		t.Fatalf("request id should increment, got %d", m.request)
	}
	if !m.loading {
		t.Fatal("should be loading after typing")
	}

	// A stale batch from the previous request must be ignored.
	m.Update(batchMsg{Request: before, Batch: orbitfs.LookupBatch{Results: testResults(3)}})
	if len(m.results) != 0 {
		t.Fatal("stale batch must not be applied")
	}

	// Typing again supersedes the in-flight request.
	m.input.SetValue("ab")
	m.restartQuery()
	latest := m.request
	m.Update(batchMsg{Request: latest, Batch: orbitfs.LookupBatch{Results: testResults(2)}})
	if len(m.results) != 2 {
		t.Fatalf("current batch should apply, got %d results", len(m.results))
	}
}

func TestStalePreviewIgnored(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.previewReq = 7

	before := m.preview.GetContent()
	msg := preview.NewUpdateMsg("/tmp/x", "stale content", "", 40, 20, 6)
	m.Update(previewMsg{Req: 6, Msg: msg})
	if m.preview.GetContent() != before || m.preview.GetContent() == "stale content" {
		t.Fatal("stale preview must be dropped")
	}
}

func TestScopeCycling(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	before := m.scope
	m.HandleKey("tab")
	if m.scope != (before+1)%(orbitfs.ScopeEverywhere+1) {
		t.Fatalf("tab should cycle scope, got %v", m.scope)
	}
	m.HandleKey("tab")
	if m.scope == before {
		t.Fatal("scope should advance past the first cycle")
	}
}

func TestViewBoundedByDimensions(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.SetDimensions(80, 24)
	m.results = testResults(30)
	m.input.SetValue("item")

	view := m.View()
	lines := splitLines(view)
	if len(lines) > 24 {
		t.Fatalf("view taller than terminal: %d lines", len(lines))
	}
	for index, line := range lines {
		if width := ansi.StringWidth(line); width > 80 {
			t.Fatalf("line %d exceeds terminal width: %d > 80", index, width)
		}
	}

	m.SetDimensions(140, 40)
	view = m.View()
	lines = splitLines(view)
	if len(lines) > 40 {
		t.Fatalf("wide view taller than terminal: %d lines", len(lines))
	}
	for index, line := range lines {
		if width := ansi.StringWidth(line); width > 140 {
			t.Fatalf("wide line %d exceeds terminal width: %d > 140", index, width)
		}
	}
}

func TestPreviewRequestSupersedesOnSelectionChange(t *testing.T) {
	m := New()
	m.Open(t.TempDir(), nil)
	defer m.Close()
	m.results = testResults(3)

	m.cursor = 0
	first := m.schedulePreview()
	if first == nil {
		t.Fatal("file selection should schedule a preview")
	}
	firstReq := m.previewReq

	m.cursor = 1
	second := m.schedulePreview()
	if second == nil {
		t.Fatal("moving selection should schedule a new preview")
	}
	if m.previewReq <= firstReq {
		t.Fatal("preview request id must advance on selection change")
	}
}

func splitLines(s string) []string {
	lines := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
