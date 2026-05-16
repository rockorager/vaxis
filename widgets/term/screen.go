package term

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
)

const (
	defaultScrollbackLines = 10000
	screenPageRows         = 64
)

type screenBuffer struct {
	state *screenState
}

type screenState struct {
	width           int
	height          int
	cells           []cell
	rows            []screenRow
	scrollback      screenHistory
	scrollbackLimit int
}

type screenRow struct {
	wrapped          bool
	wrapContinuation bool
}

type screenLine struct {
	row   screenRow
	cells []cell
}

type screenHistory struct {
	pages []*screenPage
	len   int
}

type screenPage struct {
	width int
	rows  []screenRow
	cells []cell
	start int
	len   int
}

func newScreenPage(width int) *screenPage {
	return &screenPage{
		width: width,
		rows:  make([]screenRow, screenPageRows),
		cells: make([]cell, width*screenPageRows),
	}
}

func (p *screenPage) full() bool {
	return p.start+p.len >= screenPageRows
}

func (p *screenPage) line(i int) []cell {
	start := i * p.width
	return p.cells[start : start+p.width]
}

func (h *screenHistory) append(line []cell, row screenRow, limit int) {
	if limit <= 0 {
		return
	}
	if len(h.pages) == 0 || h.pages[len(h.pages)-1].full() {
		h.pages = append(h.pages, newScreenPage(len(line)))
	}
	page := h.pages[len(h.pages)-1]
	dst := page.start + page.len
	page.rows[dst] = row
	copy(page.line(dst), line)
	page.len += 1
	h.len += 1
	h.trim(limit)
}

func (h *screenHistory) trim(limit int) {
	for h.len > limit && len(h.pages) > 0 {
		page := h.pages[0]
		drop := h.len - limit
		if drop >= page.len {
			h.len -= page.len
			h.pages = h.pages[1:]
			continue
		}
		page.start += drop
		page.len -= drop
		h.len -= drop
	}
}

func (h screenHistory) line(i int) (screenLine, bool) {
	if i < 0 || i >= h.len {
		return screenLine{}, false
	}
	for _, page := range h.pages {
		if i >= page.len {
			i -= page.len
			continue
		}
		idx := page.start + i
		return screenLine{
			row:   page.rows[idx],
			cells: page.line(idx),
		}, true
	}
	return screenLine{}, false
}

func (h *screenHistory) clear() {
	h.pages = nil
	h.len = 0
}

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

func (s screenBuffer) copyRow(dst row, src row) {
	copy(s.line(dst), s.line(src))
	s.state.rows[dst] = s.state.rows[src]
}

func (s screenBuffer) eraseRow(r row, left column, right column, bg vaxis.Color) {
	s.eraseRowRange(r, left, right, bg)
	s.state.rows[r] = screenRow{}
}

func (s screenBuffer) offset(r row, c column) int {
	return int(r)*s.state.width + int(c)
}

func (s screenBuffer) scrollUp(top row, bottom row, left column, right column, n int, bg vaxis.Color) int {
	captured := s.captureScrollback(top, bottom, n)
	for r := 0; r < s.state.height; r += 1 {
		if r > int(bottom) {
			continue
		}
		if r < int(top) {
			continue
		}
		if r+n > int(bottom) {
			s.eraseRow(row(r), left, right, bg)
			continue
		}
		s.copyRow(row(r), row(r+n))
	}
	return captured
}

func (s screenBuffer) scrollDown(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	for r := bottom; r >= top; r -= 1 {
		if r-row(n) < top {
			s.eraseRow(r, left, right, bg)
			continue
		}
		s.copyRow(r, r-row(n))
	}
}

func (s screenBuffer) captureScrollback(top row, bottom row, n int) int {
	if s.state == nil || s.state.scrollbackLimit <= 0 {
		return 0
	}
	if top != 0 || bottom != row(s.height()-1) {
		return 0
	}
	if n <= 0 {
		return 0
	}
	if n > s.height() {
		n = s.height()
	}
	for i := 0; i < n; i += 1 {
		s.state.scrollback.append(s.line(row(i)), s.state.rows[i], s.state.scrollbackLimit)
	}
	return n
}

func (s screenBuffer) scrollbackLen() int {
	if s.state == nil {
		return 0
	}
	return s.state.scrollback.len
}

func (s screenBuffer) scrollbackString(i int) string {
	if s.state == nil {
		return ""
	}
	line, ok := s.scrollbackLine(i)
	if !ok {
		return ""
	}
	str := strings.Builder{}
	for col := range line.cells {
		_, _ = str.WriteString(line.cells[col].rune())
	}
	return str.String()
}

func (s screenBuffer) scrollbackRow(i int) screenRow {
	if s.state == nil {
		return screenRow{}
	}
	line, ok := s.scrollbackLine(i)
	if !ok {
		return screenRow{}
	}
	return line.row
}

func (s screenBuffer) scrollbackLine(i int) (screenLine, bool) {
	if s.state == nil {
		return screenLine{}, false
	}
	return s.state.scrollback.line(i)
}

func (s screenBuffer) clearScrollback() {
	if s.state == nil {
		return
	}
	s.state.scrollback.clear()
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
