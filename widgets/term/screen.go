package term

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
)

type screenBuffer [][]cell

func newScreenBuffer(width int, height int) screenBuffer {
	screen := make(screenBuffer, height)
	for row := range screen {
		screen[row] = make([]cell, width)
	}
	return screen
}

func (s screenBuffer) width() int {
	if len(s) == 0 {
		return 0
	}
	return len(s[0])
}

func (s screenBuffer) height() int {
	return len(s)
}

func (s screenBuffer) line(r row) []cell {
	return s[r]
}

func (s screenBuffer) cell(r row, c column) *cell {
	return &s[r][c]
}

func (s screenBuffer) eraseCell(r row, c column, bg vaxis.Color) {
	s[r][c].erase(bg)
}

func (s screenBuffer) eraseRowRange(r row, left column, right column, bg vaxis.Color) {
	for col := left; col <= right; col += 1 {
		s[r][col].erase(bg)
	}
}

func (s screenBuffer) scrollUp(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	for r := range s {
		if r > int(bottom) {
			continue
		}
		if r < int(top) {
			continue
		}
		if r+n > int(bottom) {
			s.eraseRowRange(row(r), left, right, bg)
			continue
		}
		copy(s[r], s[r+n])
	}
}

func (s screenBuffer) scrollDown(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	for r := bottom; r >= top; r -= 1 {
		if r-row(n) < top {
			s.eraseRowRange(r, left, right, bg)
			continue
		}
		copy(s[r], s[r-row(n)])
	}
}

func (s screenBuffer) String() string {
	str := strings.Builder{}
	for row := range s {
		for col := range s[row] {
			_, _ = str.WriteString(s[row][col].rune())
		}
		if row < s.height()-1 {
			str.WriteRune('\n')
		}
	}
	return str.String()
}
