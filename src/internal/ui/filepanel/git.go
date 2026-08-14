package filepanel

import "path/filepath"

func (m *Model) SetGitStatus(root, branch string, statuses map[string]string) {
	if root == "" || !withinRoot(root, m.Location) {
		m.GitRoot = ""
		m.GitBranch = ""
		m.GitStatus = nil
		return
	}
	m.GitRoot = root
	m.GitBranch = branch
	m.GitStatus = statuses
}

func withinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && (relative == "." || (relative != ".." && (len(relative) < 3 || relative[:3] != ".."+string(filepath.Separator))))
}
