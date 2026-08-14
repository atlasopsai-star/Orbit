package gitdiff

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ResultMsg struct {
	Request   uint64
	Text      string
	Truncated bool
	Err       error
}

type Model struct {
	root    string
	path    string
	lines   []string
	request uint64
	cancel  context.CancelFunc
	open    bool
	loading bool
	err     error
	cursor  int
	width   int
	height  int
}

func New() Model { return Model{width: 72, height: 20} }

func (m *Model) Open(root, path string) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.root = root
	m.path = path
	m.lines = nil
	m.request++
	request := m.request
	m.open = true
	m.loading = true
	m.err = nil
	m.cursor = 0
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return func() tea.Msg { return ResultMsg{Request: request, Err: err} }
	}
	return func() tea.Msg {
		command := exec.CommandContext(ctx, "git", "diff", "--no-ext-diff", "--no-color", "HEAD", "--", relative)
		command.Dir = root
		var output cappedBuffer
		command.Stdout = &output
		err := command.Run()
		return ResultMsg{Request: request, Text: output.String(), Truncated: output.truncated, Err: err}
	}
}

func (m *Model) Close() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.open = false
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
}
func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	switch key {
	case "esc":
		m.Close()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.lines)-1 {
			m.cursor++
		}
	}
	return nil
}
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
		return nil
	}
	result, ok := msg.(ResultMsg)
	if !ok || result.Request != m.request {
		return nil
	}
	m.loading = false
	m.cancel = nil
	m.err = result.Err
	if result.Err == nil {
		m.lines = strings.Split(strings.TrimSuffix(result.Text, "\n"), "\n")
		if result.Truncated {
			m.lines = append(m.lines, "[Diff truncated at 2 MiB]")
		}
		if len(m.lines) == 1 && m.lines[0] == "" {
			m.lines = []string{"No changes in this file."}
		}
	}
	return nil
}
func (m Model) View() string {
	if !m.open {
		return ""
	}
	lines := []string{"ORBIT GIT DIFF", m.path, ""}
	if m.loading {
		lines = append(lines, "Loading diff…")
	} else if m.err != nil {
		lines = append(lines, "Could not read diff: "+m.err.Error())
	} else {
		start := 0
		if m.cursor >= max(1, m.height-8) {
			start = m.cursor - max(1, m.height-8) + 1
		}
		for _, line := range m.lines[start:min(len(m.lines), start+max(1, m.height-8))] {
			lines = append(lines, styleDiffLine(line))
		}
	}
	lines = append(lines, "", "↑↓ scroll  Esc close")
	return lipgloss.NewStyle().Width(max(1, m.width-6)).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#C084FC")).Render(strings.Join(lines, "\n"))
}

type cappedBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	const limit = 2 * 1024 * 1024
	remaining := limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(data), nil
	}
	if len(data) > remaining {
		_, _ = b.Buffer.Write(data[:remaining])
		b.truncated = true
		return len(data), nil
	}
	return b.Buffer.Write(data)
}

func styleDiffLine(line string) string {
	if strings.HasPrefix(line, "+") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7EE787")).Render(line)
	}
	if strings.HasPrefix(line, "-") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B81")).Render(line)
	}
	return line
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
