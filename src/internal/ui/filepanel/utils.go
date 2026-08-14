package filepanel

import "math"

func (m *Model) GetCursor() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursor
}

func (m *Model) GetLocation() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Location
}

func (m *Model) GetRenderIndex() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.renderIndex
}

func (m *Model) GetFocusedItem() Element {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getFocusedItemUnlocked()
}

func (m *Model) getFocusedItemUnlocked() Element {
	return m.getElementAtIdxUnlocked(m.GetCursorUnlocked())
}

func (m *Model) GetElementAtIdx(idx int) Element {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getElementAtIdxUnlocked(idx)
}

func (m *Model) getElementAtIdxUnlocked(idx int) Element {
	if idx < 0 || m.elemCountUnlocked() <= idx {
		return Element{}
	}
	return m.element[idx]
}

func (m *Model) GetFirstElement() Element {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getFirstElementUnlocked()
}

func (m *Model) getFirstElementUnlocked() Element {
	return m.getElementAtIdxUnlocked(0)
}

func (m *Model) ResetSelected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.selectOrderCounter = 0
	m.selected = make(map[string]int)
}

// For modification. Make sure to do a nil check
func (m *Model) GetFocusedItemPtr() *Element {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetCursorUnlocked() < 0 || m.elemCountUnlocked() <= m.GetCursorUnlocked() {
		return nil
	}
	copyElem := m.element[m.GetCursorUnlocked()]
	return &copyElem
}

// Note : If this is called on an already selected element
// it will make its order last. This is expected behaviour
func (m *Model) SetSelected(location string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setSelectedUnlocked(location)
}

func (m *Model) setSelectedUnlocked(location string) {
	m.selectOrderCounter++
	m.selected[location] = m.selectOrderCounter
}

func (m *Model) SetUnSelected(location string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.checkSelectedUnlocked(location) {
		delete(m.selected, location)
	}
}

func (m *Model) ToggleSelected(location string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toggleSelectedUnlocked(location)
}

func (m *Model) toggleSelectedUnlocked(location string) {
	if m.checkSelectedUnlocked(location) {
		delete(m.selected, location)
		return
	}
	m.setSelectedUnlocked(location)
}

// Only used in tests, including tests outside this package
func (m *Model) SetSelectedAll(locations []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, location := range locations {
		m.setSelectedUnlocked(location)
	}
}

func (m *Model) CheckSelected(location string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkSelectedUnlocked(location)
}

func (m *Model) checkSelectedUnlocked(location string) bool {
	_, isSelected := m.selected[location]
	return isSelected
}

// Returns an unordered list of selected locations
func (m *Model) GetSelectedLocations() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, 0, len(m.selected))
	for k := range m.selected {
		result = append(result, k)
	}
	return result
}

// Returns an ordered list of selected locations. Order like user see in filepanel.
func (m *Model) GetSelectedLocationsSortedAsVisible() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.selected) == 0 {
		return []string{}
	}
	if len(m.selected) == 1 {
		for k := range m.selected {
			return []string{k}
		}
	}
	result := make([]string, 0, len(m.selected))
	for _, el := range m.element {
		if _, ok := m.selected[el.Location]; ok {
			result = append(result, el.Location)
		}
	}
	return result
}

func (m *Model) GetFirstSelectedLocation() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.selected) == 0 {
		return ""
	}
	result := ""
	minOrder := math.MaxInt
	for location, order := range m.selected {
		if minOrder > order {
			result = location
			minOrder = order
		}
	}
	return result
}

// Select the item where cursor located (only work on select mode)
func (m *Model) SingleItemSelect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.singleItemSelectUnlocked()
}

func (m *Model) singleItemSelectUnlocked() {
	if !m.emptyOrInvalidUnlocked() {
		m.toggleSelectedUnlocked(m.getFocusedItemUnlocked().Location)
	}
}

func (m *Model) ElemCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.elemCountUnlocked()
}

func (m *Model) elemCountUnlocked() int {
	return len(m.element)
}

func (m *Model) SelectedCount() uint {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.selectedCountUnlocked()
}

func (m *Model) selectedCountUnlocked() uint {
	return uint(len(m.selected))
}

func (m *Model) Empty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.emptyUnlocked()
}

func (m *Model) emptyUnlocked() bool {
	return m.elemCountUnlocked() == 0
}

func (m *Model) EmptyOrInvalid() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.emptyOrInvalidUnlocked()
}

func (m *Model) emptyOrInvalidUnlocked() bool {
	return m.emptyUnlocked() || m.validateCursorAndRenderIndexUnlocked() != nil
}

func (m *Model) ToggleReverseSort() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SortReversed = !m.SortReversed
}

// SetCursorPosition sets cursor and updates renderIndex accordingly.
// Note: Intended for test utilities only!!!!!
func (m *Model) SetCursorPosition(cursor int) {
	m.scrollToCursor(cursor)
}

func (m *Model) FindElementIndexByName(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, elem := range m.element {
		if elem.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) FindElementIndexByLocation(location string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, elem := range m.element {
		if elem.Location == location {
			return i
		}
	}
	return -1
}
