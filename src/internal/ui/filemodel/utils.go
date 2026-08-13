package filemodel

import (
	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/filepanel"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
)

func (m *Model) GetFocusedFilePanel() *filepanel.Model {
	return &m.FilePanels[m.FocusedPanelIndex]
}

func New(firstPanelPaths []string, toggleDotFile bool) Model {
	return Model{
		FilePanels:       filepanel.FilePanelSlice(firstPanelPaths),
		FilePreview:      preview.New(),
		SinglePanelWidth: common.DefaultFilePanelWidth,
		DisplayDotFiles:  toggleDotFile,
	}
}
