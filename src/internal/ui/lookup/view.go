package lookup

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

type listEntry struct {
	header string
	index  int
}

func (m Model) View() string {
	if !m.open {
		return ""
	}
	borderColor := lipgloss.Color(common.Theme.ModalBorderActive)
	inner := max(20, m.width-2)

	title := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render("ORBIT LOOKUP")
	scope := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint)).Render("  •  " + m.scope.String())

	lines := make([]string, 0, m.height-2)
	lines = append(lines, lipgloss.JoinHorizontal(0, title, scope))
	lines = append(lines, "> "+m.input.View())
	lines = append(lines, m.statusLine())
	lines = append(lines, "")

	switch {
	case m.expanded:
		lines = append(lines, m.previewPaneLines()...)
	case m.compact:
		lines = append(lines, m.resultsLines()...)
	default:
		lines = append(lines, m.splitPaneLines()...)
	}

	lines = append(lines, "")
	lines = append(lines, m.footerLine())

	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}
	lines = lines[:m.height-2]

	padded := make([]string, 0, len(lines))
	for _, line := range lines {
		padded = append(padded, clipLine(line, inner))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(strings.Join(padded, "\n"))
}

func (m Model) statusLine() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelTopPath))
	switch {
	case m.loading:
		return dim.Render(fmt.Sprintf("Searching…  %d matches", len(m.results)))
	case m.err != nil:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Error)).Render("Search failed: " + m.err.Error())
	case m.truncated:
		return dim.Render(fmt.Sprintf("%d+ matches — keep typing to narrow", orbitfs.MaxLookupResults))
	case len(m.results) == 0:
		if strings.TrimSpace(m.input.Value()) == "" {
			return dim.Render("Type a file or folder name to search this Mac")
		}
		return dim.Render("No matches for “" + strings.TrimSpace(m.input.Value()) + "”")
	default:
		dirs, files := 0, 0
		for _, result := range m.results {
			if result.IsDir {
				dirs++
			} else {
				files++
			}
		}
		return dim.Render(fmt.Sprintf("%d folders · %d files", dirs, files))
	}
}

func (m Model) resultsLines() []string {
	if len(m.results) == 0 {
		return []string{lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelTopPath)).Render("  No results yet")}
	}
	selFG := lipgloss.Color(common.Theme.FilePanelItemSelectedFG)
	selBG := lipgloss.Color(common.Theme.FilePanelItemSelectedBG)
	contentWidth := max(10, m.width-4)
	lines := make([]string, 0, m.resultsBudget())
	for _, entry := range m.visibleEntries() {
		if entry.header != "" {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint)).Bold(true).Render("  "+entry.header))
			continue
		}
		result := m.results[entry.index]
		selected := entry.index == m.cursor
		mark := "  "
		if selected {
			mark = "▸ "
		}
		name := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelFG)).Bold(true).Render(result.Name)
		if selected {
			name = lipgloss.NewStyle().Foreground(selFG).Bold(true).Render(result.Name)
		}
		tag := m.typeTag(result)
		row := mark + name
		pad := contentWidth - lipgloss.Width(row) - lipgloss.Width(tag) - 2
		if pad < 1 {
			pad = 1
		}
		row = row + strings.Repeat(" ", pad) + tag
		if selected {
			row = lipgloss.NewStyle().Background(selBG).Render(row)
		}
		lines = append(lines, row)
		if !m.compact {
			parent := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelTopPath)).Render("   " + result.Parent)
			lines = append(lines, parent)
		}
	}
	if len(lines) > m.resultsBudget() {
		lines = lines[:m.resultsBudget()]
	}
	return lines
}

func (m Model) typeTag(result orbitfs.LookupResult) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.DirectoryIconColor))
	label := strings.ToUpper(result.Ext)
	if result.IsDir {
		label = "FOLDER"
	} else if label == "" {
		label = "FILE"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelTopPath))
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.ModalBorderActive))
	}
	return style.Render(label)
}

func (m Model) visibleEntries() []listEntry {
	entries := make([]listEntry, 0, len(m.results)+2)
	dirs, files := false, false
	for index, result := range m.results {
		if result.IsDir && !dirs {
			entries = append(entries, listEntry{header: "FOLDERS"})
			dirs = true
		}
		if !result.IsDir && !files {
			entries = append(entries, listEntry{header: "FILES"})
			files = true
		}
		entries = append(entries, listEntry{index: index})
	}
	start := min(m.scroll, len(entries))
	end := min(start+m.rowsVisible(), len(entries))
	return entries[start:end]
}

func (m Model) splitPaneLines() []string {
	left := m.resultsLines()
	right := m.previewPaneLines()
	leftWidth := max(10, m.width-2-m.previewPaneWidth()-2)
	rows := make([]string, 0, len(left))
	for i := 0; i < len(left); i++ {
		rightLine := ""
		if i < len(right) {
			rightLine = right[i]
		}
		rows = append(rows, padLine(left[i], leftWidth)+"  "+rightLine)
	}
	return rows
}

func (m Model) previewPaneLines() []string {
	budget := m.resultsBudget()
	lines := make([]string, 0, budget)
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.ModalBorderActive)).Bold(true).Render(" PREVIEW"))
	body := strings.Split(m.previewContent(), "\n")
	for i := 0; i < budget-1; i++ {
		if i < len(body) {
			lines = append(lines, " "+ansi.Truncate(body[i], m.previewPaneWidth()-3, ""))
		} else {
			lines = append(lines, "")
		}
	}
	return lines
}

func (m Model) previewContent() string {
	if len(m.results) == 0 {
		return ""
	}
	result := m.results[min(m.cursor, len(m.results)-1)]
	if result.IsDir {
		return m.folderView
	}
	if m.preview.IsLoading() {
		return "Loading preview…"
	}
	content := m.preview.GetContent()
	if content == "" {
		return "Preview unavailable\n\n" + result.Name + "\n" + result.Parent
	}
	return content
}

func (m Model) footerLine() string {
	keys := "↑↓ Select  Enter Jump  Tab Scope"
	if m.compact {
		keys += "  Space Preview"
	}
	keys += "  F Finder  T Terminal  E Editor  C Copy  Esc Close"
	return lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelTopPath)).Render(keys)
}

// resultsBudget is how many terminal rows the results area gets.
func (m Model) resultsBudget() int { return max(1, m.height-7) }

// rowsVisible is the scroll window size in entry units.
func (m Model) rowsVisible() int {
	budget := m.resultsBudget()
	if m.compact {
		return budget
	}
	return max(1, budget/2)
}

func padLine(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func clipLine(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}
