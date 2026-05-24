package term

import (
	"slices"

	"go.rockorager.dev/vaxis"
)

func (vt *Model) setProtectedMode(mode protectedMode) {
	switch mode {
	case protectedModeOff:
		vt.cursor.protected = false
	case protectedModeISO:
		vt.cursor.protected = true
		vt.mode.protected = protectedModeISO
	case protectedModeDEC:
		vt.cursor.protected = true
		vt.mode.protected = protectedModeDEC
	}
}

// Index ESC-D
func (vt *Model) ind() {
	vt.resetPendingWrap()
	if vt.cursor.row == vt.margin.bottom {
		if !vt.cursorInHorizontalMargins() {
			return
		}
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
	vt.cr()
}

// Horizontal tab set ESC-H
func (vt *Model) hts() {
	if i, found := slices.BinarySearch(vt.tabStop, vt.cursor.col); !found {
		vt.tabStop = slices.Insert(vt.tabStop, i, vt.cursor.col)
	}
}

// Reverse Index ESC-M
func (vt *Model) ri() {
	if vt.cursor.row == vt.margin.top && vt.cursorInHorizontalMargins() {
		vt.scrollDown(1)
		return
	}
	vt.cuu(1)
}

func (vt *Model) cursorInHorizontalMargins() bool {
	if vt.lastCol && vt.cursor.col == vt.margin.right+1 {
		return true
	}
	return vt.cursor.col >= vt.margin.left && vt.cursor.col <= vt.margin.right
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
	vt.cursor.Style = vaxis.Style{
		Foreground: vt.cursor.Foreground,
		Background: vt.cursor.Background,
	}

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
		cursor:   vt.cursor,
		decom:    vt.mode.decom,
		lastCol:  vt.lastCol,
		saved:    true,
		charsets: vt.charsets,
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

	style := vt.cursor.style
	hyperlink := vt.cursor.Hyperlink
	hyperlinkParams := vt.cursor.HyperlinkParams
	semanticContent := vt.cursor.semanticContent
	semanticClearEOL := vt.cursor.semanticClearEOL
	vt.cursor = state.cursor
	vt.cursor.style = style
	vt.cursor.Hyperlink = hyperlink
	vt.cursor.HyperlinkParams = hyperlinkParams
	vt.cursor.semanticContent = semanticContent
	vt.cursor.semanticClearEOL = semanticClearEOL
	vt.charsets = state.charsets
	vt.mode.decom = state.decom
	vt.clampCursor()
	vt.lastCol = state.lastCol && vt.cursor.col >= vt.margin.right
}

// Reset Initial State (RIS) ESC-c
func (vt *Model) ris() {
	w := vt.width()
	h := vt.height()
	vt.clearSelectionLocked()
	vt.altScreen = newScreenBuffer(w, h, 0)
	vt.primaryScreen = newScreenBuffer(w, h, defaultScrollbackLines)
	vt.margin.top = 0
	vt.margin.bottom = row(h) - 1
	vt.margin.left = 0
	vt.margin.right = column(w) - 1
	vt.cursor = cursor{}
	vt.lastCol = false
	vt.scrollOffset = 0
	vt.activeScreen = vt.primaryScreen
	vt.charsets = defaultCharsets()
	vt.title = ""
	vt.workingDirectoryURL = ""
	vt.colors = terminalColors{}
	vt.setMouseShape(vaxis.MouseShapeTextInput)
	vt.shellRedrawsPrompt = semanticPromptRedrawTrue
	vt.semanticPromptClick = semanticPromptClickNone
	vt.status = statusDisplayMain
	vt.previousChar = vaxis.Character{}
	vt.hasPreviousChar = false
	vt.clearGraphicsLocked()
	vt.setSynchronizedOutput(false)
	vt.mode = defaultMode()
	vt.savedMode = mode{}
	vt.primaryState = defaultCursorState()
	vt.altState = defaultCursorState()
	vt.primaryKittyKeyboard = kittyKeyboardStack{}
	vt.altKittyKeyboard = kittyKeyboardStack{}
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
