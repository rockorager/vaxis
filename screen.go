package rtk

import (
	"sync"
)

type screen struct {
	buf  [][]Cell
	mu   sync.Mutex
	rows int
	cols int
}

func newScreen() *screen {
	std := &screen{}
	return std
}

func (s *screen) Size() (cols int, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cols, s.rows
}

// resize resizes the stdsurface based on a SIGWINCH
func (s *screen) resize(cols int, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = make([][]Cell, rows)
	for row := range s.buf {
		s.buf[row] = make([]Cell, cols)
	}
	s.rows = rows
	s.cols = cols
}

// Set a cell at col, row
func (s *screen) SetCell(col int, row int, cell Cell) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if col < 0 || row < 0 {
		return
	}
	if col >= s.cols {
		return
	}
	if row >= s.rows {
		return
	}
	s.buf[row][col] = cell
}
