package internal

import (
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
)

// The accessors in this file exist for tests that read main-model state from
// the test goroutine while the tea event loop concurrently runs Update()/View().
// They take m.mu so the reads are ordered against every event-loop mutation.
// They are not used by the production code paths, which only ever touch the
// model from the event-loop goroutine.

func (m *model) typingModalOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.typingModal.open
}

func (m *model) spfErrorIsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.spfError.IsOpen()
}

func (m *model) notifyModelIsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifyModel.IsOpen()
}

func (m *model) getNotifyModel() notify.Model {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifyModel
}

func (m *model) renameValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getFocusedFilePanel().Rename.Value()
}

func (m *model) previewContent() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fileModel.FilePreview.GetContent()
}

func (m *model) previewContentWidth() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fileModel.FilePreview.GetContentWidth()
}
