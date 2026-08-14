package filepanel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/atlasopsai-star/Orbit/src/config/icon"
	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/rendering"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/sortmodel"
)

/*
- TODO: Write File Panel Specific unit test
  - Individual panel resizes
  - Footer content of filepanel changes due to resizing
  - i Only mode icons remains on smaller
  - ii Other things that change too
  - Other panels like clipboard and metadata's content changes too on resize
*/
func (m *Model) Render(focused bool) string {
	r := ui.FilePanelRenderer(m.height, m.width, focused)

	m.renderTopBar(r)
	m.renderSearchBar(r)
	m.renderFooter(r, m.SelectedCount())
	if m.NeedRenderHeaders() {
		m.renderColumnHeaders(r)
	}
	m.renderFileEntries(r)
	return r.Render()
}

func (m *Model) renderTopBar(r *rendering.Renderer) {
	// Keep repository context compact and only show it when the async Git
	// snapshot confirms that this panel is inside a repository.
	label := m.Location
	if m.GitBranch != "" {
		label += "  ⎇ " + m.GitBranch
	}
	truncatedPath := common.TruncateTextBeginning(label, m.GetContentWidth()-common.InnerPadding, "...")
	r.AddLines(common.FilePanelTopDirectoryIcon + common.FilePanelTopPathStyle.Render(truncatedPath))
	r.AddSection()
}

func (m *Model) renderSearchBar(r *rendering.Renderer) {
	r.AddLines(" " + m.SearchBar.View())
}

// TODO : Unit test this
func (m *Model) renderFooter(r *rendering.Renderer, selectedCount uint) {
	sortLabel, sortIcon := m.getSortInfo()
	modeLabel, modeIcon := m.getPanelModeInfo(selectedCount)
	cursorStr := m.getCursorString()

	if common.Config.Nerdfont {
		sortLabel = sortIcon + icon.Space + sortLabel
		modeLabel = modeIcon + icon.Space + modeLabel
	} else {
		// TODO : Figure out if we can set icon.Space to " " if nerdfont is false
		// That would simplify code
		sortLabel = sortIcon + " " + sortLabel
	}

	if common.Config.ShowPanelFooterInfo {
		hints := ""
		actionsKey := firstHotkey(common.Hotkeys.OpenCommandPalette, "ctrl+k")
		searchKey := firstHotkey(common.Hotkeys.SearchBar, "/")
		previewKey := firstHotkey(common.Hotkeys.ToggleFilePreviewPanel, "f")
		helpKey := firstHotkey(common.Hotkeys.OpenHelpMenu, "?")
		switch {
		case m.width >= 100:
			hints = fmt.Sprintf("%s Actions  %s Search  %s Preview  %s Help", actionsKey, searchKey, previewKey, helpKey)
		case m.width >= 70:
			hints = fmt.Sprintf("%s Actions  %s Search  %s Help", actionsKey, searchKey, helpKey)
		case m.width >= 48:
			hints = fmt.Sprintf("%s Actions  %s Search", actionsKey, searchKey)
		}
		if hints != "" {
			r.SetBorderInfoItems(sortLabel, modeLabel, cursorStr, hints)
		} else {
			r.SetBorderInfoItems(sortLabel, modeLabel, cursorStr)
		}
		if r.AreInfoItemsTruncated() {
			r.SetBorderInfoItems(sortIcon, modeIcon, cursorStr)
		}
	} else {
		r.SetBorderInfoItems(cursorStr)
	}
}

func firstHotkey(hotkeys []string, fallback string) string {
	if len(hotkeys) == 0 || hotkeys[0] == "" {
		return fallback
	}
	return hotkeys[0]
}

func (m *Model) renderColumnHeaders(r *rendering.Renderer) {
	var builder strings.Builder
	for _, column := range m.columns {
		builder.WriteString(column.RenderHeader())
	}
	r.AddLines(builder.String())
}

func (m *Model) renderFileEntries(r *rendering.Renderer) {
	if m.Empty() {
		r.AddLines(common.FilePanelNoneText)
		return
	}
	end := min(m.renderIndex+m.PanelElementHeight(), m.ElemCount())

	for itemIndex := m.renderIndex; itemIndex < end; itemIndex++ {
		if m.Renaming && itemIndex == m.GetCursor() {
			r.AddLines(m.Rename.View())
			continue
		}
		var builder strings.Builder
		for _, column := range m.columns {
			colData := column.Render(itemIndex)
			builder.WriteString(colData)
		}
		r.AddLines(builder.String())
	}
}

func (m *Model) getSortInfo() (string, string) {
	iconStr := icon.SortAsc
	if m.SortReversed {
		iconStr = icon.SortDesc
	}
	return sortmodel.SortOptionsShortStr[m.SortKind], iconStr
}

func (m *Model) getPanelModeInfo(selectedCount uint) (string, string) {
	switch m.PanelMode {
	case BrowserMode:
		return "Browser", icon.Browser
	case SelectMode:
		return "Select" + icon.Space + fmt.Sprintf("(%d)", selectedCount), icon.Select
	default:
		return "", ""
	}
}

func (m *Model) getCursorString() string {
	cursor := m.GetCursor()
	if !m.Empty() {
		cursor++ // Convert to 1-based
	}
	return fmt.Sprintf("%d/%d", cursor, m.ElemCount())
}

func (m *Model) renderSelectBox(isSelected bool) string {
	if !common.Config.ShowSelectIcons || !common.Config.Nerdfont || m.PanelMode != SelectMode {
		return ""
	}

	if m.IsFocused {
		if isSelected {
			return common.CheckboxCheckedFocused
		}
		return common.CheckboxEmptyFocused
	}
	if isSelected {
		return common.CheckboxChecked
	}
	return common.CheckboxEmpty
}

// Checks whether a panel needs re-render due to being invalid or due to directory change
func (m *Model) NeedsReRender() bool {
	if !m.EmptyOrInvalid() {
		return filepath.Dir(m.GetFirstElement().Location) != m.Location
	}
	return true
}
