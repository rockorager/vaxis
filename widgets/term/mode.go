package term

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

type mode struct {
	// ANSI-Standardized modes
	//
	// Keyboard Action mode
	kam bool
	// Insert/Replace mode
	irm bool
	// Send/Receive mode
	srm bool
	// Line feed/new line mode
	lnm bool

	// ANSI-Compatible DEC Private Modes
	//
	// Cursor Key mode
	decckm bool
	// ANSI/VT52 mode
	decanm bool
	// Column mode
	deccolm bool
	// Scroll mode
	decsclm bool
	// Origin mode
	decom bool
	// Autowrap mode
	decawm bool
	// Reverse wraparound mode
	reverseWrap bool
	// Extended reverse wraparound mode
	reverseWrapExtended bool
	// Autorepeat mode
	decarm bool
	// Printer form feed mode
	decpff bool
	// Printer extent mode
	decpex bool
	// Text Cursor Enable mode
	dectcem bool
	// National replacement character sets
	decnrcm bool
	// Application keypad
	deckpam bool
	// Normal keypad
	deckpnm bool
	// Backarrow key mode
	decbkm bool
	// Enable left and right margins
	declrmm bool

	// xterm
	//
	// Use alternate screen
	smcup bool
	// Bracketed paste
	paste bool
	// vt220 mouse
	mouseX10 bool
	// vt220 mouse button events
	mouseButtons bool
	// vt220 + drag
	mouseDrag bool
	// vt220 + all motion
	mouseMotion bool
	// Active mouse tracking behavior. The individual bools above remain mode
	// report state; this mirrors Ghostty's separate active mouse_event flag.
	mouseEvent mouseEventMode
	// Mouse UTF-8 coordinate format
	mouseUTF8 bool
	// Mouse SGR mode
	mouseSGR bool
	// Mouse URXVT format
	mouseURXVT bool
	// Mouse SGR-pixels mode
	mouseSGRPixels bool
	// Active mouse reporting format.
	mouseFormat mouseFormatMode
	// Capture Shift+mouse instead of letting Shift escape mouse reporting.
	mouseShiftCapture bool
	// Alternate scroll
	altScroll bool
	// Ignore keypad application mode when Num Lock is active.
	ignoreKeypadWithNumLock bool
	// Prefix Alt-modified text keys with ESC.
	altEscPrefix bool
	// Alt sends escape mode.
	altSendsEscape bool
	// Save cursor mode
	saveCursor bool
	// Synchronized output mode.
	synchronizedOutput bool
	// Grapheme cluster mode.
	graphemeCluster bool
	// Focus event tracking
	focusEvents bool
	// Unsolicited color scheme change notifications
	colorScheme bool
	// In-band size reports.
	inBandSizeReports bool
	// xterm modifyOtherKeys state 2 numeric encoding.
	modifyOtherKeys2 bool

	// Most recent character protection mode. As in Ghostty, DECSCA off clears
	// the cursor pen but does not reset this because erase semantics depend on
	// whether ISO protection was most recently active.
	protected protectedMode
}

type protectedMode int

const (
	protectedModeOff protectedMode = iota
	protectedModeISO
	protectedModeDEC
)

type statusDisplay int

const (
	statusDisplayMain statusDisplay = iota
	statusDisplayLine
)

type mouseEventMode int

const (
	mouseEventNone mouseEventMode = iota
	mouseEventX10
	mouseEventNormal
	mouseEventButton
	mouseEventAny
)

type mouseFormatMode int

const (
	mouseFormatX10 mouseFormatMode = iota
	mouseFormatUTF8
	mouseFormatSGR
	mouseFormatURXVT
	mouseFormatSGRPixels
)

func (vt *Model) sm(params ansi.CSI) {
	for _, param := range params.Params() {
		switch param {
		case 2:
			vt.mode.kam = true
		case 4:
			vt.mode.irm = true
		case 12:
			vt.mode.srm = true
		case 20:
			vt.mode.lnm = true
		}
	}
}

func (vt *Model) rm(params ansi.CSI) {
	for _, param := range params.Params() {
		switch param {
		case 2:
			vt.mode.kam = false
		case 4:
			vt.mode.irm = false
		case 12:
			vt.mode.srm = false
		case 20:
			vt.mode.lnm = false
		}
	}
}

func (vt *Model) decset(params ansi.CSI) {
	for _, param := range params.Params() {
		vt.setDECMode(int(param), true)
	}
}

func (vt *Model) decrst(params ansi.CSI) {
	for _, param := range params.Params() {
		vt.setDECMode(int(param), false)
	}
}

