package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/atlasopsai-star/Orbit/src/internal/trash"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
	"github.com/atlasopsai-star/Orbit/src/pkg/actions"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

type orbitActionResultMsg struct {
	title   string
	message string
	err     error
}
type orbitFolderSizeMsg struct {
	path    string
	request uint64
	result  orbitfs.SizeResult
	err     error
}

func (m *model) orbitActionContext() actions.Context {
	path, directory, isDir := m.selectedOrbitPath()
	apps := map[string]bool{}
	for _, name := range []string{"Cursor", "Visual Studio Code", "Zed", "Xcode"} {
		apps[name] = orbitAppAvailable(name)
	}
	hasTrash := directory != "" && m.hasTrash && trash.Available(directory)
	return actions.Context{SelectedPath: path, CurrentDirectory: directory, IsDirectory: isDir, IsMac: runtime.GOOS == "darwin", HasTrash: hasTrash, AvailableApps: apps}
}

func (m *model) openOrbitActionPalette() tea.Cmd {
	m.actionPalette.SetActions(actions.Default(m.orbitActionContext()))
	m.actionPalette.SetDimensions(minOrbit(72, maxOrbit(20, m.fullWidth-4)), minOrbit(24, maxOrbit(10, m.fullHeight-2)))
	return m.actionPalette.Open()
}

func (m *model) selectedOrbitPath() (string, string, bool) {
	panel := m.getFocusedFilePanel()
	if panel.Empty() {
		return "", panel.Location, false
	}
	path := panel.GetFocusedItem().Location
	info, err := os.Stat(path)
	return path, panel.Location, err == nil && info.IsDir()
}

func orbitAppAvailable(name string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	commands := map[string]string{"Cursor": "cursor", "Visual Studio Code": "code", "Zed": "zed", "Xcode": "xed"}
	if command := commands[name]; command != "" {
		if _, err := exec.LookPath(command); err == nil {
			return true
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
		if _, err := os.Stat(filepath.Join(root, name+".app")); err == nil {
			return true
		}
	}
	return false
}

func (m *model) executeOrbitAction(id string) tea.Cmd {
	path, directory, isDir := m.selectedOrbitPath()
	if id != "search-files" && path == "" {
		return nil
	}
	switch id {
	case "open":
		return orbitActionCommand("Open", "Opened "+filepath.Base(path), func() error { return orbitOpen(path) })
	case "reveal-finder":
		return orbitActionCommand("Finder", "Revealed "+filepath.Base(path)+" in Finder", func() error { return runOrbitCommand("open", "-R", path) })
	case "open-terminal":
		target := path
		if !isDir {
			target = directory
		}
		return orbitActionCommand("Terminal", "Opened Terminal at "+target, func() error { return runOrbitCommand("open", "-a", "Terminal", target) })
	case "open-cursor":
		return orbitActionCommand("Cursor", "Opened "+filepath.Base(path)+" in Cursor", func() error { return runOrbitCommand("open", "-a", "Cursor", path) })
	case "open-vscode":
		return orbitActionCommand("VS Code", "Opened "+filepath.Base(path)+" in VS Code", func() error { return runOrbitCommand("open", "-a", "Visual Studio Code", path) })
	case "open-zed":
		return orbitActionCommand("Zed", "Opened "+filepath.Base(path)+" in Zed", func() error { return runOrbitCommand("open", "-a", "Zed", path) })
	case "open-xcode":
		return orbitActionCommand("Xcode", "Opened "+filepath.Base(path)+" in Xcode", func() error { return runOrbitCommand("open", "-a", "Xcode", path) })
	case "copy-absolute-path":
		return orbitActionCommand("Path copied", path, func() error { return m.clipboardWriter(path) })
	case "copy-relative-path":
		relative, err := orbitfs.CopyRelativePath(path, directory)
		if err != nil {
			return orbitActionResult(err)
		}
		return orbitActionCommand("Relative path copied", relative, func() error { return m.clipboardWriter(relative) })
	case "move-to-trash":
		return m.getDeleteTriggerCmd(false)
	case "duplicate":
		return func() tea.Msg {
			destination := orbitfs.DuplicatePath(path, orbitPathExists)
			if orbitPathExists(destination) {
				destination = orbitfs.DuplicatePath(path, orbitPathExists)
			}
			err := orbitfs.CopyNoOverwrite(path, destination)
			return orbitActionResultMsg{title: "Duplicate", message: "Created " + filepath.Base(destination), err: err}
		}
	case "folder-size":
		if m.orbitSizeCancel != nil {
			m.orbitSizeCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.orbitSizeCancel = cancel
		m.orbitSizeRequest++
		request := m.orbitSizeRequest
		return func() tea.Msg {
			result, err := orbitfs.DirectorySize(ctx, path)
			return orbitFolderSizeMsg{path: path, request: request, result: result, err: err}
		}
	case "compress":
		return m.getCompressSelectedFilesCmd()
	case "extract":
		return m.getExtractFileCmd()
	case "search-files":
		m.searchBarFocus()
	case "search-recursive":
		return m.searchModal.Open(directory, orbitfs.FilenameSearch)
	case "search-content":
		return m.searchModal.Open(directory, orbitfs.ContentSearch)
	}
	return nil
}

func orbitActionCommand(title, message string, run func() error) tea.Cmd {
	return func() tea.Msg { return orbitActionResultMsg{title: title, message: message, err: run()} }
}
func orbitActionResult(err error) tea.Cmd {
	return orbitActionCommand("Orbit action", "", func() error { return err })
}
func orbitPathExists(path string) bool                  { _, err := os.Lstat(path); return err == nil }
func runOrbitCommand(name string, args ...string) error { return exec.Command(name, args...).Run() }
func orbitOpen(path string) error {
	if runtime.GOOS == "darwin" {
		return runOrbitCommand("open", path)
	}
	return runOrbitCommand("xdg-open", path)
}

func (m *model) applyOrbitActionResult(msg orbitActionResultMsg) tea.Cmd {
	if msg.err != nil {
		m.notifyModel = notify.New(true, msg.title, fmt.Sprintf("Could not complete action: %v", msg.err), notify.NoAction)
		return nil
	}
	m.notifyModel = notify.New(true, msg.title, msg.message, notify.NoAction)
	return nil
}

func (m *model) applyOrbitFolderSize(msg orbitFolderSizeMsg) tea.Cmd {
	if msg.request != m.orbitSizeRequest {
		return nil
	}
	m.orbitSizeCancel = nil
	if msg.err != nil {
		m.notifyModel = notify.New(true, "Folder size", fmt.Sprintf("Could not scan %s: %v", filepath.Base(msg.path), msg.err), notify.NoAction)
		return nil
	}
	m.notifyModel = notify.New(true, "Folder size", fmt.Sprintf("%s  %s\n%d files  %d folders", filepath.Base(msg.path), formatOrbitBytes(msg.result.Bytes), msg.result.Files, msg.result.Directories), notify.NoAction)
	return nil
}

func formatOrbitBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", size, units[unit])
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}
func maxOrbit(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minOrbit(a, b int) int {
	if a < b {
		return a
	}
	return b
}
