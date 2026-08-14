package filemodel

import (
	"testing"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
)

func TestUpdatePreviewPanelRejectsOlderRequest(t *testing.T) {
	m := Model{ioReqCnt: 2}
	msg := preview.NewUpdateMsg("/tmp/notes.md", "stale content", "", 20, 10, 0)

	if cmd := m.UpdatePreviewPanel(msg); cmd != nil {
		t.Fatal("stale preview unexpectedly returned a command")
	}
	if m.ioReqCnt != 2 {
		t.Fatalf("stale preview changed request counter to %d", m.ioReqCnt)
	}
}
