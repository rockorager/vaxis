package vaxis

import (
	"strings"

	"github.com/rivo/uniseg"
)

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

// Window is a Window with an offset from an optional parent and a specified
// size. A Window can be instantiated directly, however the provided constructor
// methods are recommended as they will enforce size constraints
type Window struct {
	// Vx is a reference to the [Vx] instance
	Vx *Vaxis
	// Parent is a reference to a parent [Window], if nil then the offsets
	// and size will be relative to the underlying terminal window
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
		Vx:     vx,
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

		Vx: win.Vx,
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
	if row >= win.Height || col >= win.Width {
		return
	}
	if row < 0 || col < 0 {
		return
	}
	switch win.Parent {
	case nil:
		win.Vx.screenNext.setCell(col+win.Column, row+win.Row, cell)
	default:
		win.Parent.SetCell(col+win.Column, row+win.Row, cell)
	}
}

// ShowCursor shows the cursor at colxrow, relative to this Window's location
func (win Window) ShowCursor(col int, row int, style CursorStyle) {
	col += win.Column
	row += win.Row
	if win.Parent == nil {
		win.Vx.ShowCursor(col, row, style)
		return
	}
	win.Parent.ShowCursor(col, row, style)
}

// Fill completely fills the Window with the provided cell
func (win Window) Fill(cell Text) {
	cols, rows := win.Size()
	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			win.SetCell(col, row, cell)
		}
	}
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

// Clear fills the Window with spaces with the default colors and removes all
// graphics placements
func (win Window) Clear() {
	// We fill with a \x00 cell to differentiate between eg a text input
	// space and a cleared cell. \x00 is rendered as a space, but the
	// internal model will differentiate
	win.Fill(Text{Content: "\x00", WidthHint: 1})
	for k := range win.Vx.graphicsNext {
		delete(win.Vx.graphicsNext, k)
	}
}

// Print prints segments of Text, with each block having a given style.
// Text will be wrapped, line breaks will begin a new line at the first column
// of the surface. If the text overflows the height of the surface then only the
// top portion will be shown
func (win Window) Print(segs ...Text) (col int, row int) {
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

// PrintTruncate prints a single line of text to the specified row. If the text is
// wider than the width of the window, the line will be truncated with "…":
//
//	"This line has mo…"
//
// If the row is outside the bounds of the window, nothing will be printed
func (win Window) PrintTruncate(row int, segs ...Text) {
	cols, rows := win.Size()
	if row >= rows {
		return
	}
	col := 0
	trunc := "…"
	truncWidth := 1
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

// Println prints a single line of text to the specified row. If the text is
// wider than the width of the window, the line will be truncated with "…":
//
//	"This line has mo…"
//
// If the row is outside the bounds of the window, nothing will be printed
func (win Window) Println(row int, segs ...Text) {
	cols, rows := win.Size()
	if row >= rows {
		return
	}
	col := 0
	for _, seg := range segs {
		for _, char := range Characters(seg.Content) {
			w := char.Width
			chText := seg
			if col+w > cols {
				return
			}
			chText.Content = char.Grapheme
			chText.WidthHint = w
			win.SetCell(col, row, chText)
			col += w
		}
	}
}
