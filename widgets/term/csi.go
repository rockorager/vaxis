package term

import (
	"slices"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

// Returns a single parameter from a slice of parameters, or 0 if the slice is
// empty
func ps(seq ansi.CSI) int {
	return seq.Param(0)
}

// Insert Blank Character (ICH) CSI Ps @
// Insert Ps blank characters. Cursor does not change position.
func (vt *Model) ich(ps int) {
	origCol := vt.cursor.col
	vt.resetPendingWrap()
	if ps <= 0 {
		return
	}
	col := vt.cursor.col
	if origCol < vt.margin.left || origCol > vt.margin.right {
		return
	}
	row := vt.cursor.row
	if remaining := int(vt.margin.right-col) + 1; ps > remaining {
		ps = remaining
	}
	vt.eraseWideAt(row, col, vt.cursor.Style.Background, false)
	vt.eraseWideHeadAt(row, vt.margin.right, vt.cursor.Style.Background, false)
	line := vt.activeScreen.line(row)
	for i := vt.margin.right; i >= col+column(ps); i -= 1 {
		line[i] = line[i-column(ps)]
	}
	for i := 0; i < ps; i += 1 {
		line[col+column(i)].erase(vt.cursor.Style.Background)
	}
	vt.eraseWideOverflow(row, col, vt.margin.right, vt.cursor.Style.Background)
}

// Cursur Up (CUU) CSI Ps A
// Move cursor up in same column, stopping at top margin
func (vt *Model) cuu(ps int) {
	vt.resetPendingWrap()
	if ps == 0 {
		ps = 1
	}
	clamp := row(0)
	if vt.cursor.row >= vt.margin.top {
		clamp = vt.margin.top
	}
	vt.cursor.row -= row(ps)
	if vt.cursor.row < clamp {
		vt.cursor.row = clamp
	}
}

// Cursur Down (CUD) CSI Ps B
// Move cursor down in same column, stopping at bottom margin
func (vt *Model) cud(ps int) {
	vt.resetPendingWrap()
	if ps == 0 {
		ps = 1
	}
	max := row(vt.height()) - vt.cursor.row - 1
	if vt.cursor.row <= vt.margin.bottom {
		max = vt.margin.bottom - vt.cursor.row
	}
	if row(ps) > max {
		ps = int(max)
	}
	vt.cursor.row += row(ps)
}

// Cursur Forward (CUF) CSI Ps C
// Move cursor forward Ps columns, stopping at the right margin
func (vt *Model) cuf(ps int) {
	vt.resetPendingWrap()
	if ps == 0 {
		ps = 1
	}
	max := column(vt.width()) - vt.cursor.col - 1
	if vt.cursor.col <= vt.margin.right {
		max = vt.margin.right - vt.cursor.col
	}
	if column(ps) > max {
		ps = int(max)
	}
	vt.cursor.col += column(ps)
}

// Cursur Backward (CUB) CSI Ps D
// Move cursor backward Ps columns, stopping at the left edge
func (vt *Model) cub(ps int) {
	if ps == 0 {
		ps = 1
	}
	count := ps
	reverseWrapExtended := vt.mode.decawm && vt.mode.reverseWrapExtended
	reverseWrap := vt.mode.decawm && (vt.mode.reverseWrap || reverseWrapExtended)
	if !reverseWrap {
		vt.resetPendingWrap()
		if column(count) > vt.cursor.col {
			count = int(vt.cursor.col)
		}
		vt.cursor.col -= column(count)
		return
	}

	if vt.lastCol {
		count--
	}
	vt.resetPendingWrap()
	if count == 0 {
		return
	}

	left := column(0)
	if vt.cursor.col >= vt.margin.left {
		left = vt.margin.left
	}
	if vt.cursor.col == left && !reverseWrapExtended && vt.cursor.row <= vt.margin.top {
		vt.cursor.row = vt.margin.top
		vt.cursor.col = left
		return
	}

	for count > 0 {
		max := int(vt.cursor.col - left)
		amount := min(count, max)
		vt.cursor.col -= column(amount)
		count -= amount
		if count == 0 {
			return
		}
		if vt.cursor.row == vt.margin.top {
			if !reverseWrapExtended {
				return
			}
			vt.cursor.row = vt.margin.bottom
			vt.cursor.col = vt.margin.right
			count--
			continue
		}
		if vt.cursor.row == 0 {
			return
		}
		if !reverseWrapExtended && !vt.activeScreen.row(vt.cursor.row-1).wrapped {
			return
		}
		vt.cursor.row--
		vt.cursor.col = vt.margin.right
		count--
	}
}

// Cursor Next Line (CNL) CSI Ps E
// Move cursor to left margin Ps lines down without scrolling
func (vt *Model) cnl(ps int) {
	vt.cud(ps)
	vt.cr()
}

// Cursor Preceding Line (CPL) CSI Ps F
// Move cursor to left margin Ps lines up without scrolling
func (vt *Model) cpl(ps int) {
	vt.cuu(ps)
	vt.cr()
}

// Cursor Character Absolute (CHA) CSI Ps G
// Move cursor to Ps column, stopping at right/left margin. Default is 1, but we
// default to 0 since our columns our 0 indexed
func (vt *Model) cha(ps int) {
	vt.setCursorPos(int(vt.cursor.row)+1, ps)
}

// Cursor Position (CUP) CSI Ps;Ps H
// Move cursor to the absolute position
func (vt *Model) cup(pm ansi.CSI) {
	r := 1
	c := 1
	switch pm.NumParameters {
	case 0:
	case 1:
		r = pm.Param(0)
	default:
		r = pm.Param(0)
		c = pm.Param(1)
	}
	vt.setCursorPos(r, c)
}

func (vt *Model) setCursorPos(rowReq int, colReq int) {
	vt.resetPendingWrap()
	if rowReq <= 0 {
		rowReq = 1
	}
	if colReq <= 0 {
		colReq = 1
	}

	xOffset := 0
	yOffset := 0
	xMax := vt.width()
	yMax := vt.height()
	if vt.mode.decom {
		xOffset = int(vt.margin.left)
		yOffset = int(vt.margin.top)
		xMax = int(vt.margin.right) + 1
		yMax = int(vt.margin.bottom) + 1
	}
	if colReq+xOffset > xMax {
		colReq = xMax - xOffset
	}
	if rowReq+yOffset > yMax {
		rowReq = yMax - yOffset
	}
	vt.cursor.col = column(colReq + xOffset - 1)
	vt.cursor.row = row(rowReq + yOffset - 1)
}

// Cursor Forward Tabulation (CHT) CSI Ps I
// Move cursor forward Ps tab stops
func (vt *Model) cht(ps int) {
	if ps <= 0 {
		return
	}
	if vt.cursor.col >= vt.margin.right {
		return
	}
	newcol, n := vt.margin.right, len(vt.tabStop)
	if i, found := slices.BinarySearch(vt.tabStop, vt.cursor.col); i < n {
		i += ps
		if !found {
			i--
		} // "i" was already 1 TS past the cursor.
		if i < n {
			newcol = min(vt.tabStop[i], vt.margin.right)
		}
	}
	vt.cursor.col = newcol
}

// Erase in Display (ED) CSI Ps J and DECSED CSI ? Ps J
func (vt *Model) ed(ps int, forceProtected bool) {
	protect := forceProtected || vt.mode.protected == protectedModeISO
	switch ps {

	// Erases from the cursor to the end of the screen, including the cursor
	// position. Line attribute becomes single-height, single-width for all
	// completely erased lines.
	case 0:
		vt.el(0, forceProtected)
		for r := vt.cursor.row + 1; r < row(vt.height()); r += 1 {
			vt.activeScreen.eraseRowProtected(r, 0, column(vt.width()-1), vt.cursor.Style.Background, protect)
		}

	// Erases from the beginning of the screen to the cursor, including the
	// cursor position. Line attribute becomes single-height, single-width
	// for all completely erased lines.
	case 1:
		vt.el(1, forceProtected)
		for r := row(0); r < vt.cursor.row; r += 1 {
			vt.activeScreen.eraseRowProtected(r, 0, column(vt.width()-1), vt.cursor.Style.Background, protect)
		}

	// Erases the complete display. All lines are erased and changed to
	// single-width. The cursor does not move.
	case 2:
		vt.clearSelectionLocked()
		if vt.shouldScrollClearAtSemanticPrompt() {
			vt.activeScreen.scrollClear(vt.cursor.Style.Background)
		}
		vt.graphics = nil
		vt.resetPendingWrap()
		for r := row(0); r < row(vt.height()); r += 1 {
			vt.activeScreen.eraseRowProtected(r, 0, column(vt.width()-1), vt.cursor.Style.Background, protect)
		}

	// Erases saved lines in the scrollback buffer.
	case 3:
		vt.clearSelectionLocked()
		vt.activeScreen.clearScrollback()
		vt.clampScrollOffset()

	// Erases the complete display and scrollback.
	case 22:
		vt.clearSelectionLocked()
		vt.activeScreen.clearScrollback()
		vt.clampScrollOffset()
		vt.graphics = nil
		vt.resetPendingWrap()
		for r := row(0); r < row(vt.height()); r += 1 {
			vt.activeScreen.eraseRowProtected(r, 0, column(vt.width()-1), vt.cursor.Style.Background, protect)
		}
	}
}

func (vt *Model) shouldScrollClearAtSemanticPrompt() bool {
	if vt.activeScreen.state == nil || vt.activeScreen.state != vt.primaryScreen.state {
		return false
	}
	for r := row(vt.height() - 1); r >= 0; r -= 1 {
		if trimTrailingBlankCells(vt.activeScreen.line(r)) == 0 {
			continue
		}
		return vt.activeScreen.row(r).semanticPrompt != semanticPromptNone
	}
	return false
}

// Erase in Line (EL) CSI Ps K and DECSEL CSI ? Ps K
func (vt *Model) el(ps int, forceProtected bool) {
	protect := forceProtected || vt.mode.protected == protectedModeISO
	r := vt.cursor.row
	switch ps {
	// Erases from the cursor to the end of the line, including the cursor
	// position. Line attribute is not affected.
	case 0:
		vt.resetWrap()
		vt.eraseWideAt(r, vt.cursor.col, vt.cursor.Style.Background, protect)
		for col := vt.cursor.col; col < column(vt.width()); col += 1 {
			vt.activeScreen.eraseCellProtected(r, col, vt.cursor.Style.Background, protect)
		}

	// Erases from the beginning of the line to the cursor, including the
	// cursor position. Line attribute is not affected.
	case 1:
		vt.resetPendingWrap()
		vt.eraseWideAt(r, vt.cursor.col, vt.cursor.Style.Background, protect)
		for col := column(0); col <= vt.cursor.col; col += 1 {
			vt.activeScreen.eraseCellProtected(r, col, vt.cursor.Style.Background, protect)
		}

	// Erases the complete line.
	case 2:
		vt.resetPendingWrap()
		for col := column(0); col < column(vt.width()); col += 1 {
			vt.activeScreen.eraseCellProtected(r, col, vt.cursor.Style.Background, protect)
		}
	}
}

// Insert Lines (IL) CSI Ps L
//
// Insert Ps lines at the cursor. If fewer than Ps lines remain from the current
// line to the end of the scrolling region, the number of lines inserted is the
// lesser number. Lines within the scrolling region at and below the cursor move
// down. Lines moved past the bottom margin are lost. The cursor is reset to the
// first column. This sequence is ignored when the cursor is outside the
// scrolling region.
func (vt *Model) il(ps int) {
	if ps <= 0 {
		return
	}
	if vt.cursor.row < vt.margin.top {
		return
	}
	if vt.cursor.row > vt.margin.bottom {
		return
	}
	if vt.cursor.col < vt.margin.left {
		return
	}
	if vt.cursor.col > vt.margin.right {
		return
	}
	vt.resetWrap()

	if available := int(vt.margin.bottom-vt.cursor.row) + 1; ps > available {
		ps = available
	}

	// move the lines first
	for r := vt.margin.bottom; r >= (vt.cursor.row + row(ps)); r -= 1 {
		vt.activeScreen.copyRowRange(r, r-row(ps), vt.margin.left, vt.margin.right)
		vt.activeScreen.repairWideRangeBoundaries(r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
	}

	// insert the blank lines (we do this by erasing the cells)
	for r := row(0); r < row(ps); r += 1 {
		vt.activeScreen.eraseRow(vt.cursor.row+r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
		vt.activeScreen.repairWideRangeBoundaries(vt.cursor.row+r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
	}
	vt.cursor.col = vt.margin.left
}

// Delete Line (DL) CSI Ps M
//
// Deletes Ps lines starting at the line with the cursor. If fewer than Ps lines
// remain from the current line to the end of the scrolling region, the number
// of lines deleted is the lesser number. As lines are deleted, lines within the
// scrolling region and below the cursor move up, and blank lines are added at
// the bottom of the scrolling region. The cursor is reset to the first column.
// This sequence is ignored when the cursor is outside the scrolling region.
func (vt *Model) dl(ps int) {
	if ps <= 0 {
		return
	}
	if vt.cursor.row < vt.margin.top {
		return
	}
	if vt.cursor.row > vt.margin.bottom {
		return
	}
	if vt.cursor.col < vt.margin.left {
		return
	}
	if vt.cursor.col > vt.margin.right {
		return
	}
	vt.resetWrap()

	if available := int(vt.margin.bottom-vt.cursor.row) + 1; ps > available {
		ps = available
	}

	for r := vt.cursor.row; r <= vt.margin.bottom; r += 1 {
		if r <= vt.margin.bottom-row(ps) {
			vt.activeScreen.copyRowRange(r, r+row(ps), vt.margin.left, vt.margin.right)
			vt.activeScreen.repairWideRangeBoundaries(r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
			if vt.margin.left == 0 && vt.margin.right >= column(vt.width()-1) {
				vt.activeScreen.repairEmptySoftWrap(r)
			}
			continue
		}
		vt.activeScreen.eraseRow(r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
		vt.activeScreen.repairWideRangeBoundaries(r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
		if vt.margin.left == 0 && vt.margin.right >= column(vt.width()-1) {
			vt.activeScreen.repairEmptySoftWrap(r)
		}
	}
	vt.cursor.col = vt.margin.left
}

// Delete Characters (DCH) CSI Ps P
//
// Deletes Ps characters starting with the character at the cursor position.
// When a character is deleted, all characters to the right of the cursor move
// to the left. This creates a space character at the right margin for each
// character deleted. Character attributes move with the characters. The spaces
// created at the end of the line have all their character attributes off.
func (vt *Model) dch(ps int) {
	if ps <= 0 {
		return
	}
	origCol := vt.cursor.col
	if origCol < vt.margin.left || origCol > vt.margin.right {
		return
	}
	vt.resetWrap()
	row := vt.cursor.row
	vt.eraseWideAt(row, vt.cursor.col, vt.cursor.Style.Background, false)
	vt.eraseWideHeadAt(row, vt.margin.right, vt.cursor.Style.Background, false)
	for col := vt.cursor.col; col <= vt.margin.right; col += 1 {
		if col+column(ps) > vt.margin.right {
			vt.activeScreen.eraseCell(row, col, vt.cursor.Style.Background)
			continue
		}
		vt.activeScreen.setCell(row, col, *vt.activeScreen.cell(row, col+column(ps)))
	}
	vt.eraseWideOverflow(row, vt.cursor.col, vt.margin.right, vt.cursor.Style.Background)
}

// Erase Characters (ECH) CSI Ps X
//
// Erases characters at the cursor position and the next Ps-1 characters. A
// parameter of 0 or 1 erases a single character. Character attributes are set
// to normal. No reformatting of data on the line occurs. The cursor remains in
// the same position.
func (vt *Model) ech(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}

	protect := vt.mode.protected == protectedModeISO
	vt.eraseWideAt(vt.cursor.row, vt.cursor.col, vt.cursor.Style.Background, protect)
	vt.eraseWideAt(vt.cursor.row, vt.cursor.col+column(ps)-1, vt.cursor.Style.Background, protect)
	for i := column(0); i < column(ps); i += 1 {
		if vt.cursor.col+i == column(vt.width()) {
			return
		}
		vt.activeScreen.eraseCellProtected(vt.cursor.row, vt.cursor.col+i, vt.cursor.Style.Background, protect)
	}
}

func (vt *Model) eraseWideAt(r row, c column, bg vaxis.Color, protect bool) {
	if c < 0 || c >= column(vt.width()) {
		return
	}
	line := vt.activeScreen.line(r)
	if line[c].Width > 1 {
		vt.eraseWideHead(r, c, bg, protect)
		return
	}
	for head := c - 1; head >= 0; head -= 1 {
		width := line[head].Width
		if width <= 1 {
			continue
		}
		if head+column(width) > c {
			vt.eraseWideHead(r, head, bg, protect)
		}
		return
	}
}

func (vt *Model) eraseWideHeadAt(r row, c column, bg vaxis.Color, protect bool) {
	if c < 0 || c >= column(vt.width()) {
		return
	}
	if vt.activeScreen.cell(r, c).Width > 1 {
		vt.eraseWideHead(r, c, bg, protect)
	}
}

func (vt *Model) eraseWideHead(r row, head column, bg vaxis.Color, protect bool) {
	width := vt.activeScreen.cell(r, head).Width
	for i := column(0); i < column(width) && head+i < column(vt.width()); i += 1 {
		vt.activeScreen.eraseCellProtected(r, head+i, bg, protect)
	}
}

func (vt *Model) eraseWideOverflow(r row, left column, right column, bg vaxis.Color) {
	line := vt.activeScreen.line(r)
	for col := left; col <= right; col += 1 {
		width := line[col].Width
		if width > 1 && col+column(width)-1 > right {
			vt.eraseWideHead(r, col, bg, false)
		}
	}
}

func (vt *Model) decsca(seq ansi.CSI) {
	if seq.NumParameters > 1 {
		return
	}
	switch ps(seq) {
	case 0, 2:
		vt.setProtectedMode(protectedModeOff)
	case 1:
		vt.setProtectedMode(protectedModeDEC)
	}
}

func (vt *Model) decsasd(seq ansi.CSI) {
	if seq.NumParameters != 1 {
		return
	}
	switch ps(seq) {
	case 0:
		vt.status = statusDisplayMain
	case 1:
		vt.status = statusDisplayLine
	}
}

// Cursor Backward Tabulation (CBT) CSI Ps Z
//
// Move cursor backward Ps tabulations
func (vt *Model) cbt(ps int) {
	if ps <= 0 {
		return
	}
	leftLimit := column(0)
	if vt.mode.decom && vt.cursor.col >= vt.margin.left {
		leftLimit = vt.margin.left
	}
	newcol := leftLimit
	if i, _ := slices.BinarySearch(vt.tabStop, vt.cursor.col); i > 0 {
		i -= ps
		if i >= 0 {
			newcol = max(vt.tabStop[i], leftLimit)
		}
	}
	vt.cursor.col = newcol
}

// Tab Clear (TBC) CSI Ps g
func (vt *Model) tbc(ps int) {
	switch ps {
	case 0:
		if i, found := slices.BinarySearch(vt.tabStop, vt.cursor.col); found {
			vt.tabStop = slices.Delete(vt.tabStop, i, i+1)
		}
	case 3:
		vt.tabStop = []column{}
	}
}

func (vt *Model) ctc(seq ansi.CSI, private bool) {
	if private {
		if seq.NumParameters == 1 && ps(seq) == 5 {
			vt.setDefaultTabStops()
		}
		return
	}

	switch {
	case seq.NumParameters == 0:
		vt.hts()
	case seq.NumParameters == 1:
		switch ps(seq) {
		case 0:
			vt.hts()
		case 2:
			vt.tbc(0)
		case 5:
			vt.tbc(3)
		}
	}
}

// Line Position Absolute (VPA) CSI Ps d
//
// Move cursor to line Ps
func (vt *Model) vpa(ps int) {
	vt.setCursorPos(ps, int(vt.cursor.col)+1)
}

// Line Position Relative (VPR) CSI Ps e
//
// Move down Ps lines
func (vt *Model) vpr(ps int) {
	vt.setCursorPos(int(vt.cursor.row)+1+ps, int(vt.cursor.col)+1)
}

// Character Position Absolute (HPA) CSI Ps `
//
// Move cursor to column Ps
func (vt *Model) hpa(ps int) {
	vt.setCursorPos(int(vt.cursor.row)+1, ps)
}

// Character Position Relative (HPR) CSI Ps a
//
// Move cursor to the right Ps times
func (vt *Model) hpr(ps int) {
	vt.setCursorPos(int(vt.cursor.row)+1, int(vt.cursor.col)+1+ps)
}

// Repeat (REP) CSI Ps b
//
// Repeat preceding graphic character Ps times
func (vt *Model) rep(ps int) {
	if !vt.hasPreviousChar {
		return
	}
	if ps == 0 {
		ps = 1
	}
	for i := 0; i < ps; i += 1 {
		vt.print(ansi.Print{
			Grapheme: vt.previousChar.Grapheme,
			Width:    vt.previousChar.Width,
		})
	}
}

// Set top and bottom margins CSI Ps ; Ps r
func (vt *Model) decstbm(pm ansi.CSI) {
	if pm.NumParameters > 2 {
		return
	}
	topReq := 0
	botReq := 0
	switch pm.NumParameters {
	case 0:
	case 1:
		topReq = pm.Param(0)
	case 2:
		topReq = pm.Param(0)
		botReq = pm.Param(1)
	}
	top := max(1, topReq)
	bot := vt.height()
	if botReq > 0 {
		bot = min(vt.height(), botReq)
	}
	if top >= bot {
		return
	}
	vt.margin.top = row(top - 1)
	vt.margin.bottom = row(bot - 1)
	vt.setCursorPos(1, 1)
}

// Set left and right margins (DECSLRM) CSI Ps ; Ps s
func (vt *Model) decslrm(pm ansi.CSI) {
	if !vt.mode.declrmm {
		return
	}
	if pm.NumParameters > 2 {
		return
	}
	leftReq := 0
	rightReq := 0
	switch pm.NumParameters {
	case 0:
	case 1:
		leftReq = pm.Param(0)
	case 2:
		leftReq = pm.Param(0)
		rightReq = pm.Param(1)
	}
	left := max(1, leftReq)
	right := vt.width()
	if rightReq > 0 {
		right = min(vt.width(), rightReq)
	}
	if left >= right {
		return
	}
	vt.margin.left = column(left - 1)
	vt.margin.right = column(right - 1)
	vt.setCursorPos(1, 1)
}
