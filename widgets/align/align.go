package align

import "git.sr.ht/~rockorager/rtk"

// Center returns a Surface centered vertically and horizontally within the
// parent surface.
func Center(parent rtk.Window, cols int, rows int) rtk.Window {
	pCols, pRows := parent.Size()
	row := (pRows / 2) - (rows / 2)
	col := (pCols / 2) - (cols / 2)
	return rtk.NewWindow(&parent, col, row, cols, rows)
}

func TopLeft(parent rtk.Window, cols int, rows int) rtk.Window {
	return rtk.NewWindow(&parent, 0, 0, cols, rows)
}

func TopMiddle(parent rtk.Window, cols int, rows int) rtk.Window {
	pCols, _ := parent.Size()
	col := (pCols / 2) - (cols / 2)
	return rtk.NewWindow(&parent, col, 0, cols, rows)
}

func TopRight(parent rtk.Window, cols int, rows int) rtk.Window {
	pCols, _ := parent.Size()
	col := pCols - cols
	return rtk.NewWindow(&parent, col, 0, cols, rows)
}

func BottomLeft(parent rtk.Window, cols int, rows int) rtk.Window {
	_, pRows := parent.Size()
	row := pRows - rows
	return rtk.NewWindow(&parent, 0, row, cols, rows)
}

func BottomMiddle(parent rtk.Window, cols int, rows int) rtk.Window {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := (pCols / 2) - (cols / 2)
	return rtk.NewWindow(&parent, col, row, cols, rows)
}

func BottomRight(parent rtk.Window, cols int, rows int) rtk.Window {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := pCols - cols
	return rtk.NewWindow(&parent, col, row, cols, rows)
}