func (vt *Model) setDECMode(param int, enabled bool) {
	switch param {
	case 1:
		vt.mode.decckm = enabled
	case 2:
		vt.mode.decanm = enabled
	case 3:
		vt.mode.deccolm = enabled
	case 4:
		vt.mode.decsclm = enabled
	case 5:
	case 6:
		vt.mode.decom = enabled
		vt.resetWrap()
		if enabled {
			vt.cursor.row = vt.margin.top
			vt.cursor.col = vt.margin.left
		} else {
			vt.cursor.row = 0
			vt.cursor.col = 0
		}
	case 7:
		vt.mode.decawm = enabled
		vt.resetWrap()
	case 8:
		vt.mode.decarm = enabled
	case 45:
		vt.mode.reverseWrap = enabled
	case 25:
		vt.mode.dectcem = enabled
	case 66:
		vt.mode.deckpam = enabled
		vt.mode.deckpnm = !enabled
	case 67:
		vt.mode.decbkm = enabled
	case 69:
		vt.mode.declrmm = enabled
		if !enabled {
			vt.margin.left = 0
			vt.margin.right = column(vt.width()) - 1
		}
	case 9:
		vt.mode.mouseX10 = enabled
		vt.setMouseEventMode(mouseEventX10, enabled)
	case 1000:
		vt.mode.mouseButtons = enabled
		vt.setMouseEventMode(mouseEventNormal, enabled)
	case 1002:
		vt.mode.mouseDrag = enabled
		vt.setMouseEventMode(mouseEventButton, enabled)
	case 1003:
		vt.mode.mouseMotion = enabled
		vt.setMouseEventMode(mouseEventAny, enabled)
	case 1005:
		vt.mode.mouseUTF8 = enabled
		vt.setMouseFormatMode(mouseFormatUTF8, enabled)
	case 1006:
		vt.mode.mouseSGR = enabled
		vt.setMouseFormatMode(mouseFormatSGR, enabled)
	case 1004:
		vt.mode.focusEvents = enabled
	case 1007:
		vt.mode.altScroll = enabled
	case 1015:
		vt.mode.mouseURXVT = enabled
		vt.setMouseFormatMode(mouseFormatURXVT, enabled)
	case 1016:
		vt.mode.mouseSGRPixels = enabled
		vt.setMouseFormatMode(mouseFormatSGRPixels, enabled)
	case 1035:
		vt.mode.ignoreKeypadWithNumLock = enabled
	case 1036:
		vt.mode.altEscPrefix = enabled
	case 1039:
		vt.mode.altSendsEscape = enabled
	case 47:
		vt.switchAltScreen(47, enabled)
	case 1048:
		vt.mode.saveCursor = enabled
		if enabled {
			vt.decsc()
		} else {
			vt.decrc()
		}
	case 1047:
		vt.switchAltScreen(1047, enabled)
	case 1049:
		vt.switchAltScreen(1049, enabled)
	case 1045:
		vt.mode.reverseWrapExtended = enabled
	case 2004:
		vt.mode.paste = enabled
	case 2026:
		vt.mode.synchronizedOutput = enabled
	case 2027:
		vt.mode.graphemeCluster = enabled
	case 2031:
		vt.mode.colorScheme = enabled
	case 2048:
		vt.mode.inBandSizeReports = enabled
	}
}

func (vt *Model) setMouseEventMode(mode mouseEventMode, enabled bool) {
	if enabled {
		vt.mode.mouseEvent = mode
		return
	}
	vt.mode.mouseEvent = mouseEventNone
}

func (vt *Model) setMouseFormatMode(mode mouseFormatMode, enabled bool) {
	if enabled {
		vt.mode.mouseFormat = mode
		return
	}
	vt.mode.mouseFormat = mouseFormatX10
}

func (vt *Model) saveMode(params ansi.CSI) {
	for _, param := range params.Params() {
		vt.setSavedDECMode(int(param), vt.decModeValue(int(param)))
	}
}

func (vt *Model) restoreMode(params ansi.CSI) {
	for _, param := range params.Params() {
		vt.setDECMode(int(param), vt.savedDECModeValue(int(param)))
	}
}

