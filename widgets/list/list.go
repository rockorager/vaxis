package list

import (
	"git.sr.ht/~rockorager/vaxis"
)

type List struct {
	index  int
	items  []string
	offset int
}

func New(items []string) List {
	return List{
		items: items,
	}
}

func (m *List) Draw(win vaxis.Window) {
	_, height := win.Size()
	if m.index >= m.offset+height {
		m.offset = m.index - height + 1
	} else if m.index < m.offset {
		m.offset = m.index
	}

	defaultStyle := vaxis.Style{}
	selectedStyle := vaxis.Style{Attribute: vaxis.AttrReverse}

	index := m.index - m.offset
	for i, subject := range m.items[m.offset:] {
		var style vaxis.Style
		if i == index {
			style = selectedStyle
		} else {
			style = defaultStyle
		}
		win.Println(i, vaxis.Segment{Text: subject, Style: style})
	}

}

func (m *List) Down() {
	m.index = min(len(m.items)-1, m.index+1)
}

func (m *List) Up() {
	m.index = max(0, m.index-1)
}

func (m *List) Home() {
	m.index = 0
}

func (m *List) End() {
	m.index = len(m.items) - 1
}

func (m *List) PageDown(win vaxis.Window) {
	_, height := win.Size()
	m.index = min(len(m.items)-1, m.index+height)
}

func (m *List) PageUp(win vaxis.Window) {
	_, height := win.Size()
	m.index = max(0, m.index-height)
}

func (m *List) SetItems(items []string) {
	m.items = items
	m.index = min(len(items)-1, m.index)
}

// Returns the index of the currently selected item.
func (m *List) Index() int {
	return m.index
}

// Can be deleted once minimal go bumps to 1.21
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Can be deleted once minimal go bumps to 1.21
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
