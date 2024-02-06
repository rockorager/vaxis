package vaxis

import (
	"strings"

	"github.com/rivo/uniseg"
)

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
	vx.mu.Lock()
	w, h := vx.screenNext.size()
	vx.mu.Unlock()
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
	case rows+row > h:
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
func (win Window) SetCell(col int, row int, cell Cell) {
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

// SetStyle changes the style at a given location, leaving the text in place.
func (win Window) SetStyle(col int, row int, style Style) {
	if row >= win.Height || col >= win.Width {
		return
	}
	if row < 0 || col < 0 {
		return
	}
	switch win.Parent {
	case nil:
		win.Vx.screenNext.setStyle(col+win.Column, row+win.Row, style)
	default:
		win.Parent.SetStyle(col+win.Column, row+win.Row, style)
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
func (win Window) Fill(cell Cell) {
	cols, rows := win.Size()
	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			win.SetCell(col, row, cell)
		}
	}
}

// returns the Origin of the window, column x row, 0-indexed
func (win Window) Origin() (int, int) {
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
	win.Fill(Cell{Character: Character{" ", 1}, Style: Style{}})
	win.Vx.graphicsNext = []*placement{}
}

// Print prints [Segment]s, with each block having a given style. Text will be
// wrapped, line breaks will begin a new line at the first column of the surface.
// If the text overflows the height of the surface then only the top portion
// will be shown
func (win Window) Print(segs ...Segment) (col int, row int) {
	cols, rows := win.Size()
	for _, seg := range segs {
		for _, char := range Characters(seg.Text) {
			if strings.ContainsRune(char.Grapheme, '\n') {
				col = 0
				row += 1
				continue
			}
			if row > rows {
				return col, row
			}
			if !win.Vx.caps.unicodeCore {
				// characterWidth will cache the result
				char.Width = win.Vx.characterWidth(char.Grapheme)
			}
			cell := Cell{
				Character: char,
				Style:     seg.Style,
			}
			win.SetCell(col, row, cell)
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
func (win Window) PrintTruncate(row int, segs ...Segment) {
	cols, rows := win.Size()
	if row >= rows {
		return
	}
	col := 0
	truncator := Character{
		Grapheme: "…",
		Width:    1,
	}
	for _, seg := range segs {
		for _, char := range Characters(seg.Text) {
			if !win.Vx.caps.unicodeCore {
				// characterWidth will cache the result
				char.Width = win.Vx.characterWidth(char.Grapheme)
			}
			w := char.Width
			cell := Cell{
				Character: char,
				Style:     seg.Style,
			}
			if col+truncator.Width+w > cols {
				cell.Character = truncator
				win.SetCell(col, row, cell)
				return
			}
			win.SetCell(col, row, cell)
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
func (win Window) Println(row int, segs ...Segment) {
	cols, rows := win.Size()
	if row >= rows {
		return
	}
	col := 0
	for _, seg := range segs {
		for _, char := range Characters(seg.Text) {
			if !win.Vx.caps.unicodeCore {
				// characterWidth will cache the result
				char.Width = win.Vx.characterWidth(char.Grapheme)
			}
			w := char.Width
			if col+w > cols {
				return
			}
			if !win.Vx.caps.unicodeCore {
				// characterWidth will cache the result
				win.Vx.characterWidth(char.Grapheme)
			}
			cell := Cell{
				Character: char,
				Style:     seg.Style,
			}
			win.SetCell(col, row, cell)
			col += w
		}
	}
}

// Wrap uses unicode line break logic to wrap text. this is expensive, but
// has good results
func (win Window) Wrap(segs ...Segment) (col int, row int) {
	cols, rows := win.Size()
	var (
		state   = -1
		segment string
	)
	for _, seg := range segs {
		rest := seg.Text
		for len(rest) > 0 {
			if row >= rows {
				break
			}
			segment, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)
			chars := Characters(segment)
			total := 0
			for _, char := range chars {
				if !win.Vx.caps.unicodeCore {
					// characterWidth will cache the result
					char.Width = win.Vx.characterWidth(char.Grapheme)
				}
				total += char.Width
			}
			// Figure out how to break the line
			switch {
			case total > cols:
				// the line is greater than our entire width, so we'll
				// break at a grapheme
			case total+col > cols:
				// there isn't space left, go to a new line
				col = 0
				row += 1
			default:
				// it fits on our line. Print it
			}
			for _, char := range chars {
				if uniseg.HasTrailingLineBreakInString(char.Grapheme) {
					row += 1
					col = 0
					continue
				}
				cell := Cell{
					Character: char,
					Style:     seg.Style,
				}
				win.SetCell(col, row, cell)
				col += char.Width
				if col >= cols {
					row += 1
					col = 0
				}
			}
		}
	}
	return col, row
}
