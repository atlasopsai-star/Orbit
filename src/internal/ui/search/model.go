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

type Model struct {
	input   textinput.Model
	root    string
	mode    orbitfs.SearchMode
	results []orbitfs.SearchResult
	request uint64
	cancel  context.CancelFunc
	cursor  int
	open    bool
	loading bool
	err     error
	width   int
	height  int
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
	m.cursor = 0
	m.request++
	m.loading = false
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
	m.cursor = 0
	m.loading = false
}
func (m *Model) IsOpen() bool { return m.open }
func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	m.input.SetWidth(max(20, m.width-10))
}
func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	switch key {
	case "esc":
		m.Close()
	case "up":
		if len(m.results) > 0 {
			m.cursor = (m.cursor + len(m.results) - 1) % len(m.results)
		}
	case "down":
		if len(m.results) > 0 {
			m.cursor = (m.cursor + 1) % len(m.results)
		}
	case "k":
		if m.input.Value() == "" && len(m.results) > 0 {
			m.cursor = (m.cursor + len(m.results) - 1) % len(m.results)
		}
	case "j":
		if m.input.Value() == "" && len(m.results) > 0 {
			m.cursor = (m.cursor + 1) % len(m.results)
		}
	}
	return nil
}
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
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
	if query == "" {
		m.loading = false
		return inputCmd
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.loading = true
	root, mode := m.root, m.mode
	searchCmd := func() tea.Msg {
		results, err := orbitfs.Search(ctx, root, query, mode)
		return ResultMsg{Request: request, Results: results, Err: err}
	}
	return tea.Batch(inputCmd, searchCmd)
}
func (m *Model) Apply(msg ResultMsg) {
	if !m.open || msg.Request != m.request {
		return
	}
	m.cancel = nil
	m.loading = false
	m.results = msg.Results
	m.cursor = 0
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
		lines = append(lines, "Searching…")
	} else if m.err != nil {
		lines = append(lines, "Search failed: "+m.err.Error())
	} else if len(m.results) == 0 {
		lines = append(lines, "No matches", "Type a filename or text query")
	} else {
		limit := max(3, m.height-8)
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
				lines = append(lines, fmt.Sprintf("%s%s:%d", prefix, relative, result.Line), "    "+result.Text)
			} else {
				lines = append(lines, prefix+relative)
			}
		}
		if len(m.results) > limit {
			lines = append(lines, fmt.Sprintf("  +%d more", len(m.results)-limit))
		}
	}
	lines = append(lines, "", "↑↓ navigate  Esc close")
	return lipgloss.NewStyle().Width(max(20, m.width-2)).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6EE7F9")).Render(strings.Join(lines, "\n"))
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
