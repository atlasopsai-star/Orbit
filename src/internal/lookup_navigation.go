package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	tea "charm.land/bubbletea/v2"

	"github.com/atlasopsai-star/Orbit/src/internal/common"
	lookupui "github.com/atlasopsai-star/Orbit/src/internal/ui/lookup"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
	"github.com/atlasopsai-star/Orbit/src/pkg/actions"
)

// openLookup opens the Orbit Lookup modal with the focused panel's directory
// as the CURRENT-scope anchor.
func (m *model) openLookup() tea.Cmd {
	m.lookupModal.SetDimensions(minOrbit(160, maxOrbit(40, m.fullWidth-2)), minOrbit(44, maxOrbit(10, m.fullHeight-2)))
	return m.lookupModal.Open(m.getFocusedFilePanel().Location, common.Config.LookupRoots)
}

// navigateToLookupResult jumps the focused panel to a Lookup result. Folders
// open directly; files open their parent directory with the file focused and
// its preview loaded. The user stays inside Orbit — nothing external launches.
func (m *model) navigateToLookupResult(msg lookupui.NavigateMsg) tea.Cmd {
	info, err := os.Stat(msg.Path)
	if err != nil {
		m.notifyModel = notify.New(true, "Lookup", fmt.Sprintf("Could not open %s: %v", filepath.Base(msg.Path), err), notify.NoAction)
		return nil
	}
	if info.IsDir() {
		if err := m.updateCurrentFilePanelDir(msg.Path); err != nil {
			m.notifyModel = notify.New(true, "Lookup", fmt.Sprintf("Could not open folder %s: %v", filepath.Base(msg.Path), err), notify.NoAction)
			return nil
		}
		return m.fileModel.GetFilePreviewCmd(true)
	}
	// TargetFile is a panel-local name. Set it before changing directories so
	// the existing refresh path focuses the result file.
	m.getFocusedFilePanel().TargetFile = filepath.Base(msg.Path)
	if err := m.updateCurrentFilePanelDir(filepath.Dir(msg.Path)); err != nil {
		m.notifyModel = notify.New(true, "Lookup", fmt.Sprintf("Could not open %s: %v", filepath.Base(msg.Path), err), notify.NoAction)
		return nil
	}
	return m.fileModel.GetFilePreviewCmd(true)
}

// handleLookupQuickAction executes a Lookup quick action on the selected
// result, reusing the existing Orbit action implementations.
func (m *model) handleLookupQuickAction(msg lookupui.QuickActionMsg) tea.Cmd {
	switch msg.Action {
	case lookupui.QuickFinder:
		return orbitActionCommand("Finder", "Revealed "+filepath.Base(msg.Path)+" in Finder", func() error { return runOrbitCommand("open", "-R", msg.Path) })
	case lookupui.QuickTerminal:
		target := msg.Path
		if info, err := os.Stat(msg.Path); err == nil && !info.IsDir() {
			target = filepath.Dir(msg.Path)
		}
		terminal, args := orbitTerminalCommand(target, common.Config.PreferredTerminal)
		return orbitActionCommand("Terminal", "Opened "+terminal+" at "+target, func() error { return runOrbitCommand(terminal, args...) })
	case lookupui.QuickEditor:
		return orbitActionCommand("Open", "Opened "+filepath.Base(msg.Path), func() error {
			if runtime.GOOS == "darwin" && common.Config.PreferredEditor != "" {
				return runOrbitCommand("open", "-a", common.Config.PreferredEditor, msg.Path)
			}
			return orbitOpen(msg.Path)
		})
	case lookupui.QuickCopyPath:
		return orbitActionCommand("Path copied", msg.Path, func() error { return m.clipboardWriter(msg.Path) })
	case lookupui.QuickPalette:
		info, err := os.Stat(msg.Path)
		isDir := err == nil && info.IsDir()
		parent := msg.Path
		if !isDir {
			parent = filepath.Dir(msg.Path)
		}
		ctx := m.orbitActionContext()
		ctx.SelectedPath = msg.Path
		ctx.CurrentDirectory = parent
		ctx.IsDirectory = isDir
		m.actionPalette.SetActions(actions.Default(ctx))
		m.actionPalette.SetDimensions(minOrbit(72, maxOrbit(8, m.fullWidth-2)), minOrbit(24, maxOrbit(8, m.fullHeight-2)))
		return m.actionPalette.Open()
	}
	return nil
}
