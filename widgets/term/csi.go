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
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	col := vt.cursor.col
	row := vt.cursor.row
	line := vt.activeScreen.line(row)
	for i := vt.margin.right; i > col; i -= 1 {
		if (i - column(ps)) < 0 {
			continue
		}
		line[i] = line[i-column(ps)]
	}
	for i := 0; i < ps; i += 1 {
		if int(col)+i >= (vt.width() - 1) {
			break
		}
		line[col+column(i)] = cell{
			Cell: vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: " ",
					Width:    1,
				},
			},
		}
	}
}

// Cursur Up (CUU) CSI Ps A
// Move cursor up in same column, stopping at top margin
func (vt *Model) cuu(ps int) {
	vt.resetWrap()
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
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.row += row(ps)
	if vt.cursor.row > vt.margin.bottom {
		vt.cursor.row = vt.margin.bottom
	}
}

// Cursur Forward (CUF) CSI Ps C
// Move cursor forward Ps columns, stopping at the right margin
func (vt *Model) cuf(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.col += column(ps)
	if vt.cursor.col > vt.margin.right {
		vt.cursor.col = vt.margin.right
	}
}

// Cursur Backward (CUB) CSI Ps D
// Move cursor backward Ps columns, stopping at the left margin
func (vt *Model) cub(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.col -= column(ps)
	if vt.cursor.col < vt.margin.left {
		vt.cursor.col = vt.margin.left
	}
}

// Cursor Next Line (CNL) CSI Ps E
// Move cursor to left margin Ps lines down, scrolling if necessary
func (vt *Model) cnl(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	for i := 0; i < ps; i += 1 {
		vt.nel()
	}
}

// Cursor Preceding Line (CPL) CSI Ps F
// Move cursor to left margin Ps lines down, scrolling if necessary
func (vt *Model) cpl(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	for i := 0; i < ps; i += 1 {
		vt.ri()
	}
	vt.cursor.col = vt.margin.left
}

// Cursor Character Absolute (CHA) CSI Ps G
// Move cursor to Ps column, stopping at right/left margin. Default is 1, but we
// default to 0 since our columns our 0 indexed
func (vt *Model) cha(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.col = column(ps - 1)
	if vt.cursor.col > vt.margin.right {
		vt.cursor.col = vt.margin.right
	}
	if vt.cursor.col < vt.margin.left {
		vt.cursor.col = vt.margin.left
	}
}

// Cursor Position (CUP) CSI Ps;Ps H
// Move cursor to the absolute position
func (vt *Model) cup(pm ansi.CSI) {
	vt.resetWrap()
	switch pm.NumParameters {
	case 0:
		vt.cursor.row = 0
		vt.cursor.col = 0
	case 1:
		vt.cursor.row = row(pm.Param(0) - 1)
		vt.cursor.col = 0
	default:
		vt.cursor.row = row(pm.Param(0) - 1)
		vt.cursor.col = column(pm.Param(1) - 1)
	}
	if vt.cursor.col > column(vt.width()-1) {
		vt.cursor.col = column(vt.width() - 1)
	}
	if vt.cursor.row > row(vt.height()-1) {
		vt.cursor.row = row(vt.height() - 1)
	}
}

