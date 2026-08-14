package lookup

import (
	"context"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

// NavigateMsg asks the parent model to jump to a Lookup result.
type NavigateMsg struct{ Path string }

// QuickAction names a Lookup quick action executed by the parent model.
type QuickAction int

const (
	QuickFinder QuickAction = iota
	QuickTerminal
	QuickEditor
	QuickCopyPath
	QuickPalette
)

// QuickActionMsg carries a quick action for the currently selected result.
type QuickActionMsg struct {
	Action QuickAction
	Path   string
}

type startedMsg struct {
	Request uint64
	Stream  <-chan orbitfs.LookupBatch
}

type batchMsg struct {
	Request uint64
	Batch   orbitfs.LookupBatch
}

type previewMsg struct {
	Req uint64
	Msg preview.UpdateMsg
}

// Model is the Orbit Lookup modal: a global file & folder search with live
// previews, fully keyboard navigable, always inside the terminal.
type Model struct {
	input      textinput.Model
	scope      orbitfs.LookupScope
	cwd        string
	extraRoots []string
	roots      []string
	results    []orbitfs.LookupResult
	stream     <-chan orbitfs.LookupBatch
	request    uint64
	cancel     context.CancelFunc
	cursor     int
	scroll     int
	open       bool
	loading    bool
	truncated  bool
	err        error
	width      int
	height     int

	// Live preview of the selected result.
	preview    preview.Model
	previewReq uint64
	folderView string
	expanded   bool
	compact    bool
}

func New() Model {
	input := textinput.New()
	input.SetWidth(52)
	return Model{input: input, scope: orbitfs.ScopeEverywhere, width: 110, height: 34}
}

// Open resets the modal and focuses the search input. extraRoots are the
// configured lookup_roots that join the home directory for EVERYWHERE.
func (m *Model) Open(cwd string, extraRoots []string) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.cwd = cwd
	m.extraRoots = extraRoots
	m.roots = orbitfs.ResolveLookupRoots(m.scope, cwd, extraRoots)
	m.results = nil
	m.stream = nil
	m.request++
	m.cursor = 0
	m.scroll = 0
	m.loading = false
	m.truncated = false
	m.err = nil
	m.folderView = ""
	m.expanded = false
	m.open = true
	m.preview.SetOpen(true)
	m.preview.SetEmptyWithDimensions(m.previewPaneWidth(), m.previewPaneHeight())
	m.input.SetValue("")
	return m.input.Focus()
}

func (m *Model) Close() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.open = false
	m.preview.SetOpen(false)
	m.input.Blur()
	m.input.SetValue("")
	m.results = nil
	m.stream = nil
	m.cursor = 0
	m.scroll = 0
	m.loading = false
	m.truncated = false
	m.err = nil
	m.folderView = ""
}

func (m *Model) IsOpen() bool { return m.open }

func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
	m.input.SetWidth(max(8, m.width-12))
	m.compact = m.width < 100 || m.height < 26
}

// HandleKey handles control keys while the modal is open. Regular characters
// fall through to the text input via Update.
func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	switch key {
	case "esc", "ctrl+f":
		m.Close()
	case "up":
		return m.moveCursor(-1)
	case "down":
		return m.moveCursor(1)
	case "k":
		if m.input.Value() == "" {
			return m.moveCursor(-1)
		}
	case "j":
		if m.input.Value() == "" {
			return m.moveCursor(1)
		}
	case "enter":
		if len(m.results) == 0 {
			return nil
		}
		result := m.results[min(m.cursor, len(m.results)-1)]
		m.Close()
		return func() tea.Msg { return NavigateMsg{Path: result.Path} }
	case "tab":
		m.scope = (m.scope + 1) % (orbitfs.ScopeEverywhere + 1)
		m.roots = orbitfs.ResolveLookupRoots(m.scope, m.cwd, m.extraRoots)
		return m.restartQuery()
	case " ":
		if m.input.Value() == "" && m.compact && len(m.results) > 0 {
			m.expanded = !m.expanded
			m.preview.SetEmptyWithDimensions(m.previewPaneWidth(), m.previewPaneHeight())
			return m.schedulePreview()
		}
	case "F":
		return m.quickAction(QuickFinder)
	case "T":
		return m.quickAction(QuickTerminal)
	case "E":
		return m.quickAction(QuickEditor)
	case "C":
		return m.quickAction(QuickCopyPath)
	case "ctrl+k":
		return m.quickAction(QuickPalette)
	}
	return nil
}

func (m *Model) quickAction(action QuickAction) tea.Cmd {
	if len(m.results) == 0 {
		return nil
	}
	result := m.results[min(m.cursor, len(m.results)-1)]
	return func() tea.Msg { return QuickActionMsg{Action: action, Path: result.Path} }
}

// moveCursor moves the selection by delta, clamping at the first and last
// result (no wrap-around), and keeps the cursor inside the visible window.
func (m *Model) moveCursor(delta int) tea.Cmd {
	if len(m.results) == 0 {
		return nil
	}
	target := m.cursor + delta
	if target < 0 {
		target = 0
	}
	if target >= len(m.results) {
		target = len(m.results) - 1
	}
	m.cursor = target
	m.keepCursorVisible()
	return m.schedulePreview()
}

// keepCursorVisible advances the scroll window so the selected entry stays on
// screen as the selection moves beyond the visible area.
func (m *Model) keepCursorVisible() {
	if len(m.results) == 0 {
		return
	}
	position := m.entryPosition(m.cursor)
	rows := m.rowsVisible()
	if position < m.scroll {
		m.scroll = position
	}
	if position >= m.scroll+rows {
		m.scroll = position - rows + 1
	}
}

// entryPosition returns the position of the result index in entry space
// (group headers plus one slot per result).
func (m *Model) entryPosition(index int) int {
	position := 0
	for i := 0; i < len(m.results) && i <= index; i++ {
		if i == 0 || m.results[i].IsDir != m.results[i-1].IsDir {
			position++
		}
		position++
	}
	return position
}

// controlKey reports whether the key is claimed by the modal rather than
// typed into the search input.
func (m *Model) controlKey(key string) bool {
	switch key {
	case "esc", "up", "down", "enter", "tab", "F", "T", "E", "C", "ctrl+k", "ctrl+f":
		return true
	case " ", "k", "j":
		return m.input.Value() == ""
	}
	return false
}
