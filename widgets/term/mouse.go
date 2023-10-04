package term

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
)

func (vt *Model) handleMouse(msg vaxis.Mouse) string {
	if vt.mode&mouseButtons == 0 && vt.mode&mouseDrag == 0 && vt.mode&mouseMotion == 0 && vt.mode&mouseSGR == 0 {
		if vt.mode&altScroll != 0 && vt.mode&smcup != 0 {
			// Translate wheel motion into arrows up and down
			// 3x rows
			if msg.Button == vaxis.MouseWheelUp {
				vt.pty.WriteString("\x1bOA")
				vt.pty.WriteString("\x1bOA")
				vt.pty.WriteString("\x1bOA")
			}
			if msg.Button == vaxis.MouseWheelDown {
				vt.pty.WriteString("\x1bOB")
				vt.pty.WriteString("\x1bOB")
				vt.pty.WriteString("\x1bOB")
			}
		}
		return ""
	}
	// Return early if we aren't reporting motion
	if vt.mode&mouseMotion == 0 && msg.EventType == vaxis.EventMotion && msg.Button == vaxis.MouseNoButton {
		return ""
	}
	// Return early if we aren't reporting drags
	if vt.mode&mouseDrag == 0 && msg.EventType == vaxis.EventMotion {
		return ""
	}

	if vt.mode&mouseSGR != 0 {
		switch msg.EventType {
		case vaxis.EventMotion:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", msg.Button+32, msg.Col+1, msg.Row+1)
		case vaxis.EventPress:
			return fmt.Sprintf("\x1b[<%d;%d;%dM", msg.Button, msg.Col+1, msg.Row+1)
		case vaxis.EventRelease:
			return fmt.Sprintf("\x1b[<%d;%d;%dm", msg.Button, msg.Col+1, msg.Row+1)
		default:
			// unhandled
			return ""
		}
	}

	// legacy encoding
	encodedCol := 32 + msg.Col + 1
	encodedRow := 32 + msg.Row + 1

	return fmt.Sprintf("\x1b[M%c%c%c", msg.Button+32, encodedCol, encodedRow)
}
