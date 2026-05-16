package term

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func (vt *Model) handleMouse(msg vaxis.Mouse) string {
	if !vt.mode.mouseX10 && !vt.mode.mouseButtons && !vt.mode.mouseDrag && !vt.mode.mouseMotion {
		if vt.mode.altScroll && vt.mode.smcup {
			// Translate wheel motion into arrows up and down
			// 3x rows
			if msg.Button == vaxis.MouseWheelUp {
				vt.writePtyString("\x1bOA")
				vt.writePtyString("\x1bOA")
				vt.writePtyString("\x1bOA")
			} else if msg.Button == vaxis.MouseWheelDown {
				vt.writePtyString("\x1bOB")
				vt.writePtyString("\x1bOB")
				vt.writePtyString("\x1bOB")
			}
		}
		return ""
	}
	if msg.Modifiers&vaxis.ModShift != 0 && !vt.mode.mouseShiftCapture {
		return ""
	}

	if vt.mode.mouseX10 {
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
	if msg.EventType == vaxis.EventMotion && !vt.mode.mouseMotion &&
		(msg.Button == vaxis.MouseNoButton || !vt.mode.mouseDrag) {
		return ""
	}

	if vt.mode.mouseSGR {
		button := vt.mouseButtonCode(msg, false)
		switch msg.EventType {
		case vaxis.EventMotion:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", button+32, msg.Col+1, msg.Row+1)
		case vaxis.EventPress:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", button, msg.Col+1, msg.Row+1)
		case vaxis.EventRelease:
			return fmt.Sprintf("\x1b[<%d;%d;%dm", button, msg.Col+1, msg.Row+1)
		default:
			// unhandled
			return ""
		}
	}

	// legacy encoding
	encodedCol := 32 + msg.Col + 1
	encodedRow := 32 + msg.Row + 1

	return fmt.Sprintf("\x1b[M%c%c%c", vt.mouseButtonCode(msg, true)+32, encodedCol, encodedRow)
}

func (vt *Model) mouseButtonCode(msg vaxis.Mouse, legacy bool) vaxis.MouseButton {
	if legacy && msg.EventType == vaxis.EventRelease {
		return vaxis.MouseNoButton
	}

	button := msg.Button
	if !vt.mode.mouseX10 {
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
	return button
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
