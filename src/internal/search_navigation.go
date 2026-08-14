package internal

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
	searchui "github.com/atlasopsai-star/Orbit/src/internal/ui/search"
)

func (m *model) navigateToSearchResult(msg searchui.SelectedMsg) tea.Cmd {
	panel := m.getFocusedFilePanel()
	// TargetFile is a panel-local name, not an absolute path. Set it before
	// changing directories so the existing refresh path can focus the result.
	panel.TargetFile = filepath.Base(msg.Path)
	if err := m.updateCurrentFilePanelDir(filepath.Dir(msg.Path)); err != nil {
		m.notifyModel = notify.New(true, "Search result", fmt.Sprintf("Could not open %s: %v", filepath.Base(msg.Path), err), notify.NoAction)
		return nil
	}
	// The existing panel target mechanism selects the file; preview line jumping remains best-effort.
	return m.fileModel.GetFilePreviewCmd(true)
}
