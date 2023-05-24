package rtk

import "github.com/rivo/uniseg"

// Surface represents a logical view on an area
type Surface interface {
	// SetContent is used to update the content of the View at the given
	// location.  This will generally be called by the Draw() method of
	// a Widget.
	SetCell(col int, row int, cell Cell)

	// Size returns the current size of the Surface
	Size() (cols int, rows int)

	// Resize changes the physical size of the Surface
	Resize(cols, rows int)

	// Move the Surface to a new offset. Both row and col are 0-indexed
	Move(col, row int)

	// ShowCursor is used to display the cursor at a given location.
	// If the coordinates -1, -1 are given or are otherwise outside the
	// dimensions of the screen, the cursor will be hidden.
	ShowCursor(col int, row int)
}

// Fill completely fills the surface with the provided cell
func Fill(srf Surface, cell Cell) {
	cols, rows := srf.Size()
	for row := 0; row < rows; row += 1 {
		for col := 0; col < cols; col += 1 {
			srf.SetCell(col, row, cell)
		}
	}
}

// Clear fills the surface with spaces with the default colors
func Clear(srf Surface) {
	Fill(srf, Cell{EGC: " "})
}

// Print prints text to a surface. The text will be wrapped to the width, line
// breaks will begin a new line at the first column of the surface. If the text
// overflows the height of the surface then only the top portion will be shown
func Print(srf Surface, text string) {
	cols, rows := srf.Size()
	var (
		row = 0
		col = 0
	)
	for _, egc := range EGCs(text) {
		if row > rows {
			break
		}
		if egc == "\n" {
			col = 0
			row += 1
			continue
		}
		w := uniseg.StringWidth(egc)
		if col+w > cols {
			col = 0
			row += 1
		}
		srf.SetCell(col, row, Cell{EGC: egc})
		col += w
	}
}

// Printf prints ansi-styled strings to the surface. Text will be wrapped, line
// breaks will begin a new line at the first column of the surface. If the text
// overflows the height of the surface then only the top portion will be shown
func Printf(srf Surface, text string) {
	// TODO
}

// SubSurface is a Surface with an offset from some parent and a specified size.
// The subsurface can be moved or resized after creation.
type SubSurface struct {
	col    int // col offset from parent
	row    int // row offset from parent
	cols   int // width of the surface, in cols
	rows   int // height of the surface, in rows
	parent Surface
}

// NewSubSurface returns a new SubSurface. The x and y coordinates are an offset
// relative to the parent. The origin 0,0 represents the upper left.  The width
// and height indicate a width and height. If the value -1 is supplied, then the
// dimension is calculated from the parent.
func NewSubSurface(srf Surface, col, row, cols, rows int) *SubSurface {
	v := &SubSurface{
		row:    row,
		col:    col,
		parent: srf,
	}
	v.Resize(cols, rows)
	return v
}

// Size returns the visible size of the SubSurface in character cells.
func (srf *SubSurface) Size() (int, int) {
	return srf.cols, srf.rows
}

// SetContent is used to place data at the given cell location.  Note that since
// the SubSurface doesn't retain this data, if the location is outside of the
// visible area, it is simply discarded.
//
// Generally, this is called during the View() phase by the object that
// represents the content.
func (srf *SubSurface) SetCell(col int, row int, cell Cell) {
	if srf.parent == nil {
		return
	}
	if col < 0 || row < 0 {
		return
	}
	if col >= srf.cols {
		return
	}
	if row >= srf.rows {
		return
	}
	srf.parent.SetCell(col+srf.col, row+srf.row, cell)
}

// ShowCursor is used to display the cursor at a given location. If the
// coordinates -1, -1 are given or are otherwise outside the dimensions of the
// screen, the cursor will be hidden.
func (srf *SubSurface) ShowCursor(col int, row int) {
	srf.parent.ShowCursor(col+srf.col, row+srf.row)
}

// Resize resizes the surface. A negative value for either width or height will
// cause the SubSurface to expand to fill to the end of the parent in the
// relevant dimension
func (srf *SubSurface) Resize(cols int, rows int) {
	if srf.parent == nil {
		return
	}
	pCols, pRows := srf.parent.Size()
	if cols < 0 || cols > pCols-cols {
		cols = pCols - cols
	}
	if rows < 0 || rows > pRows-rows {
		rows = pRows - rows
	}

	srf.cols = cols
	srf.rows = rows
}

func (srf *SubSurface) Move(col int, row int) {
	srf.col = col
	srf.row = row
}

// GlobalCoordinates walks up the SubSurface parents until the parent is not a
// SubSurface, and returns the coordinates at this level. If the root surface is
// the tcell.Screen and all children are SubSurfaces, this will return the
// global coordinates.
func (srf *SubSurface) GlobalCoordinates(localX int, localY int) (int, int) {
	x := localX + srf.col
	y := localY + srf.row
	s := srf
	for {
		parent, ok := s.parent.(*SubSurface)
		if !ok {
			break
		}
		x = x + parent.col
		y = y + parent.row
		s = parent
	}
	return x, y
}