// Cursor Forward Tabulation (CHT) CSI Ps I
// Move cursor forward Ps tab stops
func (vt *Model) cht(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	// Note: Actually, should stop at the margin only when DECLRMM and
	// DECOM (methinks) are set.  However, currently we implement neither
	// and this "margin" is just the absolute boundary of the window.
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

// Erase in Display (ED) CSI Ps J
func (vt *Model) ed(ps int) {
	switch ps {

	// Erases from the cursor to the end of the screen, including the cursor
	// position. Line attribute becomes single-height, single-width for all
	// completely erased lines.
	case 0:
		vt.el(0)
		for r := vt.cursor.row + 1; r < row(vt.height()); r += 1 {
			vt.activeScreen.eraseRow(r, 0, column(vt.width()-1), vt.cursor.Style.Background)
		}

	// Erases from the beginning of the screen to the cursor, including the
	// cursor position. Line attribute becomes single-height, single-width
	// for all completely erased lines.
	case 1:
		vt.el(1)
		for r := row(0); r < vt.cursor.row; r += 1 {
			vt.activeScreen.eraseRow(r, 0, column(vt.width()-1), vt.cursor.Style.Background)
		}

	// Erases the complete display. All lines are erased and changed to
	// single-width. The cursor does not move.
	case 2:
		vt.resetPendingWrap()
		for r := row(0); r < row(vt.height()); r += 1 {
			for col := column(0); col < column(vt.width()); col += 1 {
				vt.activeScreen.eraseCell(r, col, vt.cursor.Style.Background)
			}
			vt.activeScreen.row(r).wrapped = false
			vt.activeScreen.row(r).wrapContinuation = false
		}

	// Erases saved lines in the scrollback buffer.
	case 3:
		vt.activeScreen.clearScrollback()
		vt.clampScrollOffset()
	}
}

// Erase in Line (EL) CSI Ps K
func (vt *Model) el(ps int) {
	r := vt.cursor.row
	switch ps {
	// Erases from the cursor to the end of the line, including the cursor
	// position. Line attribute is not affected.
	case 0:
		vt.resetWrap()
		for col := vt.cursor.col; col < column(vt.width()); col += 1 {
			vt.activeScreen.eraseCell(r, col, vt.cursor.Style.Background)
		}

	// Erases from the beginning of the line to the cursor, including the
	// cursor position. Line attribute is not affected.
	case 1:
		vt.resetPendingWrap()
		for col := column(0); col <= vt.cursor.col; col += 1 {
			vt.activeScreen.eraseCell(r, col, vt.cursor.Style.Background)
		}

	// Erases the complete line.
	case 2:
		vt.resetPendingWrap()
		for col := column(0); col < column(vt.width()); col += 1 {
			vt.activeScreen.eraseCell(r, col, vt.cursor.Style.Background)
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
	vt.resetWrap()
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

	if ps == 0 {
		ps = 1
	}

	if available := int(vt.margin.bottom-vt.cursor.row) + 1; ps > available {
		ps = available
	}

	// move the lines first
	for r := vt.margin.bottom; r >= (vt.cursor.row + row(ps)); r -= 1 {
		vt.activeScreen.copyRow(r, r-row(ps))
	}

	// insert the blank lines (we do this by erasing the cells)
	for r := row(0); r < row(ps); r += 1 {
		vt.activeScreen.eraseRow(vt.cursor.row+r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
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
	vt.resetWrap()
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

	if ps == 0 {
		ps = 1
	}

	if available := int(vt.margin.bottom-vt.cursor.row) + 1; ps > available {
		ps = available
	}

	for r := vt.cursor.row; r <= vt.margin.bottom; r += 1 {
		if r <= vt.margin.bottom-row(ps) {
			vt.activeScreen.copyRow(r, r+row(ps))
			continue
		}
		vt.activeScreen.eraseRow(r, vt.margin.left, vt.margin.right, vt.cursor.Style.Background)
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
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	row := vt.cursor.row
	for col := vt.cursor.col; col <= vt.margin.right; col += 1 {
		if col+column(ps) > vt.margin.right {
			vt.activeScreen.eraseCell(row, col, vt.cursor.Style.Background)
			continue
		}
		vt.activeScreen.setCell(row, col, *vt.activeScreen.cell(row, col+column(ps)))
	}
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

	for i := column(0); i < column(ps); i += 1 {
		if vt.cursor.col+i == column(vt.width()) {
			return
		}
		vt.activeScreen.eraseCell(vt.cursor.row, vt.cursor.col+i, vt.cursor.Style.Background)
	}
}

// Cursor Backward Tabulation (CBT) CSI Ps Z
//
// Move cursor backward Ps tabulations
func (vt *Model) cbt(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	// Note: Same comment as in "cht".
	newcol := vt.margin.left
	if i, _ := slices.BinarySearch(vt.tabStop, vt.cursor.col); i > 0 {
		i -= ps
		if i >= 0 {
			newcol = max(vt.tabStop[i], vt.margin.left)
		}
	}
	vt.cursor.col = newcol
}

// Tab Clear (TBC) CSI Ps g
func (vt *Model) tbc(ps int) {
	switch ps {
	case 0:
		if i, found := slices.BinarySearch(vt.tabStop, vt.cursor.col); found {
			slices.Delete(vt.tabStop, i, i+1)
		}
	case 3:
		vt.tabStop = []column{}
	}
}

// Line Position Absolute (VPA) CSI Ps d
//
// Move cursor to line Ps
func (vt *Model) vpa(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.row = row(ps - 1)
	if vt.cursor.row > row(vt.height()-1) {
		vt.cursor.row = row(vt.height() - 1)
	}
}

// Line Position Relative (VPR) CSI Ps e
//
// Move down Ps lines
func (vt *Model) vpr(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.row += row(ps)
	if vt.cursor.row > row(vt.height()-1) {
		vt.cursor.row = row(vt.height() - 1)
	}
}

// Character Position Absolute (HPA) CSI Ps `
//
// Move cursor to column Ps
func (vt *Model) hpa(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.col = column(ps - 1)
	if vt.cursor.col > column(vt.width()-1) {
		vt.cursor.col = column(vt.width() - 1)
	}
}

// Character Position Relative (HPR) CSI Ps a
//
// Move cursor to the right Ps times
func (vt *Model) hpr(ps int) {
	vt.resetWrap()
	if ps == 0 {
		ps = 1
	}
	vt.cursor.col += column(ps)
	if vt.cursor.col > column(vt.width()-1) {
		vt.cursor.col = column(vt.width() - 1)
	}
}

// Repeat (REP) CSI Ps b
//
// Repeat preceding graphic character Ps times
func (vt *Model) rep(ps int) {
	vt.resetWrap()
	col := vt.cursor.col
	if col == 0 {
		return
	}
	ch := *vt.activeScreen.cell(vt.cursor.row, col-1)
	for i := 0; i < ps; i += 1 {
		if col+column(i) == vt.margin.right {
			return
		}
		vt.activeScreen.cell(vt.cursor.row, vt.cursor.col+column(i)).Character = ch.Character
	}
}

// Set top and bottom margins CSI Ps ; Ps r
func (vt *Model) decstbm(pm ansi.CSI) {
	var (
		top row
		bot row
	)
	switch pm.NumParameters {
	case 0:
		top = 0
		bot = row(vt.height()) - 1
	case 1:
		top = row(pm.Param(0) - 1)
		bot = row(vt.height()) - 1
	default:
		top = row(pm.Param(0) - 1)
		bot = row(pm.Param(1) - 1)
	}
	if top >= bot {
		return
	}
	vt.resetWrap()
	vt.margin.top = top
	vt.margin.bottom = bot
	vt.cursor.row = 0
	vt.cursor.col = 0
}
