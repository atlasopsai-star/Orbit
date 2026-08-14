package lookup

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

func waitBatch(request uint64, stream <-chan orbitfs.LookupBatch) tea.Cmd {
	return func() tea.Msg {
		batch, ok := <-stream
		if !ok {
			return batchMsg{Request: request, Batch: orbitfs.LookupBatch{Done: true}}
		}
		return batchMsg{Request: request, Batch: batch}
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
		return nil
	}
	switch msg := msg.(type) {
	case startedMsg:
		if msg.Request != m.request {
			return nil
		}
		m.stream = msg.Stream
		return waitBatch(msg.Request, msg.Stream)
	case batchMsg:
		if msg.Request != m.request {
			return nil
		}
		m.results = msg.Batch.Results
		if m.cursor > len(m.results)-1 {
			m.cursor = max(0, len(m.results)-1)
		}
		m.keepCursorVisible()
		m.truncated = msg.Batch.Truncated
		m.err = msg.Batch.Err
		m.loading = !msg.Batch.Done && msg.Batch.Err == nil
		if msg.Batch.Done {
			m.loading = false
			m.cancel = nil
			m.stream = nil
			return m.schedulePreview()
		}
		return waitBatch(msg.Request, m.stream)
	case previewMsg:
		if msg.Req == m.previewReq {
			m.preview.Apply(msg.Msg)
		}
		return nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok && m.controlKey(key.String()) {
		return nil
	}
	before := m.input.Value()
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	query := strings.TrimSpace(m.input.Value())
	if query == before {
		return inputCmd
	}
	return tea.Batch(inputCmd, m.restartQuery())
}

// restartQuery cancels any in-flight search and starts a fresh one for the
// current query. Old results can never overwrite newer ones because every
// started/batch message carries the request id and is dropped when stale.
func (m *Model) restartQuery() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.request++
	request := m.request
	m.results = nil
	m.cursor = 0
	m.scroll = 0
	m.truncated = false
	m.err = nil
	m.folderView = ""
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		m.loading = false
		m.stream = nil
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.loading = true
	roots := append([]string(nil), m.roots...)
	stream := orbitfs.LookupStream(ctx, query, roots, orbitfs.LookupOptions{UseSpotlight: true, Limit: orbitfs.MaxLookupResults})
	return func() tea.Msg { return startedMsg{Request: request, Stream: stream} }
}
