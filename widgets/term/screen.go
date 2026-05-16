package term

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
)

const defaultScrollbackLines = 10000

type screenBuffer struct {
	state *screenState
}

type screenState struct {
	width           int
	height          int
	cells           []cell
	rows            []screenRow
	scrollback      []screenLine
	scrollbackLimit int
}

type screenRow struct {
	wrapped bool
}

type screenLine []cell

func newScreenBuffer(width int, height int, scrollbackLimit int) screenBuffer {
	return screenBuffer{
		state: &screenState{
			width:           width,
			height:          height,
			cells:           make([]cell, width*height),
			rows:            make([]screenRow, height),
			scrollbackLimit: scrollbackLimit,
		},
	}
}

func (s screenBuffer) width() int {
	if s.state == nil {
		return 0
	}
	return s.state.width
}

func (s screenBuffer) height() int {
	if s.state == nil {
		return 0
	}
	return s.state.height
}

func (s screenBuffer) line(r row) []cell {
	start := int(r) * s.state.width
	return s.state.cells[start : start+s.state.width]
}

func (s screenBuffer) row(r row) *screenRow {
	return &s.state.rows[r]
}

func (s screenBuffer) cell(r row, c column) *cell {
	return &s.state.cells[s.offset(r, c)]
}

func (s screenBuffer) setCell(r row, c column, cell cell) {
	s.state.cells[s.offset(r, c)] = cell
}

func (s screenBuffer) eraseCell(r row, c column, bg vaxis.Color) {
	s.state.cells[s.offset(r, c)].erase(bg)
}

func (s screenBuffer) eraseRowRange(r row, left column, right column, bg vaxis.Color) {
	line := s.line(r)
	for col := left; col <= right; col += 1 {
		line[col].erase(bg)
	}
}

func (s screenBuffer) offset(r row, c column) int {
	return int(r)*s.state.width + int(c)
}

func (s screenBuffer) scrollUp(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	s.captureScrollback(top, bottom, n)
	for r := 0; r < s.state.height; r += 1 {
		if r > int(bottom) {
			continue
		}
		if r < int(top) {
			continue
		}
		if r+n > int(bottom) {
			s.eraseRowRange(row(r), left, right, bg)
			s.state.rows[r] = screenRow{}
			continue
		}
		copy(s.line(row(r)), s.line(row(r+n)))
		s.state.rows[r] = s.state.rows[r+n]
	}
}

func (s screenBuffer) scrollDown(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	for r := bottom; r >= top; r -= 1 {
		if r-row(n) < top {
			s.eraseRowRange(r, left, right, bg)
			s.state.rows[r] = screenRow{}
			continue
		}
		copy(s.line(r), s.line(r-row(n)))
		s.state.rows[r] = s.state.rows[r-row(n)]
	}
}

func (s screenBuffer) captureScrollback(top row, bottom row, n int) {
	if s.state == nil || s.state.scrollbackLimit <= 0 {
		return
	}
	if top != 0 || bottom != row(s.height()-1) {
		return
	}
	if n <= 0 {
		return
	}
	if n > s.height() {
		n = s.height()
	}
	for i := 0; i < n; i += 1 {
		line := make(screenLine, s.width())
		copy(line, s.line(row(i)))
		s.state.scrollback = append(s.state.scrollback, line)
	}
	if extra := len(s.state.scrollback) - s.state.scrollbackLimit; extra > 0 {
		copy(s.state.scrollback, s.state.scrollback[extra:])
		s.state.scrollback = s.state.scrollback[:s.state.scrollbackLimit]
	}
}

func (s screenBuffer) scrollbackLen() int {
	if s.state == nil {
		return 0
	}
	return len(s.state.scrollback)
}

func (s screenBuffer) scrollbackString(i int) string {
	if s.state == nil || i < 0 || i >= len(s.state.scrollback) {
		return ""
	}
	str := strings.Builder{}
	for col := range s.state.scrollback[i] {
		_, _ = str.WriteString(s.state.scrollback[i][col].rune())
	}
	return str.String()
}

func (s screenBuffer) String() string {
	if s.state == nil {
		return ""
	}
	str := strings.Builder{}
	for r := 0; r < s.state.height; r += 1 {
		line := s.line(row(r))
		for col := range line {
			_, _ = str.WriteString(line[col].rune())
		}
		if r < s.height()-1 {
			str.WriteRune('\n')
		}
	}
	return str.String()
}
