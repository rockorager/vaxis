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

func (h *screenHistory) truncateLast(n int) {
	for n > 0 && len(h.pages) > 0 {
		page := h.pages[len(h.pages)-1]
		drop := min(n, page.len)
		page.len -= drop
		h.len -= drop
		n -= drop
		if page.len == 0 {
			h.pages = h.pages[:len(h.pages)-1]
		}
	}
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

func (s screenBuffer) eraseCellProtected(r row, c column, bg vaxis.Color, protect bool) {
	cell := &s.state.cells[s.offset(r, c)]
	if protect && cell.protected {
		return
	}
	cell.erase(bg)
}

func (s screenBuffer) eraseRowRange(r row, left column, right column, bg vaxis.Color) {
	line := s.line(r)
	for col := left; col <= right; col += 1 {
		line[col].erase(bg)
	}
}

func (s screenBuffer) eraseRowRangeProtected(r row, left column, right column, bg vaxis.Color, protect bool) bool {
	line := s.line(r)
	skipped := false
	for col := left; col <= right; col += 1 {
		if protect && line[col].protected {
			skipped = true
			continue
		}
		line[col].erase(bg)
	}
	return skipped
}

func (s screenBuffer) copyRow(dst row, src row) {
	copy(s.line(dst), s.line(src))
	s.state.rows[dst] = s.state.rows[src]
}

func (s screenBuffer) copyRowRange(dst row, src row, left column, right column) {
	if left == 0 && right >= column(s.width()-1) {
		s.copyRow(dst, src)
		return
	}
	dstLine := s.line(dst)
	srcLine := s.line(src)
	copy(dstLine[left:right+1], srcLine[left:right+1])
}

func (s screenBuffer) copyRowFrom(dst row, src screenBuffer, srcRow row) {
	copy(s.line(dst), src.line(srcRow))
	s.state.rows[dst] = src.state.rows[srcRow]
}

func (s screenBuffer) eraseRow(r row, left column, right column, bg vaxis.Color) {
	s.eraseRowRange(r, left, right, bg)
	if left == 0 && right >= column(s.width()-1) {
		s.state.rows[r] = screenRow{}
	}
}

func (s screenBuffer) eraseRowProtected(r row, left column, right column, bg vaxis.Color, protect bool) {
	skipped := s.eraseRowRangeProtected(r, left, right, bg, protect)
	if !skipped && left == 0 && right >= column(s.width()-1) {
		s.state.rows[r] = screenRow{}
	}
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
		s.copyRowRange(row(r), row(r+n), left, right)
	}
	return captured
}

