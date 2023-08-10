package vaxis

type screen struct {
	buf  [][]Text
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
	s.buf = make([][]Text, rows)
	for row := range s.buf {
		s.buf[row] = make([]Text, cols)
	}
	s.rows = rows
	s.cols = cols
}

// Set a cell at col, row
func (s *screen) setCell(col int, row int, text Text) int {
	if col < 0 || row < 0 {
		return 0
	}
	if col >= s.cols {
		return 0
	}
	if row >= s.rows {
		return 0
	}
	s.buf[row][col] = text
	return characterWidth(text.Content)
}
