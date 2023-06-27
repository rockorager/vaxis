package rtk

import (
	"github.com/rivo/uniseg"
)

// Fill completely fills the Window with the provided cell
func Fill(win Window, cell Cell) {
	cols, rows := win.Size()
	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			win.SetCell(col, row, cell)
		}
	}
}

// Clear fills the Window with spaces with the default colors
func Clear(win Window) {
	Fill(win, Cell{Character: " "})
}

// Print prints text to a Window. The text will be wrapped to the width, line
// breaks will begin a new line at the first column of the surface. If the text
// overflows the height of the surface then only the top portion will be shown.
// Print returns the max width of the text area, and the position of the cursor
// at the end of the text.
func Print(win Window, text string) (maxWidth int, col int, row int) {
	cols, rows := win.Size()
	for _, char := range Characters(text) {
		if row > rows {
			break
		}
		if char == "\n" {
			if col > maxWidth {
				maxWidth = col
			}
			col = 0
			row += 1
			continue
		}
		w := uniseg.StringWidth(char)
		if col+w > cols {
			if col > maxWidth {
				maxWidth = col
			}
			col = 0
			row += 1
		}
		win.SetCell(col, row, Cell{Character: char})
		col += w
	}
	return maxWidth, col, row
}

type Segment struct {
	Text       string
	Foreground Color
	Background Color
	Attributes AttributeMask
}

// PrintSegments prints Segments of text, with each block having a given style.
// Text will be wrapped, line breaks will begin a new line at the first column
// of the surface. If the text overflows the height of the surface then only the
// top portion will be shown
func PrintSegments(win Window, segs ...Segment) (maxWidth int, col int, row int) {
	cols, rows := win.Size()
	for _, seg := range segs {
		var (
			b              = []byte(seg.Text)
			state      int = -1
			boundaries int
			cluster    []byte
		)
		for len(b) > 0 {
			cluster, b, boundaries, state = uniseg.Step(b, state)
			if row > rows {
				break
			}
			if boundaries&uniseg.MaskLine == uniseg.LineMustBreak {
				if col > maxWidth {
					maxWidth = col
				}
				col = 0
				row += 1
				continue
			}
			win.SetCell(col, row, Cell{
				Character:  string(cluster),
				Foreground: seg.Foreground,
				Background: seg.Background,
				Attribute:  seg.Attributes,
			})

			col += boundaries >> uniseg.ShiftWidth
			if col+nextBreak(b) > cols {
				if col > maxWidth {
					maxWidth = col
				}
				col = 0
				row += 1
			}
		}
	}
	return maxWidth, col, row
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
	Column int // col offset from parent
	Row    int // row offset from parent
	Width  int // width of the surface, in cols
	Height int // height of the surface, in rows
	Parent *Window
}

// NewWindow returns a new Window. The x and y coordinates are an offset
// relative to the parent. The origin 0,0 represents the upper left.  The width
// and height can be set to 0 to have the window expand to fill it's parent. The
// Window cannot exist outside of it's parent's Window.
func NewWindow(parent *Window, col, row, cols, rows int) Window {
	return Window{
		Row:    row,
		Column: col,
		Width:  cols,
		Height: rows,
		Parent: parent,
	}
}

// Size returns the visible size of the Window in character cells.
func (win Window) Size() (width int, height int) {
	var (
		pCols int
		pRows int
	)
	switch {
	case win.Parent == nil:
		if stdScreen == nil {
			return 0, 0
		}
		pCols, pRows = stdScreen.Size()
	default:
		pCols, pRows = win.Parent.Size()
	}

	switch {
	case (win.Column + win.Width) > pCols:
		width = pCols - win.Column
	case win.Width <= 0:
		width = pCols - win.Column
	default:
		width = win.Width
	}
	switch {
	case (win.Row + win.Height) > pRows:
		height = pRows - win.Row
	case win.Height <= 0:
		height = pRows - win.Row
	default:
		height = win.Height
	}
	return width, height
}

// SetCell is used to place data at the given cell location.  Note that since
// the Window doesn't retain this data, if the location is outside of the
// visible area, it is simply discarded.
func (win Window) SetCell(col int, row int, cell Cell) {
	cols, rows := win.Size()
	if cols == 0 || rows == 0 {
		return
	}
	if col >= cols {
		return
	}
	if row >= rows {
		return
	}
	switch {
	case win.Parent == nil:
		stdScreen.SetCell(col+win.Column, row+win.Row, cell)
	default:
		win.Parent.SetCell(col+win.Column, row+win.Row, cell)
	}
}

func (win Window) ShowCursor(col int, row int, style CursorStyle) {
	if win.Parent == nil {
		ShowCursor(col, row, style)
		return
	}
	win.Parent.ShowCursor(col+win.Column, row+win.Row, style)
}
