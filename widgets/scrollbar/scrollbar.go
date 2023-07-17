package scrollbar

import "git.sr.ht/~rockorager/vaxis"

type Model struct {
	// The character to display for the bar, defaults to '▐'
	Character  string
	Foreground vaxis.Color

	// Number of items in the scrolling area
	TotalHeight int
	// Number of items in the visible area
	ViewHeight int
	// Index of the item at the top of the visible area
	Top int
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

	if m.Character == "" {
		m.Character = "▐"
	}
	for i := 0; i < barH; i += 1 {
		cell := vaxis.Cell{
			Character:  m.Character,
			Foreground: m.Foreground,
		}
		win.SetCell(0, barTop+i, cell)
	}
}
