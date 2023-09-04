package scrollbar

import "git.sr.ht/~rockorager/vaxis"

type Model struct {
	// The character to display for the bar, defaults to '▐'
	Character vaxis.Character
	Style     vaxis.Style

	// Number of items in the scrolling area
	TotalHeight int
	// Number of items in the visible area
	ViewHeight int
	// Index of the item at the top of the visible area
	Top int
}

var defaultChar = vaxis.Character{
	Grapheme: "▐",
	Width:    1,
}

func (m *Model) Draw(win vaxis.Window) {
	if m.TotalHeight < 1 {
		return
	}

	if m.ViewHeight >= m.TotalHeight {
		// Only draw if needed
		return
	}
	_, h := win.Size()
	barH := (m.ViewHeight * h) / m.TotalHeight
	if barH < 1 {
		barH = 1
	}
	barTop := (m.Top * h) / m.TotalHeight

	if m.Character.Grapheme == "" {
		m.Character = defaultChar
	}
	for i := 0; i < barH; i += 1 {
		cell := vaxis.Cell{
			Character: m.Character,
			Style:     m.Style,
		}
		win.SetCell(0, barTop+i, cell)
	}
}