func (vt *Model) decModeValue(param int) bool {
	switch param {
	case 1:
		return vt.mode.decckm
	case 2:
		return vt.mode.decanm
	case 3:
		return vt.mode.deccolm
	case 4:
		return vt.mode.decsclm
	case 6:
		return vt.mode.decom
	case 7:
		return vt.mode.decawm
	case 8:
		return vt.mode.decarm
	case 45:
		return vt.mode.reverseWrap
	case 25:
		return vt.mode.dectcem
	case 66:
		return vt.mode.deckpam
	case 67:
		return vt.mode.decbkm
	case 69:
		return vt.mode.declrmm
	case 9:
		return vt.mode.mouseX10
	case 1000:
		return vt.mode.mouseButtons
	case 1002:
		return vt.mode.mouseDrag
	case 1003:
		return vt.mode.mouseMotion
	case 1005:
		return vt.mode.mouseUTF8
	case 1006:
		return vt.mode.mouseSGR
	case 1004:
		return vt.mode.focusEvents
	case 1007:
		return vt.mode.altScroll
	case 1015:
		return vt.mode.mouseURXVT
	case 1016:
		return vt.mode.mouseSGRPixels
	case 1035:
		return vt.mode.ignoreKeypadWithNumLock
	case 1036:
		return vt.mode.altEscPrefix
	case 1039:
		return vt.mode.altSendsEscape
	case 47, 1047, 1049:
		return vt.mode.smcup
	case 1048:
		return vt.mode.saveCursor
	case 1045:
		return vt.mode.reverseWrapExtended
	case 2004:
		return vt.mode.paste
	case 2026:
		return vt.mode.synchronizedOutput
	case 2027:
		return vt.mode.graphemeCluster
	case 2031:
		return vt.mode.colorScheme
	case 2048:
		return vt.mode.inBandSizeReports
	default:
		return false
	}
}

func (vt *Model) savedDECModeValue(param int) bool {
	return savedModeValue(vt.savedMode, param)
}

func (vt *Model) setSavedDECMode(param int, enabled bool) {
	setModeValue(&vt.savedMode, param, enabled)
}

func savedModeValue(m mode, param int) bool {
	switch param {
	case 1:
		return m.decckm
	case 2:
		return m.decanm
	case 3:
		return m.deccolm
	case 4:
		return m.decsclm
	case 6:
		return m.decom
	case 7:
		return m.decawm
	case 8:
		return m.decarm
	case 45:
		return m.reverseWrap
	case 25:
		return m.dectcem
	case 66:
		return m.deckpam
	case 67:
		return m.decbkm
	case 69:
		return m.declrmm
	case 9:
		return m.mouseX10
	case 1000:
		return m.mouseButtons
	case 1002:
		return m.mouseDrag
	case 1003:
		return m.mouseMotion
	case 1005:
		return m.mouseUTF8
	case 1006:
		return m.mouseSGR
	case 1004:
		return m.focusEvents
	case 1007:
		return m.altScroll
	case 1015:
		return m.mouseURXVT
	case 1016:
		return m.mouseSGRPixels
	case 1035:
		return m.ignoreKeypadWithNumLock
	case 1036:
		return m.altEscPrefix
	case 1039:
		return m.altSendsEscape
	case 47, 1047, 1049:
		return m.smcup
	case 1048:
		return m.saveCursor
	case 1045:
		return m.reverseWrapExtended
	case 2004:
		return m.paste
	case 2026:
		return m.synchronizedOutput
	case 2027:
		return m.graphemeCluster
	case 2031:
		return m.colorScheme
	case 2048:
		return m.inBandSizeReports
	default:
		return false
	}
}

func setModeValue(m *mode, param int, enabled bool) {
	switch param {
	case 1:
		m.decckm = enabled
	case 2:
		m.decanm = enabled
	case 3:
		m.deccolm = enabled
	case 4:
		m.decsclm = enabled
	case 6:
		m.decom = enabled
	case 7:
		m.decawm = enabled
	case 8:
		m.decarm = enabled
	case 45:
		m.reverseWrap = enabled
	case 25:
		m.dectcem = enabled
	case 66:
		m.deckpam = enabled
		m.deckpnm = !enabled
	case 67:
		m.decbkm = enabled
	case 69:
		m.declrmm = enabled
	case 9:
		m.mouseX10 = enabled
	case 1000:
		m.mouseButtons = enabled
	case 1002:
		m.mouseDrag = enabled
	case 1003:
		m.mouseMotion = enabled
	case 1005:
		m.mouseUTF8 = enabled
	case 1006:
		m.mouseSGR = enabled
	case 1004:
		m.focusEvents = enabled
	case 1007:
		m.altScroll = enabled
	case 1015:
		m.mouseURXVT = enabled
	case 1016:
		m.mouseSGRPixels = enabled
	case 1035:
		m.ignoreKeypadWithNumLock = enabled
	case 1036:
		m.altEscPrefix = enabled
	case 1039:
		m.altSendsEscape = enabled
	case 47, 1047, 1049:
		m.smcup = enabled
	case 1048:
		m.saveCursor = enabled
	case 1045:
		m.reverseWrapExtended = enabled
	case 2004:
		m.paste = enabled
	case 2026:
		m.synchronizedOutput = enabled
	case 2027:
		m.graphemeCluster = enabled
	case 2031:
		m.colorScheme = enabled
	case 2048:
		m.inBandSizeReports = enabled
	}
}

