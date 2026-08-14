package filepanel

import (
	"fmt"
)

func (m *Model) scrollToCursor(cursor int) {
	m.mu.Lock()
	m.scrollToCursorUnlocked(cursor)
	m.mu.Unlock()
}

func (m *Model) scrollToCursorUnlocked(cursor int) {
	if cursor < 0 || cursor >= m.elemCountUnlocked() {
		return
	}
	m.cursor = cursor

	// Modify renderIndex if needed
	renderCount := m.panelElementHeightUnlocked()
	if m.cursor < m.renderIndex {
		// Due to size change, when last element is selected, we might have
		// empty space (renderIndex ... ElemCount()-1 spans less then renderCount)
		// Even with >0 renderIndex
		m.renderIndex = m.cursor
	} else if m.cursor > m.renderIndex+renderCount-1 {
		m.renderIndex = m.cursor - renderCount + 1
	}
}

func (m *Model) moveCursorBy(delta int) {
	if m.emptyUnlocked() {
		return
	}
	// Wrap cursor
	cursor := (m.cursor + delta + m.elemCountUnlocked()) % m.elemCountUnlocked()
	m.scrollToCursorUnlocked(cursor)
}

func (m *Model) pageScrollBy(delta int) {
	if m.emptyUnlocked() {
		return
	}
	cursor := m.cursor + delta
	if cursor < 0 {
		cursor = 0
	} else if cursor >= m.elemCountUnlocked() {
		cursor = m.elemCountUnlocked() - 1
	}
	m.scrollToCursorUnlocked(cursor)
}

// Control file panel list up
func (m *Model) ListUp() {
	m.mu.Lock()
	m.moveCursorBy(-1)
	m.mu.Unlock()
}

// Control file panel list down
func (m *Model) ListDown() {
	m.mu.Lock()
	m.moveCursorBy(1)
	m.mu.Unlock()
}

func (m *Model) PgUp() {
	m.mu.Lock()
	m.pageScrollBy(-m.getPageScrollSize())
	m.mu.Unlock()
}

func (m *Model) PgDown() {
	m.mu.Lock()
	m.pageScrollBy(m.getPageScrollSize())
	m.mu.Unlock()
}

// Handles the action of selecting an item in the file panel upwards. (only work on select mode)
// This basically just toggles the "selected" status of element that is pointed by the cursor
// and then moves the cursor up
// TODO : Add unit tests for ItemSelectUp and singleItemSelect
func (m *Model) ItemSelectUp() {
	m.mu.Lock()
	m.singleItemSelectUnlocked()
	m.moveCursorBy(-1)
	m.mu.Unlock()
}

// Handles the action of selecting an item in the file panel downwards. (only work on select mode)
func (m *Model) ItemSelectDown() {
	m.mu.Lock()
	m.singleItemSelectUnlocked()
	m.moveCursorBy(1)
	m.mu.Unlock()
}

// Applies targetFile cursor positioning, if configured for the panel.
// Must be called with m.mu held (from updateElementsIfNeededUnlocked).
func (m *Model) applyTargetFileCursor() {
	idx := m.findElementIndexByNameUnlocked(m.TargetFile)
	if idx != -1 {
		m.scrollToCursorUnlocked(idx)
	}
	m.TargetFile = ""
}

func (m *Model) findElementIndexByNameUnlocked(name string) int {
	for i, elem := range m.element {
		if elem.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) ValidateCursorAndRenderIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validateCursorAndRenderIndexUnlocked()
}

func (m *Model) validateCursorAndRenderIndexUnlocked() error {
	if m.cursor < 0 || m.elemCountUnlocked() <= m.cursor {
		return fmt.Errorf("invalid cursor : %d, element count : %d", m.cursor, m.elemCountUnlocked())
	}
	renderCount := m.panelElementHeightUnlocked()
	if (m.cursor < m.renderIndex) || (m.cursor > m.renderIndex+renderCount-1) {
		return fmt.Errorf("invalid renderIndex : %d, cursor : %d, renderCount : %d",
			m.renderIndex, m.cursor, renderCount)
	}
	return nil
}

func (m *Model) GetCursorUnlocked() int {
	return m.cursor
}
