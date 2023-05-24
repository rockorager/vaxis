package rtk

import (
	"sync"
)

type stdSurface struct {
	buf  [][]Cell
	mu   sync.Mutex
	rows int
	cols int
	rtk  *RTK
}

func newStdSurface(rtk *RTK) *stdSurface {
	std := &stdSurface{
		rtk: rtk,
	}
	return std
}

func (std *stdSurface) Size() (cols int, rows int) {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.cols, std.rows
}

func (std *stdSurface) Resize(cols int, rows int) {
	// Exported resize is a no-op
}

// resize resizes the stdsurface based on a SIGWINCH
func (std *stdSurface) resize(cols int, rows int) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.buf = make([][]Cell, rows)
	for row := range std.buf {
		std.buf[row] = make([]Cell, cols)
	}
	std.rows = rows
	std.cols = cols
}

func (std *stdSurface) Move(col int, row int) {
	// No-op. Can't move the stdsurface
}

// Set a cell at col, row
func (std *stdSurface) SetCell(col int, row int, cell Cell) {
	std.mu.Lock()
	defer std.mu.Unlock()
	if col < 0 || row < 0 {
		return
	}
	if col >= std.cols {
		return
	}
	if row >= std.rows {
		return
	}
	std.buf[row][col] = cell
}

func (std *stdSurface) ShowCursor(col int, row int) {
	std.rtk.ShowCursor(col, row)
}
