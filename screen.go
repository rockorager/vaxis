package vaxis

type screen struct {
	buf  []Cell
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
	s.buf = make([]Cell, rows*cols)
	s.rows = rows
	s.cols = cols
}

func (s *screen) index(col int, row int) int {
	return row*s.cols + col
}

func (s *screen) row(row int) []Cell {
	start := row * s.cols
	return s.buf[start : start+s.cols]
}

func (s *screen) cell(col int, row int) Cell {
	return s.buf[s.index(col, row)]
}

func (s *screen) setCellDirect(col int, row int, text Cell) {
	s.buf[s.index(col, row)] = text
}

func (s *screen) setStyleDirect(col int, row int, style Style) {
	s.buf[s.index(col, row)].Style = style
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
	s.setCellDirect(col, row, text)
}
