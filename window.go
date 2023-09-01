package vaxis

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Fill completely fills the Window with the provided cell
func Fill(win Window, cell Text) {
	cols, rows := win.Size()
	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			win.SetCell(col, row, cell)
		}
	}
}

// Clear fills the Window with spaces with the default colors and removes all
// graphics placements
func Clear(win Window) {
	// We fill with a \x00 cell to differentiate between eg a text input
	// space and a cleared cell. \x00 is rendered as a space, but the
	// internal model will differentiate
	Fill(win, Text{Content: "\x00", WidthHint: 1})
	for k := range win.vx.graphicsNext {
		delete(win.vx.graphicsNext, k)
	}
}

// Print prints segments of Text, with each block having a given style.
// Text will be wrapped, line breaks will begin a new line at the first column
// of the surface. If the text overflows the height of the surface then only the
// top portion will be shown
func Print(win Window, segs ...Text) (col int, row int) {
	cols, rows := win.Size()
	log.Info("win", "cols", cols, "rows", rows)
	for _, seg := range segs {
		for _, char := range Characters(seg.Content) {
			if strings.ContainsRune(char.Grapheme, '\n') {
				col = 0
				row += 1
				continue
			}
			if row > rows {
				return col, row
			}
			chText := seg
			chText.Content = char.Grapheme
			chText.WidthHint = char.Width
			win.SetCell(col, row, chText)
			col += char.Width
			if col >= cols {
				row += 1
				col = 0
			}
		}
	}
	return col, row
}

// printWrap uses unicode line break logic to wrap text. this is expensive, but
// has good results
// TODO make this into a widget, it's too expensive to do every Draw call...we
// need to have a Reflow widget or something that cache's the line results and
// only reflows if the window is a different width
// func printWrap(win Window, segs ...Text) (col int, row int) {
// 	cols, rows := win.Size()
// 	for _, seg := range segs {
// 		var (
// 			b       = []byte(seg.Content)
// 			state   = -1
// 			cluster []byte
// 		)
// 		for len(b) > 0 {
// 			cluster, b, _, state = uniseg.Step(b, state)
// 			if row > rows {
// 				break
// 			}
// 			if uniseg.HasTrailingLineBreak(cluster) {
// 				// if col > maxWidth {
// 				// 	maxWidth = col
// 				// }
// 				col = 0
// 				row += 1
// 				continue
// 			}
// 			cSeg := seg
// 			cSeg.Content = string(cluster)
// 			win.SetCell(col, row, cSeg)
// 			col += characterWidth(string(cluster))
// 			if col+nextBreak(b) > cols {
// 				// if col > maxWidth {
// 				// 	maxWidth = col
// 				// }
// 				col = 0
// 				row += 1
// 			}
// 		}
// 	}
// 	return col, row
// }

// PrintLine prints a single line of text to the specified row. If the text is
// wider than the width of the window, trunc will be used as a truncating
// indicator (eg "This line has mo…"). If the row is outside the bounds of the
// window, nothing will be printed
func PrintLine(win Window, row int, trunc string, segs ...Text) {
	cols, rows := win.Size()
	if row >= rows {
		return
	}
	col := 0
	truncWidth := win.vx.characterWidth(trunc)
	for _, seg := range segs {
		for _, char := range Characters(seg.Content) {
			w := char.Width
			chText := seg
			if col+truncWidth+w > cols {
				chText.Content = trunc
				win.SetCell(col, row, chText)
				return
			}
			chText.Content = char.Grapheme
			chText.WidthHint = w
			win.SetCell(col, row, chText)
			col += w
		}
	}
}

// returns the stringwidth until the next can or must break
func nextBreak(b []byte) int {
	var (
		bound int
		w     int
	)
	state := -1
	for len(b) > 0 {
		_, b, bound, state = uniseg.Step(b, state)
		w += bound >> uniseg.ShiftWidth
		if bound&uniseg.MaskLine == uniseg.LineMustBreak {
			break
		}
		if bound&uniseg.MaskLine == uniseg.LineCanBreak {
			break
		}
	}
	return w
}

// Window is a Window with an offset from an optional parent and a specified size.
// If parent is nil, the underlying screen will be the parent and offsets will
// be relative to that.
type Window struct {
	vx     *Vaxis
	Parent *Window
	Column int // col offset from parent
	Row    int // row offset from parent
	Width  int // width of the surface, in cols
	Height int // height of the surface, in rows
}

// Window returns a window the full size of the screen. Child windows can be
// created from the returned Window
func (vx *Vaxis) Window() Window {
	w, h := vx.screenNext.size()
	return Window{
		Row:    0,
		Column: 0,
		Width:  w,
		Height: h,
		vx:     vx,
	}
}

// New creates a new child Window with an offset relative to the parent window
func (win Window) New(col, row, cols, rows int) Window {
	newWin := Window{
		Row:    row,
		Column: col,
		Width:  cols,
		Height: rows,
		Parent: &win,

		vx: win.vx,
	}
	w, h := win.Size()

	switch {
	case cols < 0:
		newWin.Width = w - col
	case cols+col > w:
		newWin.Width = w - col
	}

	switch {
	case rows < 0:
		newWin.Height = h - row
	case rows+row > w:
		newWin.Height = h - row
	}
	return newWin
}

// Size returns the visible size of the Window in character cells.
func (win Window) Size() (width int, height int) {
	return win.Width, win.Height
}

// SetCell is used to place data at the given cell location.  Note that since
// the Window doesn't retain this data, if the location is outside of the
// visible area, it is simply discarded.
func (win Window) SetCell(col int, row int, cell Text) {
	if row >= win.Height || col > win.Width {
		return
	}
	if row < 0 || col < 0 {
		return
	}
	switch win.Parent {
	case nil:
		win.vx.screenNext.setCell(col+win.Column, row+win.Row, cell)
	default:
		win.Parent.SetCell(col+win.Column, row+win.Row, cell)
	}
}

func (win Window) ShowCursor(col int, row int, style CursorStyle) {
	col += win.Column
	row += win.Row
	if win.Parent == nil {
		win.vx.ShowCursor(col, row, style)
		return
	}
	win.Parent.ShowCursor(col, row, style)
}

// returns the origin of the window, column x row, 0-indexed
func (win Window) origin() (int, int) {
	w := win
	col := 0
	row := 0
	for {
		col += w.Column
		row += w.Row
		if w.Parent == nil {
			return col, row
		}
		w = *w.Parent
	}
}
