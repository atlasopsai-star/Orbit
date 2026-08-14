package filemodel

import "charm.land/lipgloss/v2"

func (m *Model) Render() string {
	f := make([]string, m.PanelCount()+1)
	for i := range m.FilePanels {
		// RenderCurrent keeps the focus flag and all panel fields inside the
		// panel's own critical section; a range-copy here would read the panel
		// struct outside the lock and race with concurrent panel mutations.
		f[i] = m.FilePanels[i].RenderCurrent()
	}
	f[m.PanelCount()] = m.GetFilePreviewRender()
	return lipgloss.JoinHorizontal(lipgloss.Top, f...)
}

func (m *Model) GetFilePreviewRender() string {
	if !m.FilePreview.IsOpen() {
		return ""
	}
	// Check if width and height have been synced yet
	if m.FilePreview.GetContentHeight() == m.Height &&
		m.FilePreview.GetContentWidth() == m.ExpectedPreviewWidth {
		if m.FilePreview.IsLoading() {
			return m.FilePreview.RenderText(FilePreviewLoadingText)
		}
		return m.FilePreview.GetContent()
	}

	// Placeholder resizing text till they get synced
	return m.FilePreview.RenderTextWithDimension(
		FilePreviewResizingText, m.Height, m.ExpectedPreviewWidth)
}
