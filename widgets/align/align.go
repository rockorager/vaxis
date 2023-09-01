package align

import "git.sr.ht/~rockorager/vaxis"

// Center returns a Surface centered vertically and horizontally within the
// parent surface.
func Center(parent vaxis.Window, cols int, rows int) vaxis.Window {
	pCols, pRows := parent.Size()
	row := (pRows / 2) - (rows / 2)
	col := (pCols / 2) - (cols / 2)
	return parent.New(col, row, cols, rows)
}

func TopLeft(parent vaxis.Window, cols int, rows int) vaxis.Window {
	return parent.New(0, 0, cols, rows)
}

func TopMiddle(parent vaxis.Window, cols int, rows int) vaxis.Window {
	pCols, _ := parent.Size()
	col := (pCols / 2) - (cols / 2)
	return parent.New(col, 0, cols, rows)
}

func TopRight(parent vaxis.Window, cols int, rows int) vaxis.Window {
	pCols, _ := parent.Size()
	col := pCols - cols
	return parent.New(col, 0, cols, rows)
}

func BottomLeft(parent vaxis.Window, cols int, rows int) vaxis.Window {
	_, pRows := parent.Size()
	row := pRows - rows
	return parent.New(0, row, cols, rows)
}

func BottomMiddle(parent vaxis.Window, cols int, rows int) vaxis.Window {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := (pCols / 2) - (cols / 2)
	return parent.New(col, row, cols, rows)
}

func BottomRight(parent vaxis.Window, cols int, rows int) vaxis.Window {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := pCols - cols
	return parent.New(col, row, cols, rows)
}
