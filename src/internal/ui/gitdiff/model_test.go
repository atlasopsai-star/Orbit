package gitdiff

import (
	"strings"
	"testing"
)

func TestCappedBufferBoundsDiffMemory(t *testing.T) {
	buffer := cappedBuffer{}
	data := make([]byte, 3*1024*1024)
	if _, err := buffer.Write(data); err != nil {
		t.Fatal(err)
	}
	if buffer.Len() != 2*1024*1024 || !buffer.truncated {
		t.Fatalf("len=%d truncated=%v", buffer.Len(), buffer.truncated)
	}
}

func TestDiffModalAppliesResultAndScrolls(t *testing.T) {
	m := New()
	m.Open("/tmp/repo", "/tmp/repo/file.go")
	request := m.request
	m.Update(ResultMsg{Request: request - 1, Text: "stale"})
	if m.lines != nil {
		t.Fatal("stale diff applied")
	}
	m.Update(ResultMsg{Request: request, Text: "@@\n-old\n+new\n"})
	if m.loading || len(m.lines) != 3 || !strings.Contains(m.View(), "new") {
		t.Fatalf("state loading=%v lines=%#v view=%q", m.loading, m.lines, m.View())
	}
	m.HandleKey("down")
	m.HandleKey("up")
	m.HandleKey("esc")
	if m.IsOpen() {
		t.Fatal("escape did not close diff")
	}
}
