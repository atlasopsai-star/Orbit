package filemodel

func (m *Model) SetGitStatus(root, branch string, statuses map[string]string) {
	for index := range m.FilePanels {
		m.FilePanels[index].SetGitStatus(root, branch, statuses)
	}
}
