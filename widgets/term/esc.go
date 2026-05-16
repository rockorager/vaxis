package term

import (
	"slices"

	"git.sr.ht/~rockorager/vaxis"
)

func (vt *Model) esc(esc string) {
	switch esc {
	case "7":
		vt.decsc()
	case "8":
		vt.decrc()
	case "D":
		vt.ind()
	case "E":
		vt.nel()
	case "H":
		vt.hts()
	case "M":
		vt.ri()
	case "N":
		vt.charsets.saved = vt.charsets.selected
		vt.charsets.singleShift = true
		vt.charsets.selected = g2
	case "O":
		vt.charsets.saved = vt.charsets.selected
		vt.charsets.singleShift = true
		vt.charsets.selected = g3
	case "=":
		vt.mode.deckpam = true
		vt.mode.deckpnm = false
	case ">":
		vt.mode.deckpnm = true
		vt.mode.deckpam = false
	case "c":
		vt.ris()
	case "(0":
		vt.charsets.designations[g0] = decSpecialAndLineDrawing
	case ")0":
		vt.charsets.designations[g1] = decSpecialAndLineDrawing
	case "*0":
		vt.charsets.designations[g2] = decSpecialAndLineDrawing
	case "+0":
		vt.charsets.designations[g3] = decSpecialAndLineDrawing
	case "(B":
		vt.charsets.designations[g0] = ascii
	case ")B":
		vt.charsets.designations[g1] = ascii
	case "*B":
		vt.charsets.designations[g2] = ascii
	case "+B":
		vt.charsets.designations[g3] = ascii
	case "#8":
		vt.decaln()
	}
}

// Index ESC-D
func (vt *Model) ind() {
	vt.resetWrap()
	if vt.cursor.row == vt.margin.bottom {
		vt.scrollUp(1)
		return
	}
	if vt.cursor.row >= row(vt.height()-1) {
		// don't let row go beyond the height

		return
	}
	vt.cursor.row += 1
}

// Next line ESC-E
// Moves cursor to the left margin of the next line, scrolling if necessary
func (vt *Model) nel() {
	vt.ind()
	vt.cursor.col = vt.margin.left
}

// Horizontal tab set ESC-H
func (vt *Model) hts() {
	if i, found := slices.BinarySearch(vt.tabStop, vt.cursor.col); !found {
		vt.tabStop = slices.Insert(vt.tabStop, i, vt.cursor.col)
	}
}

// Reverse Index ESC-M
func (vt *Model) ri() {
	vt.resetWrap()
	if vt.cursor.row < 0 {
		return
	}
	if vt.cursor.row == vt.margin.top {
		vt.scrollDown(1)
		return
	}
	vt.cursor.row -= 1
}

func (vt *Model) decaln() {
	w := vt.width()
	h := vt.height()
	if w <= 0 || h <= 0 {
		return
	}
	vt.resetMargins(w, h)
	vt.mode.decom = false
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false

	fill := cell{
		Cell: vaxis.Cell{
			Character: vaxis.Character{
				Grapheme: "E",
				Width:    1,
			},
			Style: vt.cursor.Style,
		},
	}
	for r := row(0); r < row(h); r += 1 {
		*vt.activeScreen.row(r) = screenRow{}
		for col := column(0); col < column(w); col += 1 {
			vt.activeScreen.setCell(r, col, fill)
		}
	}
}

// Save Cursor DECSC ESC-7
func (vt *Model) decsc() {
	state := cursorState{
		cursor:  vt.cursor,
		decawm:  vt.mode.decawm,
		decom:   vt.mode.decom,
		lastCol: vt.lastCol,
		saved:   true,
		charsets: charsets{
			selected: vt.charsets.selected,
			saved:    vt.charsets.saved,
			designations: map[charsetDesignator]charset{
				g0: vt.charsets.designations[g0],
				g1: vt.charsets.designations[g1],
				g2: vt.charsets.designations[g2],
				g3: vt.charsets.designations[g3],
			},
		},
	}
	switch {
	case vt.mode.smcup:
		// We are in alt screen
		vt.altState = state
	default:
		vt.primaryState = state
	}
}

// Restore Cursor DECRC ESC-8
func (vt *Model) decrc() {
	var state cursorState
	switch {
	case vt.mode.smcup:
		// In the alt screen
		state = vt.altState
	default:
		state = vt.primaryState
	}
	if !state.saved {
		state = defaultCursorState()
	}

	vt.cursor = state.cursor
	vt.charsets = charsets{
		selected: state.charsets.selected,
		saved:    state.charsets.saved,
		designations: map[charsetDesignator]charset{
			g0: state.charsets.designations[g0],
			g1: state.charsets.designations[g1],
			g2: state.charsets.designations[g2],
			g3: state.charsets.designations[g3],
		},
	}
	vt.mode.decawm = state.decawm
	vt.mode.decom = state.decom
	vt.clampCursor()
	vt.lastCol = state.lastCol && vt.cursor.col >= vt.margin.right
}

// Reset Initial State (RIS) ESC-c
func (vt *Model) ris() {
	w := vt.width()
	h := vt.height()
	vt.altScreen = newScreenBuffer(w, h, 0)
	vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
	vt.margin.bottom = row(h) - 1
	vt.margin.right = column(w) - 1
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false
	vt.activeScreen = vt.primaryScreen
	vt.charsets = charsets{
		selected: 0,
		saved:    0,
		designations: map[charsetDesignator]charset{
			g0: ascii,
			g1: ascii,
			g2: ascii,
			g3: ascii,
		},
	}
	vt.mode = mode{
		decawm:  true,
		dectcem: true,
	}
	vt.primaryState = defaultCursorState()
	vt.altState = defaultCursorState()
	vt.setDefaultTabStops()
}

func (vt *Model) setDefaultTabStops() {
	vt.tabStop = []column{}
	width := vt.width()
	if width == 0 {
		width = 50 * 7
	}
	for i := 8; i < width; i += 8 {
		vt.tabStop = append(vt.tabStop, column(i))
	}
}
