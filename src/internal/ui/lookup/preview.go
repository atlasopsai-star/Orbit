package lookup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/preview"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

// schedulePreview renders the currently selected result into the preview pane.
// Rapid selection changes supersede older renders: each request gets a unique
// id and stale results are dropped by the request guard in Update, so Orbit
// never pays for five competing previews on a quick arrow-key run.
func (m *Model) schedulePreview() tea.Cmd {
	if !m.open || len(m.results) == 0 {
		return nil
	}
	result := m.results[min(m.cursor, len(m.results)-1)]
	if result.IsDir {
		if m.preview.GetLocation() == result.Path && m.folderView != "" {
			return nil
		}
		m.preview.SetLocation(result.Path)
		m.folderView = buildFolderView(result)
		return nil
	}
	if m.preview.GetLocation() == result.Path && !m.preview.IsLoading() && m.preview.GetContent() != "" {
		return nil
	}
	m.preview.SetLocation(result.Path)
	m.preview.SetLoading()
	m.previewReq++
	req := m.previewReq
	path := result.Path
	width := m.previewPaneWidth()
	height := m.previewPaneHeight()
	return func() tea.Msg {
		content, raw := m.preview.RenderWithPath(path, width, height, width)
		return previewMsg{Req: req, Msg: preview.NewUpdateMsg(path, content, raw, width, height, int(req))}
	}
}

func (m *Model) previewPaneWidth() int {
	if m.compact {
		return max(20, m.width-8)
	}
	return max(24, m.width/3)
}

func (m *Model) previewPaneHeight() int {
	if m.compact {
		return max(8, m.height-9)
	}
	return max(10, m.height-9)
}

// buildFolderView renders a cheap folder summary. Only inexpensive data is
// gathered: entry count, a short contents listing, and the git branch when a
// .git directory exists. Folder size stays "Not calculated" unless the user
// runs the existing Calculate Folder Size action.
func buildFolderView(result orbitfs.LookupResult) string {
	var b strings.Builder
	b.WriteString(result.Name + "\n")
	b.WriteString("FOLDER\n\n")
	b.WriteString("Path\n" + result.Path + "\n\n")
	if !result.ModTime.IsZero() {
		b.WriteString("Modified\n" + result.ModTime.Format("Jan _2, 2006 15:04") + "\n\n")
	}
	entries, err := os.ReadDir(result.Path)
	if err == nil {
		b.WriteString(fmt.Sprintf("Items\n%d\n\n", len(entries)))
		if branch := folderGitBranch(result.Path); branch != "" {
			b.WriteString("Git\n" + branch + "\n\n")
		}
		b.WriteString("CONTENTS\n")
		for index, entry := range entries {
			if index >= 24 {
				b.WriteString("  …\n")
				break
			}
			suffix := ""
			if entry.IsDir() {
				suffix = "/"
			}
			b.WriteString("  " + entry.Name() + suffix + "\n")
		}
	}
	b.WriteString("\nSize\nNot calculated")
	return b.String()
}

func folderGitBranch(path string) string {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", path, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
