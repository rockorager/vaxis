package term

import (
	"fmt"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func (vt *Model) handleMouse(msg vaxis.Mouse) string {
	if vt.mode.mouseEvent == mouseEventNone {
		if vt.mode.altScroll && vt.mode.smcup {
			// Translate wheel motion into arrows up and down
			// 3x rows
			if msg.Button == vaxis.MouseWheelUp {
				return "\x1bOA\x1bOA\x1bOA"
			} else if msg.Button == vaxis.MouseWheelDown {
				return "\x1bOB\x1bOB\x1bOB"
			}
		}
		return ""
	}
	if msg.Modifiers&vaxis.ModShift != 0 && !vt.mode.mouseShiftCapture {
		return ""
	}

	if vt.mode.mouseEvent == mouseEventX10 {
		if msg.EventType != vaxis.EventPress {
			return ""
		}
		if msg.Button != vaxis.MouseLeftButton && msg.Button != vaxis.MouseMiddleButton && msg.Button != vaxis.MouseRightButton {
			return ""
		}
	}

	// Return early if event is (pure) motion but we aren't reporting
	// those (!mouseMotion) or event is drag (motion with pressed button)
	// but we aren't reporting those either (!mouseMotion && !mouseDrag).
	if msg.EventType == vaxis.EventMotion && vt.mode.mouseEvent != mouseEventAny &&
		(msg.Button == vaxis.MouseNoButton || vt.mode.mouseEvent != mouseEventButton) {
		return ""
	}
	if !supportedMouseButton(msg.Button) {
		return ""
	}

	legacyRelease := vt.mode.mouseFormat != mouseFormatSGR && vt.mode.mouseFormat != mouseFormatSGRPixels
	button := vt.mouseButtonCode(msg, legacyRelease)
	switch vt.mode.mouseFormat {
	case mouseFormatSGR:
		switch msg.EventType {
		case vaxis.EventMotion:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", button, msg.Col+1, msg.Row+1)
		case vaxis.EventPress:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", button, msg.Col+1, msg.Row+1)
		case vaxis.EventRelease:
			return fmt.Sprintf("\x1b[<%d;%d;%dm", button, msg.Col+1, msg.Row+1)
		default:
			// unhandled
			return ""
		}
	case mouseFormatURXVT:
		return fmt.Sprintf("\x1b[%d;%d;%dM", button+32, msg.Col+1, msg.Row+1)
	case mouseFormatSGRPixels:
		x, y := msg.XPixel, msg.YPixel
		return fmt.Sprintf("\x1b[<%d;%d;%d%c", button, x, y, mouseFinal(msg))
	}

	// legacy encoding
	if vt.mode.mouseFormat == mouseFormatX10 && (msg.Col > 222 || msg.Row > 222) {
		return ""
	}
	encodedCol := string(rune(32 + msg.Col + 1))
	encodedRow := string(rune(32 + msg.Row + 1))
	if vt.mode.mouseFormat == mouseFormatUTF8 {
		encodedCol = string(rune(msg.Col + 33))
		encodedRow = string(rune(msg.Row + 33))
	}

	return "\x1b[M" + string(rune(button+32)) + encodedCol + encodedRow
}

func supportedMouseButton(button vaxis.MouseButton) bool {
	switch button {
	case vaxis.MouseLeftButton,
		vaxis.MouseMiddleButton,
		vaxis.MouseRightButton,
		vaxis.MouseNoButton,
		vaxis.MouseWheelUp,
		vaxis.MouseWheelDown,
		vaxis.MouseWheelLeft,
		vaxis.MouseWheelRight,
		vaxis.MouseButton8,
		vaxis.MouseButton9:
		return true
	default:
		return false
	}
}

func mouseFinal(msg vaxis.Mouse) byte {
	if msg.EventType == vaxis.EventRelease {
		return 'm'
	}
	return 'M'
}

func (vt *Model) mouseButtonCode(msg vaxis.Mouse, legacy bool) vaxis.MouseButton {
	if legacy && msg.EventType == vaxis.EventRelease {
		return vaxis.MouseNoButton
	}

	button := msg.Button
	if vt.mode.mouseEvent != mouseEventX10 {
		if msg.Modifiers&vaxis.ModShift != 0 {
			button += 4
		}
		if msg.Modifiers&vaxis.ModAlt != 0 {
			button += 8
		}
		if msg.Modifiers&vaxis.ModCtrl != 0 {
			button += 16
		}
	}
	if msg.EventType == vaxis.EventMotion {
		button += 32
	}
	return button
}

type promptClickMove struct {
	left  int
	right int
}

func (vt *Model) handlePromptClick(msg vaxis.Mouse) string {
	if msg.Button != vaxis.MouseLeftButton || msg.EventType != vaxis.EventRelease {
		return ""
	}
	if vt.scrollOffset != 0 || vt.semanticPromptClick == semanticPromptClickNone || !vt.cursorIsAtPrompt() {
		return ""
	}
	if vt.mode.smcup || vt.activeScreen.state == nil || vt.height() == 0 || vt.width() == 0 {
		return ""
	}
	if msg.Row < 0 || msg.Row >= vt.height() || msg.Col < 0 || msg.Col >= vt.width() {
		return ""
	}
	promptStart, ok := vt.promptRedrawStartRow()
	if !ok || row(msg.Row) < promptStart {
		return ""
	}

	if vt.semanticPromptClick == semanticPromptClickEvents {
		return fmt.Sprintf("\x1b[<0;%d;%dM", msg.Col+1, msg.Row+1)
	}

	move := vt.promptClickMove(row(msg.Row), column(msg.Col))
	if move.left == 0 && move.right == 0 {
		return ""
	}
	leftArrow, rightArrow := "\x1b[D", "\x1b[C"
	if vt.mode.decckm {
		leftArrow, rightArrow = "\x1bOD", "\x1bOC"
	}
	return strings.Repeat(leftArrow, move.left) + strings.Repeat(rightArrow, move.right)
}

func (vt *Model) promptClickMove(clickRow row, clickCol column) promptClickMove {
	if vt.cursor.semanticContent != semanticInput &&
		vt.activeScreen.cell(vt.cursor.row, min(vt.cursor.col, column(vt.width()-1))).semanticContent != semanticInput {
		return promptClickMove{}
	}
	if clickRow == vt.cursor.row && clickCol == vt.cursor.col {
		return promptClickMove{}
	}
	if vt.cursor.row < clickRow || (vt.cursor.row == clickRow && vt.cursor.col < clickCol) {
		return vt.promptClickMoveRight(clickRow, clickCol)
	}
	return vt.promptClickMoveLeft(clickRow, clickCol)
}

func (vt *Model) promptClickMoveRight(clickRow row, clickCol column) promptClickMove {
	count := 0
	for r := vt.cursor.row; r < row(vt.height()); r += 1 {
		line := vt.activeScreen.line(r)
		isCursorRow := r == vt.cursor.row
		if !isCursorRow && vt.activeScreen.row(r).semanticPrompt != semanticPromptContinuation {
			break
		}
		start := column(0)
		if isCursorRow {
			start = vt.cursor.col + 1
		} else {
			start = firstSemanticInputCol(line)
		}
		for c := start; c < column(vt.width()); c += 1 {
			if line[c].semanticContent != semanticInput {
				continue
			}
			count += 1
			if r == clickRow && c == clickCol {
				return promptClickMove{right: count}
			}
		}
		if !vt.activeScreen.row(r).wrapped {
			if vt.activeScreen.cell(vt.cursor.row, min(vt.cursor.col, column(vt.width()-1))).semanticContent == semanticInput {
				count += 1
			}
			break
		}
	}
	return promptClickMove{right: count}
}

func (vt *Model) promptClickMoveLeft(clickRow row, clickCol column) promptClickMove {
	count := 0
	for r := vt.cursor.row; r >= 0; r -= 1 {
		line := vt.activeScreen.line(r)
		end := column(vt.width())
		if r == vt.cursor.row {
			end = vt.cursor.col
		}
		for c := end - 1; c >= 0; c -= 1 {
			if line[c].semanticContent != semanticInput {
				continue
			}
			count += 1
			if r == clickRow && c == clickCol {
				return promptClickMove{left: count}
			}
		}
		if !vt.activeScreen.row(r).wrapContinuation {
			break
		}
	}
	return promptClickMove{left: count}
}

func firstSemanticInputCol(line []cell) column {
	for i := range line {
		if line[i].semanticContent == semanticInput {
			return column(i)
		}
	}
	return column(len(line))
}

func (vt *Model) xtshiftescape(seq ansi.CSI) {
	switch seq.NumParameters {
	case 0:
		vt.mode.mouseShiftCapture = false
	case 1:
		switch seq.Parameters[0] {
		case 0:
			vt.mode.mouseShiftCapture = false
		case 1:
			vt.mode.mouseShiftCapture = true
		}
	}
}
