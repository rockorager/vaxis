package vaxis

type screen struct {
	buf  [][]Cell
	rows int
	cols int
}

func newScreen() *screen {
	std := &screen{}
	return std
}

func (s *screen) size() (cols int, rows int) {
	return s.cols, s.rows
}

// resize resizes the stdsurface based on a SIGWINCH
func (s *screen) resize(cols int, rows int) {
	s.buf = make([][]Cell, rows)
	for row := range s.buf {
		s.buf[row] = make([]Cell, cols)
	}
	s.rows = rows
	s.cols = cols
}

// Set a cell at col, row
func (s *screen) setCell(col int, row int, text Cell) {
	if col < 0 || row < 0 {
		return
	}
	if col >= s.cols {
		return
	}
	if row >= s.rows {
		return
	}
	s.buf[row][col] = text
}
