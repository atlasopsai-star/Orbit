package internal

import (
	"context"
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/gitstatus"
)

type gitStatusMsg struct {
	request  uint64
	location string
	snapshot gitstatus.Snapshot
	err      error
}

func (m *model) refreshGitStatusCmd() tea.Cmd {
	location := m.getFocusedFilePanel().Location
	if location == "" || location == m.gitLocation && (m.gitLoading || m.gitChecked) {
		return nil
	}
	if m.gitCancel != nil {
		m.gitCancel()
	}
	m.gitLocation = location
	m.gitRequest++
	request := m.gitRequest
	m.gitChecked = false
	m.gitLoading = true
	ctx, cancel := context.WithCancel(context.Background())
	m.gitCancel = cancel
	return func() tea.Msg {
		snapshot, err := gitstatus.Detect(ctx, location)
		return gitStatusMsg{request: request, location: location, snapshot: snapshot, err: err}
	}
}

func (m *model) invalidateGitStatus() {
	m.gitChecked = false
	m.gitStatus = gitstatus.Snapshot{}
}

func (m *model) applyGitStatus(msg gitStatusMsg) {
	if msg.request != m.gitRequest || msg.location != m.gitLocation {
		return
	}
	m.gitLoading = false
	m.gitChecked = true
	m.gitCancel = nil
	if msg.err != nil {
		m.gitStatus = gitstatus.Snapshot{}
		m.fileModel.SetGitStatus("", "", nil)
		slog.Debug("Git status unavailable", "location", msg.location, "error", msg.err)
		return
	}
	m.gitStatus = msg.snapshot
	m.fileModel.SetGitStatus(msg.snapshot.Root, msg.snapshot.Branch, msg.snapshot.Files)
}
