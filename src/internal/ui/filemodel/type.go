package filemodel

import (
	"github.com/atlasopsai-star/Orbit/src/internal/ui/filepanel"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
)

// TODO: Make the fields unexported, as much as possible
// some fields like `Width` should not be updated directly, only via
// Set functions. Having them exported is dangerous
type Model struct {
	FilePanels           []filepanel.Model
	SinglePanelWidth     int
	Width                int
	ExpectedPreviewWidth int
	Height               int
	Renaming             bool
	MaxFilePanel         int
	FilePreview          preview.Model
	FocusedPanelIndex    int
	ioReqCnt             int
	DisplayDotFiles      bool
}
