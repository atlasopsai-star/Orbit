package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atlasopsai-star/Orbit/src/internal/common"
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

func (m *model) orbitActionContext() actions.Context {
	path, directory, isDir := m.selectedOrbitPath()
	apps := map[string]bool{}
	for _, name := range []string{"Cursor", "Visual Studio Code", "Zed", "Xcode", "Terminal", "iTerm2", "Ghostty", "Warp"} {
		apps[name] = orbitAppAvailable(name)
	}
	hasTrash := directory != "" && m.hasTrash && trash.Available(directory)
	return actions.Context{SelectedPath: path, CurrentDirectory: directory, IsDirectory: isDir, IsMac: runtime.GOOS == "darwin", HasTrash: hasTrash, AvailableApps: apps, GitStatus: m.gitStatus.Files[path]}
}

func (m *model) openOrbitActionPalette() tea.Cmd {
	m.actionPalette.SetActions(actions.Default(m.orbitActionContext()))
	m.actionPalette.SetDimensions(minOrbit(72, maxOrbit(8, m.fullWidth-2)), minOrbit(24, maxOrbit(8, m.fullHeight-2)))
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
	commands := map[string]string{
		"Cursor":             "cursor",
		"Visual Studio Code": "code",
		"Zed":                "zed",
		"Xcode":              "xed",
		"Terminal":           "",
		"iTerm2":             "",
		"Ghostty":            "",
		"Warp":               "",
	}
	if command := commands[name]; command != "" {
		if _, err := exec.LookPath(command); err == nil {
			return true
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	appNames := map[string]string{"iTerm2": "iTerm", "Terminal": "Terminal"}
	appName := appNames[name]
	if appName == "" {
		appName = name
	}
	for _, root := range []string{"/Applications", "/System/Applications", filepath.Join(home, "Applications")} {
		if _, err := os.Stat(filepath.Join(root, appName+".app")); err == nil {
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
		return orbitActionCommand("Open", "Opened "+filepath.Base(path), func() error {
			if runtime.GOOS == "darwin" && common.Config.PreferredEditor != "" {
				return runOrbitCommand("open", "-a", common.Config.PreferredEditor, path)
			}
			return orbitOpen(path)
		})
	case "reveal-finder":
		return orbitActionCommand("Finder", "Revealed "+filepath.Base(path)+" in Finder", func() error { return runOrbitCommand("open", "-R", path) })
	case "open-terminal":
		target := path
		if !isDir {
			target = directory
		}
		terminal, args := orbitTerminalCommand(target, common.Config.PreferredTerminal)
		return orbitActionCommand("Terminal", "Opened "+terminal+" at "+target, func() error { return runOrbitCommand(terminal, args...) })
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
		return m.folderSizeModal.Open(path, false)
	case "compress":
		return m.getCompressSelectedFilesCmd()
	case "extract":
		return m.getExtractFileCmd()
	case "git-diff":
		return m.openGitDiff(path)
	case "search-files":
		m.searchBarFocus()
	case "search-recursive":
		return m.searchModal.Open(directory, orbitfs.FilenameSearch)
	case "search-content":
		return m.searchModal.Open(directory, orbitfs.ContentSearch)
	case "lookup":
		return m.openLookup()
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

func orbitTerminalCommand(path, preferred string) (string, []string) {
	app := "Terminal"
	switch strings.ToLower(strings.TrimSpace(preferred)) {
	case "iterm2", "iterm":
		app = "iTerm"
	case "ghostty":
		app = "Ghostty"
	case "warp":
		app = "Warp"
	case "terminal", "":
	default:
		// Unknown preferences intentionally fall back to Terminal.app.
	}
	return "open", []string{"-a", app, path}
}

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
	m.invalidateGitStatus()
	return nil
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
