package search

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

type ResultMsg struct {
	Request uint64
	Results []orbitfs.SearchResult
	Err     error
}
type SelectedMsg struct {
	Path string
	Line int
}
type searchStartedMsg struct {
	Request uint64
	Stream  <-chan orbitfs.SearchBatch
}
type searchBatchMsg struct {
	Request uint64
	Batch   orbitfs.SearchBatch
}

type Model struct {
	input     textinput.Model
	root      string
	mode      orbitfs.SearchMode
	results   []orbitfs.SearchResult
	stream    <-chan orbitfs.SearchBatch
	request   uint64
	cancel    context.CancelFunc
	cursor    int
	open      bool
	loading   bool
	truncated bool
	err       error
	width     int
	height    int
}

func New() Model {
	input := textinput.New()
	input.SetWidth(48)
	return Model{input: input, width: 78, height: 20}
}
func (m *Model) Open(root string, mode orbitfs.SearchMode) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.root = root
	m.mode = mode
	m.results = nil
	m.stream = nil
	m.cursor = 0
	m.request++
	m.loading = false
	m.truncated = false
	m.err = nil
	m.open = true
	m.input.SetValue("")
	return m.input.Focus()
}
func (m *Model) Close() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.open = false
	m.input.Blur()
	m.input.SetValue("")
	m.results = nil
	m.stream = nil
	m.cursor = 0
	m.loading = false
	m.truncated = false
}
func (m *Model) IsOpen() bool { return m.open }
func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	m.input.SetWidth(max(8, m.width-10))
}

func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	switch key {
	case "esc":
		m.Close()
	case "up":
		m.moveCursor(-1)
	case "down":
		m.moveCursor(1)
	case "k":
		if m.input.Value() == "" {
			m.moveCursor(-1)
		}
	case "j":
		if m.input.Value() == "" {
			m.moveCursor(1)
		}
	case "enter":
		if len(m.results) == 0 {
			return nil
		}
		result := m.results[min(m.cursor, len(m.results)-1)]
		m.Close()
		return func() tea.Msg { return SelectedMsg{Path: result.Path, Line: result.Line} }
	}
	return nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.results) == 0 {
		return
	}
	m.cursor = (m.cursor + delta + len(m.results)) % len(m.results)
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
		return nil
	}
	switch msg := msg.(type) {
	case searchStartedMsg:
		if msg.Request != m.request {
			return nil
		}
		m.stream = msg.Stream
		return waitSearchBatch(msg.Request, msg.Stream)
	case searchBatchMsg:
		if msg.Request != m.request {
			return nil
		}
		m.results = append(m.results, msg.Batch.Results...)
		if len(m.results) > orbitfs.MaxSearchResults {
			m.results = m.results[:orbitfs.MaxSearchResults]
		}
		m.truncated = m.truncated || msg.Batch.Truncated
		m.err = msg.Batch.Err
		m.loading = !msg.Batch.Done && msg.Batch.Err == nil
		if msg.Batch.Done {
			m.loading = false
			m.cancel = nil
			m.stream = nil
			return nil
		}
		return waitSearchBatch(msg.Request, m.stream)
	case ResultMsg:
		if msg.Request == m.request {
			m.results = append([]orbitfs.SearchResult(nil), msg.Results...)
			m.err = msg.Err
			m.loading = false
		}
		return nil
	}
	before := m.input.Value()
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	query := strings.TrimSpace(m.input.Value())
	if query == before {
		return inputCmd
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.request++
	request := m.request
	m.err = nil
	m.results = nil
	m.truncated = false
	if query == "" {
		m.loading = false
		return inputCmd
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.loading = true
	root, mode := m.root, m.mode
	stream := orbitfs.SearchStream(ctx, root, query, mode, orbitfs.MaxSearchResults)
	start := func() tea.Msg { return searchStartedMsg{Request: request, Stream: stream} }
	return tea.Batch(inputCmd, start)
}

func waitSearchBatch(request uint64, stream <-chan orbitfs.SearchBatch) tea.Cmd {
	return func() tea.Msg {
		batch, ok := <-stream
		if !ok {
			return searchBatchMsg{Request: request, Batch: orbitfs.SearchBatch{Done: true}}
		}
		return searchBatchMsg{Request: request, Batch: batch}
	}
}
func (m *Model) Apply(msg ResultMsg) {
	if !m.open || msg.Request != m.request {
		return
	}
	m.results = append([]orbitfs.SearchResult(nil), msg.Results...)
	m.cursor = 0
	m.loading = false
	m.err = msg.Err
}

func (m Model) View() string {
	if !m.open {
		return ""
	}
	mode := "FILENAMES"
	if m.mode == orbitfs.ContentSearch {
		mode = "CONTENT"
	}
	lines := []string{"ORBIT SEARCH  •  " + mode, "Root  " + m.root, "> " + m.input.View(), ""}
	if m.loading {
		lines = append(lines, fmt.Sprintf("Searching…  %d matches", len(m.results)))
	} else if m.err != nil {
		lines = append(lines, "Search failed: "+m.err.Error())
	} else if len(m.results) == 0 {
		lines = append(lines, "No matches", "Type a filename or text query")
	} else {
		limit := max(1, m.height-8)
		compact := m.width < 64 || m.height < 16
		for index, result := range m.results[:min(len(m.results), limit)] {
			prefix := "  "
			if index == m.cursor {
				prefix = "› "
			}
			relative, err := filepath.Rel(m.root, result.Path)
			if err != nil {
				relative = result.Path
			}
			if m.mode == orbitfs.ContentSearch {
				lines = append(lines, fmt.Sprintf("%s%s:%d", prefix, relative, result.Line))
				if !compact {
					lines = append(lines, "    "+result.Text)
				}
			} else {
				lines = append(lines, prefix+relative)
			}
		}
		if m.truncated {
			lines = append(lines, fmt.Sprintf("  %d+ matches — refine your query", orbitfs.MaxSearchResults))
		} else if len(m.results) > limit {
			lines = append(lines, fmt.Sprintf("  +%d more", len(m.results)-limit))
		}
	}
	footer := "↑↓ navigate  Enter select  Esc close"
	if m.width < 64 || m.height < 16 {
		footer = "Enter select  Esc close"
	}
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(max(1, m.width-6)).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6EE7F9")).Render(strings.Join(lines, "\n"))
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