func (vt *Model) switchAltScreen(mode int, enabled bool) {
	if mode == 1049 && enabled {
		vt.decsc()
	}

	wasAlt := vt.mode.smcup
	switched := wasAlt != enabled
	if mode == 1047 && !enabled && wasAlt {
		vt.ed(2, false)
	}

	if enabled {
		vt.activeScreen = vt.altScreen
		vt.mode.smcup = true
		vt.scrollOffset = 0
		if mode == 1049 {
			vt.ed(2, false)
		}
		if switched {
			vt.clearCursorHyperlink()
		}
		return
	}

	vt.activeScreen = vt.primaryScreen
	vt.mode.smcup = false
	vt.scrollOffset = 0
	if mode == 1049 && wasAlt {
		vt.decrc()
	}
	if switched {
		vt.clearCursorHyperlink()
	}
}

func (vt *Model) clearCursorHyperlink() {
	vt.cursor.Hyperlink = ""
	vt.cursor.HyperlinkParams = ""
}

func (vt *Model) decrqm(pd int, ansiMode bool) {
	ps := 0
	if ansiMode {
		switch pd {
		case 2:
			ps = modeReportState(vt.mode.kam)
		case 4:
			ps = modeReportState(vt.mode.irm)
		case 12:
			ps = modeReportState(vt.mode.srm)
		case 20:
			ps = modeReportState(vt.mode.lnm)
		}
		vt.enqueueReplyString(fmt.Sprintf("\x1B[%d;%d$y", pd, ps))
		return
	}

	switch pd {
	case 1:
		ps = modeReportState(vt.mode.decckm)
	case 2:
		ps = modeReportState(vt.mode.decanm)
	case 3:
		ps = modeReportState(vt.mode.deccolm)
	case 4:
		ps = modeReportState(vt.mode.decsclm)
	case 5:
	case 6:
		ps = modeReportState(vt.mode.decom)
	case 7:
		ps = modeReportState(vt.mode.decawm)
	case 8:
		ps = modeReportState(vt.mode.decarm)
	case 45:
		ps = modeReportState(vt.mode.reverseWrap)
	case 25:
		ps = modeReportState(vt.mode.dectcem)
	case 66:
		ps = modeReportState(vt.mode.deckpam)
	case 67:
		ps = modeReportState(vt.mode.decbkm)
	case 69:
		ps = modeReportState(vt.mode.declrmm)
	case 9:
		ps = modeReportState(vt.mode.mouseX10)
	case 1000:
		ps = modeReportState(vt.mode.mouseButtons)
	case 1002:
		ps = modeReportState(vt.mode.mouseDrag)
	case 1003:
		ps = modeReportState(vt.mode.mouseMotion)
	case 1005:
		ps = modeReportState(vt.mode.mouseUTF8)
	case 1006:
		ps = modeReportState(vt.mode.mouseSGR)
	case 1004:
		ps = modeReportState(vt.mode.focusEvents)
	case 1007:
		ps = modeReportState(vt.mode.altScroll)
	case 1015:
		ps = modeReportState(vt.mode.mouseURXVT)
	case 1016:
		ps = modeReportState(vt.mode.mouseSGRPixels)
	case 1035:
		ps = modeReportState(vt.mode.ignoreKeypadWithNumLock)
	case 1036:
		ps = modeReportState(vt.mode.altEscPrefix)
	case 1039:
		ps = modeReportState(vt.mode.altSendsEscape)
	case 47, 1047, 1049:
		ps = modeReportState(vt.mode.smcup)
	case 1048:
		ps = modeReportState(vt.mode.saveCursor)
	case 1045:
		ps = modeReportState(vt.mode.reverseWrapExtended)
	case 2004:
		ps = modeReportState(vt.mode.paste)
	case 2026:
		ps = modeReportState(vt.mode.synchronizedOutput)
	case 2027:
		ps = modeReportState(vt.mode.graphemeCluster)
	case 2031:
		ps = modeReportState(vt.mode.colorScheme)
	case 2048:
		ps = modeReportState(vt.mode.inBandSizeReports)
	}
	vt.enqueueReplyString(fmt.Sprintf("\x1B[?%d;%d$y", pd, ps))
}

func modeReportState(enabled bool) int {
	if enabled {
		return 1
	}
	return 2
}
