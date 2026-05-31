package vaxis

import (
	"strings"

	"github.com/rockorager/go-uucode"
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

// Window returns the root drawing window for the surface currently owned by
// Vaxis. In alternate-screen mode this is the full terminal screen. In
// primary-screen mode this is the live region surface. Child windows can be
// created from the returned Window.
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

	vx := win.Vx
	w := &win
	for {
		col += w.Column
		row += w.Row
		if w.Vx != nil {
			vx = w.Vx
		}
		if w.Parent == nil {
			break
		}
		w = w.Parent
		if row >= w.Height || col >= w.Width {
			return
		}
		if row < 0 || col < 0 {
			return
		}
	}
	if vx == nil || vx.screenNext == nil {
		return
	}
	if row >= vx.screenNext.rows || col >= vx.screenNext.cols {
		return
	}
	if row < 0 || col < 0 {
		return
	}
	vx.screenNext.setCellDirect(col, row, cell)
}

// SetStyle changes the style at a given location, leaving the text in place.
func (win Window) SetStyle(col int, row int, style Style) {
	if row >= win.Height || col >= win.Width {
		return
	}
	if row < 0 || col < 0 {
		return
	}

	vx := win.Vx
	w := &win
	for {
		col += w.Column
		row += w.Row
		if w.Vx != nil {
			vx = w.Vx
		}
		if w.Parent == nil {
			break
		}
		w = w.Parent
		if row >= w.Height || col >= w.Width {
			return
		}
		if row < 0 || col < 0 {
			return
		}
	}
	if vx == nil || vx.screenNext == nil {
		return
	}
	if row >= vx.screenNext.rows || col >= vx.screenNext.cols {
		return
	}
	if row < 0 || col < 0 {
		return
	}
	vx.screenNext.setStyleDirect(col, row, style)
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
		it := NewCharacterIterator(seg.Text)
		for char, ok := it.Next(); ok; char, ok = it.Next() {
			if strings.ContainsRune(char.Grapheme, '\n') {
				col = 0
				row += 1
				continue
			}
			if row > rows {
				return col, row
			}
			if !win.Vx.caps.unicodeCore || !win.Vx.caps.explicitWidth {
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
	if row < 0 || row >= rows || cols <= 0 {
		return
	}

	type printCell struct {
		char  Character
		style Style
	}

	truncator := Character{
		Grapheme: "…",
		Width:    1,
	}
	col := 0
	var pending printCell
	havePending := false
	flush := func(cell printCell, more bool) bool {
		w := cell.char.Width
		if col+w > cols {
			ellipsisCol := col
			if ellipsisCol > cols-truncator.Width {
				ellipsisCol = cols - truncator.Width
			}
			win.SetCell(ellipsisCol, row, Cell{
				Character: truncator,
				Style:     cell.style,
			})
			return false
		}
		if more && col+w >= cols {
			win.SetCell(cols-truncator.Width, row, Cell{
				Character: truncator,
				Style:     cell.style,
			})
			return false
		}
		win.SetCell(col, row, Cell{
			Character: cell.char,
			Style:     cell.style,
		})
		col += w
		return true
	}

	for _, seg := range segs {
		it := NewCharacterIterator(seg.Text)
		for char, ok := it.Next(); ok; char, ok = it.Next() {
			if !win.Vx.caps.unicodeCore || !win.Vx.caps.explicitWidth {
				// characterWidth will cache the result
				char.Width = win.Vx.characterWidth(char.Grapheme)
			}
			cell := printCell{
				char:  char,
				style: seg.Style,
			}
			if havePending && !flush(pending, true) {
				return
			}
			pending = cell
			havePending = true
		}
	}
	if havePending {
		flush(pending, false)
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
		it := NewCharacterIterator(seg.Text)
		for char, ok := it.Next(); ok; char, ok = it.Next() {
			if !win.Vx.caps.unicodeCore || !win.Vx.caps.explicitWidth {
				// characterWidth will cache the result
				char.Width = win.Vx.characterWidth(char.Grapheme)
			}
			w := char.Width
			if col+w > cols {
				return
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
	width := func(char *Character) {
		if !win.Vx.caps.unicodeCore || !win.Vx.caps.explicitWidth {
			// characterWidth will cache the result
			char.Width = win.Vx.characterWidth(char.Grapheme)
		}
	}
	emit := func(char Character, style Style) {
		if hasTrailingLineBreakInString(char.Grapheme) {
			row += 1
			col = 0
			return
		}
		cell := Cell{
			Character: char,
			Style:     style,
		}
		win.SetCell(col, row, cell)
		col += char.Width
		if col >= cols {
			row += 1
			col = 0
		}
	}
	for _, seg := range segs {
		rest := seg.Text
		for len(rest) > 0 {
			if row >= rows {
				break
			}
			lineIt := uucode.NewLineIterator(rest)
			lineSegment, ok := lineIt.Next()
			if !ok {
				break
			}
			segment := rest[lineSegment.Start:lineSegment.End]
			rest = rest[lineSegment.End:]
			var buffered [64]Character
			chars := buffered[:0]
			total := 0
			tooWide := false
			overflow := false
			charIt := NewCharacterIterator(segment)
			for char, ok := charIt.Next(); ok; char, ok = charIt.Next() {
				width(&char)
				total += char.Width
				if tooWide {
					emit(char, seg.Style)
					continue
				}
				if !overflow && total > cols {
					tooWide = true
					for _, bufferedChar := range chars {
						emit(bufferedChar, seg.Style)
					}
					emit(char, seg.Style)
					continue
				}
				if len(chars) == cap(chars) {
					overflow = true
					continue
				}
				chars = append(chars, char)
			}
			if tooWide {
				continue
			}
			if total <= cols && total+col > cols {
				col = 0
				row += 1
			}
			if overflow {
				charIt = NewCharacterIterator(segment)
				for char, ok := charIt.Next(); ok; char, ok = charIt.Next() {
					width(&char)
					emit(char, seg.Style)
				}
				continue
			}
			for _, char := range chars {
				emit(char, seg.Style)
			}
		}
	}
	return col, row
}
