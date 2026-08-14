package internal

import tea "charm.land/bubbletea/v2"

func (m *model) openGitDiff(path string) tea.Cmd {
	if m.gitStatus.Root == "" {
		return nil
	}
	return m.gitDiffModal.Open(m.gitStatus.Root, path)
}