func (s screenBuffer) scrollDown(top row, bottom row, left column, right column, n int, bg vaxis.Color) {
	for r := bottom; r >= top; r -= 1 {
		if r-row(n) < top {
			s.eraseRow(r, left, right, bg)
			continue
		}
		s.copyRowRange(r, r-row(n), left, right)
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

func (s screenBuffer) resizeHeight(newHeight int, bg vaxis.Color) (screenBuffer, int, bool) {
	if s.state == nil || s.width() == 0 || s.height() == 0 {
		return screenBuffer{}, 0, false
	}
	if newHeight <= 0 {
		return screenBuffer{}, 0, false
	}
	next := newScreenBuffer(s.width(), newHeight, s.state.scrollbackLimit)
	next.state.scrollback = s.state.scrollback
	switch {
	case newHeight > s.height():
		grow := newHeight - s.height()
		pulled := min(grow, next.scrollbackLen())
		historyStart := next.scrollbackLen() - pulled
		for r := 0; r < pulled; r += 1 {
			line, ok := next.scrollbackLine(historyStart + r)
			if !ok {
				continue
			}
			dst := row(r)
			copy(next.line(dst), line.cells)
			next.state.rows[dst] = line.row
		}
		next.state.scrollback.truncateLast(pulled)
		for r := 0; r < s.height(); r += 1 {
			next.copyRowFrom(row(pulled+r), s, row(r))
		}
		for r := pulled + s.height(); r < newHeight; r += 1 {
			next.eraseRow(row(r), 0, column(next.width()-1), bg)
		}
		return next, pulled, true
	case newHeight < s.height():
		drop := s.height() - newHeight
		for r := 0; r < drop; r += 1 {
			next.state.scrollback.append(s.line(row(r)), s.state.rows[r], next.state.scrollbackLimit)
		}
		for r := 0; r < newHeight; r += 1 {
			next.copyRowFrom(row(r), s, row(drop+r))
		}
		return next, -drop, true
	default:
		for r := 0; r < s.height(); r += 1 {
			next.copyRowFrom(row(r), s, row(r))
		}
		return next, 0, true
	}
}

func (s screenBuffer) resizeReflow(newWidth int, newHeight int, bg vaxis.Color) (screenBuffer, bool) {
	next, _, _, ok := s.resizeReflowCursor(newWidth, newHeight, bg, 0, 0, false)
	return next, ok
}

func (s screenBuffer) resizeNoReflow(newWidth int, newHeight int, bg vaxis.Color) (screenBuffer, bool) {
	next, _, _, ok := s.resizeNoReflowCursor(newWidth, newHeight, bg, 0, 0, false)
	return next, ok
}

func (s screenBuffer) resizeNoReflowCursor(newWidth int, newHeight int, bg vaxis.Color, cursorRow row, cursorCol column, mapCursor bool) (screenBuffer, row, column, bool) {
	if s.state == nil || s.width() == 0 || s.height() == 0 {
		return screenBuffer{}, 0, 0, false
	}
	if newWidth <= 0 || newHeight <= 0 {
		return screenBuffer{}, 0, 0, false
	}
	next := newScreenBuffer(newWidth, newHeight, s.state.scrollbackLimit)
	var rows []screenLine
	for i := 0; i < s.scrollbackLen(); i += 1 {
		line, ok := s.scrollbackLine(i)
		if ok {
			rows = append(rows, resizePhysicalLine(line, newWidth))
		}
	}
	for r := 0; r < s.height(); r += 1 {
		rows = append(rows, resizePhysicalLine(screenLine{
			row:   s.state.rows[r],
			cells: s.line(row(r)),
		}, newWidth))
	}
	overflow := len(rows) - newHeight
	if overflow < 0 {
		overflow = 0
	}
	for i := 0; i < overflow; i += 1 {
		next.state.scrollback.append(rows[i].cells, rows[i].row, next.state.scrollbackLimit)
	}
	for r := 0; r < newHeight; r += 1 {
		src := overflow + r
		if src >= len(rows) {
			next.eraseRow(row(r), 0, column(newWidth-1), bg)
			continue
		}
		copy(next.line(row(r)), rows[src].cells)
		next.state.rows[r] = rows[src].row
	}
	if !mapCursor {
		return next, 0, 0, true
	}
	return next, row(s.scrollbackLen() + int(cursorRow) - overflow), column(min(int(cursorCol), newWidth-1)), true
}

func resizePhysicalLine(line screenLine, width int) screenLine {
	cells := make([]cell, width)
	copy(cells, line.cells)
	return screenLine{
		row:   line.row,
		cells: cells,
	}
}

func (s screenBuffer) resizeReflowCursor(newWidth int, newHeight int, bg vaxis.Color, cursorRow row, cursorCol column, mapCursor bool) (screenBuffer, row, column, bool) {
	if s.state == nil || s.width() == 0 || s.height() == 0 {
		return screenBuffer{}, 0, 0, false
	}
	if newWidth <= 0 || newHeight <= 0 {
		return screenBuffer{}, 0, 0, false
	}
	next := newScreenBuffer(newWidth, newHeight, s.state.scrollbackLimit)
	cursorSourceRow := -1
	if mapCursor {
		cursorSourceRow = s.scrollbackLen() + int(cursorRow)
	}
	rows, cursorReflowRow, cursorReflowCol, cursorOK := s.reflowRows(newWidth, cursorSourceRow, int(cursorCol))
	overflow := len(rows) - newHeight
	if overflow < 0 {
		overflow = 0
	}
	for i := 0; i < overflow; i += 1 {
		next.state.scrollback.append(rows[i].cells, rows[i].row, next.state.scrollbackLimit)
	}
	for r := 0; r < newHeight; r += 1 {
		src := overflow + r
		if src >= len(rows) {
			next.eraseRow(row(r), 0, column(newWidth-1), bg)
			continue
		}
		copy(next.line(row(r)), rows[src].cells)
		next.state.rows[r] = rows[src].row
	}
	nextCursorRow := row(cursorReflowRow - overflow)
	if !cursorOK {
		nextCursorRow = cursorRow
		nextCursorCol := cursorCol
		return next, nextCursorRow, nextCursorCol, true
	}
	return next, nextCursorRow, column(cursorReflowCol), true
}

func (s screenBuffer) reflowRows(width int, cursorSourceRow int, cursorSourceCol int) ([]screenLine, int, int, bool) {
	var rows []screenLine
	var logical []cell
	cursorLogicalCol := -1
	cursorRow := 0
	cursorCol := 0
	cursorOK := false
	for i := 0; i < s.scrollbackLen(); i += 1 {
		line, ok := s.scrollbackLine(i)
		if !ok {
			continue
		}
		logical, rows, cursorLogicalCol, cursorRow, cursorCol, cursorOK = appendLineForReflow(
			logical, rows, line, width, i, cursorSourceRow, cursorSourceCol,
			cursorLogicalCol, cursorRow, cursorCol, cursorOK,
		)
	}
	last := s.height() - 1
	for last >= 0 && trimTrailingBlankCells(s.line(row(last))) == 0 &&
		!s.state.rows[last].wrapped && !s.state.rows[last].wrapContinuation {
		last -= 1
	}
	for r := 0; r <= last; r += 1 {
		sourceRow := s.scrollbackLen() + r
		logical, rows, cursorLogicalCol, cursorRow, cursorCol, cursorOK = appendLineForReflow(
			logical, rows, screenLine{
				row:   s.state.rows[r],
				cells: s.line(row(r)),
			}, width, sourceRow, cursorSourceRow, cursorSourceCol,
			cursorLogicalCol, cursorRow, cursorCol, cursorOK,
		)
	}
	if len(logical) > 0 {
		start := len(rows)
		rows = appendReflowedRows(rows, logical, width)
		if cursorLogicalCol >= 0 {
			cursorRow = start + cursorLogicalCol/width
			cursorCol = cursorLogicalCol % width
			cursorOK = true
		}
	}
	return rows, cursorRow, cursorCol, cursorOK
}

func appendLineForReflow(
	logical []cell,
	rows []screenLine,
	line screenLine,
	width int,
	sourceRow int,
	cursorSourceRow int,
	cursorSourceCol int,
	cursorLogicalCol int,
	cursorRow int,
	cursorCol int,
	cursorOK bool,
) ([]cell, []screenLine, int, int, int, bool) {
	end := len(line.cells)
	if !line.row.wrapped {
		end = trimTrailingBlankCells(line.cells)
	}
	if sourceRow == cursorSourceRow && cursorSourceCol >= end {
		end = min(len(line.cells), cursorSourceCol+1)
	}
	before := len(logical)
	logical = append(logical, line.cells[:end]...)
	if sourceRow == cursorSourceRow {
		cursorLogicalCol = before + min(cursorSourceCol, end)
	}
	if line.row.wrapped {
		return logical, rows, cursorLogicalCol, cursorRow, cursorCol, cursorOK
	}
	start := len(rows)
	rows = appendReflowedRows(rows, logical, width)
	if cursorLogicalCol >= 0 {
		cursorRow = start + cursorLogicalCol/width
		cursorCol = cursorLogicalCol % width
		cursorOK = true
	}
	return nil, rows, -1, cursorRow, cursorCol, cursorOK
}

func appendReflowedRows(rows []screenLine, logical []cell, width int) []screenLine {
	if len(logical) == 0 {
		return append(rows, screenLine{
			row:   screenRow{},
			cells: make([]cell, width),
		})
	}
	for len(logical) > 0 {
		n := min(width, len(logical))
		line := make([]cell, width)
		copy(line, logical[:n])
		logical = logical[n:]
		rows = append(rows, screenLine{
			row: screenRow{
				wrapped:          len(logical) > 0,
				wrapContinuation: len(rows) > 0 && rows[len(rows)-1].row.wrapped,
			},
			cells: line,
		})
	}
	return rows
}

func trimTrailingBlankCells(line []cell) int {
	end := len(line)
	for end > 0 {
		grapheme := line[end-1].Grapheme
		if grapheme != "" && grapheme != " " {
			break
		}
		end -= 1
	}
	return end
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
