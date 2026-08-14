package palette

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/actions"
)

type ActionSelectedMsg struct{ ID string }

type Model struct {
	input   textinput.Model
	actions []actions.Action
	cursor  int
	query   string
	open    bool
	width   int
	height  int
}

func New() Model {
	input := textinput.New()
	input.SetWidth(42)
	return Model{input: input, width: 72, height: 18}
}

func (m *Model) SetActions(items []actions.Action) {
	m.actions = append([]actions.Action(nil), items...)
	m.cursor = 0
}
func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	m.input.SetWidth(max(8, m.width-8))
}
func (m *Model) Open() tea.Cmd {
	m.open = true
	m.cursor = 0
	m.query = ""
	m.input.SetValue("")
	return m.input.Focus()
}
func (m *Model) SetQuery(query string) {
	m.query = query
	m.input.SetValue(query)
	m.cursor = 0
}

func (m *Model) Close()                     { m.open = false; m.input.Blur(); m.input.SetValue(""); m.cursor = 0 }
func (m *Model) IsOpen() bool               { return m.open }
func (m *Model) Filtered() []actions.Action { return actions.Filter(m.actions, m.query) }

func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	items := m.Filtered()
	switch key {
	case "up":
		if len(items) > 0 {
			m.cursor = (m.cursor + len(items) - 1) % len(items)
		}
	case "down":
		if len(items) > 0 {
			m.cursor = (m.cursor + 1) % len(items)
		}
	case "k":
		if m.input.Value() == "" && len(items) > 0 {
			m.cursor = (m.cursor + len(items) - 1) % len(items)
		}
	case "j":
		if m.input.Value() == "" && len(items) > 0 {
			m.cursor = (m.cursor + 1) % len(items)
		}
	case "enter", "right":
		if len(items) == 0 {
			return nil
		}
		id := items[min(m.cursor, len(items)-1)].ID
		m.Close()
		return func() tea.Msg { return ActionSelectedMsg{ID: id} }
	case "esc":
		m.Close()
	}
	return nil
}
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.query = m.input.Value()
	items := m.Filtered()
	if len(items) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	return cmd
}

func (m Model) View() string {
	if !m.open {
		return ""
	}
	items := m.Filtered()
	maxRows := max(1, m.height-7)
	compact := m.width < 64 || m.height < 16
	lines := []string{"ORBIT ACTIONS", "> " + m.input.View(), ""}
	if len(items) == 0 {
		lines = append(lines, "No matching actions", "Try: finder, cursor, path, size")
	} else {
		for index, item := range items[:min(len(items), maxRows)] {
			prefix := "  "
			if index == m.cursor {
				prefix = "› "
			}
			if compact {
				lines = append(lines, prefix+item.Label)
				continue
			}
			lines = append(lines, prefix+item.Label+"  "+lipgloss.NewStyle().Faint(true).Render(item.Category), "    "+item.Description)
		}
		if len(items) > maxRows {
			lines = append(lines, fmt.Sprintf("  +%d more", len(items)-maxRows))
		}
	}
	footer := "↑↓ navigate  Enter run  Esc close"
	if compact {
		footer = "Enter run  Esc close"
	}
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(max(1, m.width-6)).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#F6C453")).Render(strings.Join(lines, "\n"))
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
