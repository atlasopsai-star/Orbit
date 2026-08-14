package filepanel

import (
	"github.com/atlasopsai-star/Orbit/src/internal/common"
)

func (m *Model) UpdateDimensions(width, height int) {
	m.SetWidth(width)
	m.SetHeight(height)
}

func (m *Model) SetWidth(width int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setWidthUnlocked(width)
}

func (m *Model) setWidthUnlocked(width int) {
	if width < MinWidth {
		width = MinWidth
	}
	m.width = width
	m.SearchBar.SetWidth(m.width - common.InnerPadding)
	m.columns = m.makeColumns(common.Config.FilePanelExtraColumns, common.Config.FilePanelNamePercent)
}

func (m *Model) SetHeight(height int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setHeightUnlocked(height)
}

func (m *Model) setHeightUnlocked(height int) {
	if height < MinHeight {
		height = MinHeight
	}
	m.height = height
	// Adjust scroll if needed
	m.scrollToCursorUnlocked(m.GetCursorUnlocked())
}

func (m *Model) GetWidth() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.width
}

func (m *Model) GetHeight() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.height
}

func (m *Model) GetMainPanelHeight() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mainPanelHeightUnlocked()
}

func (m *Model) mainPanelHeightUnlocked() int {
	return m.height - common.BorderPadding
}

func (m *Model) GetContentWidth() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.contentWidthUnlocked()
}

func (m *Model) contentWidthUnlocked() int {
	return m.width - common.BorderPadding
}

func (m *Model) NeedRenderHeaders() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.needRenderHeadersUnlocked()
}

func (m *Model) needRenderHeadersUnlocked() bool {
	return common.Config.FilePanelExtraColumns > 0 && len(m.columns) > 1
}

// PanelElementHeight calculates the number of visible elements in content area
func (m *Model) PanelElementHeight() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.panelElementHeightUnlocked()
}

func (m *Model) panelElementHeightUnlocked() int {
	headerHeight := 0
	if m.needRenderHeadersUnlocked() {
		headerHeight = ColumnHeaderHeight
	}
	return m.mainPanelHeightUnlocked() - contentPadding - headerHeight
}
