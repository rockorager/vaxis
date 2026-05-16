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
	// Mouse SGR mode
	mouseSGR bool
	// Alternate scroll
	altScroll bool
	// Save cursor mode
	saveCursor bool
	// Focus event tracking
	focusEvents bool
	// Unsolicited color scheme change notifications
	colorScheme bool

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
		switch param {
		case 1:
			vt.mode.decckm = true
		case 2:
			vt.mode.decanm = true
		case 3:
			vt.mode.deccolm = true
		case 4:
			vt.mode.decsclm = true
		case 5:
		case 6:
			vt.mode.decom = true
			vt.resetWrap()
			vt.cursor.row = vt.margin.top
			vt.cursor.col = vt.margin.left
		case 7:
			vt.mode.decawm = true
			vt.resetWrap()
		case 8:
			vt.mode.decarm = true
		case 25:
			vt.mode.dectcem = true
		case 69:
			vt.mode.declrmm = true
		case 9:
			vt.mode.mouseX10 = true
		case 1000:
			vt.mode.mouseButtons = true
		case 1002:
			vt.mode.mouseDrag = true
		case 1003:
			vt.mode.mouseMotion = true
		case 1006:
			vt.mode.mouseSGR = true
		case 1004:
			vt.mode.focusEvents = true
		case 1007:
			vt.mode.altScroll = true
		case 47:
			vt.switchAltScreen(47, true)
		case 1048:
			vt.mode.saveCursor = true
			vt.decsc()
		case 1047:
			vt.switchAltScreen(1047, true)
		case 1049:
			vt.switchAltScreen(1049, true)
		case 2004:
			vt.mode.paste = true
		case 2031:
			vt.mode.colorScheme = true
		}
	}
}

func (vt *Model) decrst(params ansi.CSI) {
	for _, param := range params.Params() {
		switch param {
		case 1:
			vt.mode.decckm = false
		case 2:
			vt.mode.decanm = false
		case 3:
			vt.mode.deccolm = false
		case 4:
			vt.mode.decsclm = false
		case 5:
		case 6:
			vt.mode.decom = false
			vt.resetWrap()
			vt.cursor.row = 0
			vt.cursor.col = 0
		case 7:
			vt.mode.decawm = false
			vt.resetWrap()
		case 8:
			vt.mode.decarm = false
		case 25:
			vt.mode.dectcem = false
		case 69:
			vt.mode.declrmm = false
			vt.margin.left = 0
			vt.margin.right = column(vt.width()) - 1
		case 9:
			vt.mode.mouseX10 = false
		case 1000:
			vt.mode.mouseButtons = false
		case 1002:
			vt.mode.mouseDrag = false
		case 1003:
			vt.mode.mouseMotion = false
		case 1006:
			vt.mode.mouseSGR = false
		case 1004:
			vt.mode.focusEvents = false
		case 1007:
			vt.mode.altScroll = false
		case 47:
			vt.switchAltScreen(47, false)
		case 1048:
			vt.mode.saveCursor = false
			vt.decrc()
		case 1047:
			vt.switchAltScreen(1047, false)
		case 1049:
			vt.switchAltScreen(1049, false)
		case 2004:
			vt.mode.paste = false
		case 2031:
			vt.mode.colorScheme = false
		}
	}
}

func (vt *Model) switchAltScreen(mode int, enabled bool) {
	if mode == 1049 && enabled {
		vt.decsc()
	}

	wasAlt := vt.mode.smcup
	if mode == 1047 && !enabled && wasAlt {
		vt.ed(2, false)
	}

	if enabled {
		vt.activeScreen = vt.altScreen
		vt.mode.smcup = true
		// Enable altScroll in the alt screen. This is only used if the
		// application doesn't enable mouse.
		vt.mode.altScroll = true
		vt.scrollOffset = 0
		if mode == 1049 {
			vt.ed(2, false)
		}
		return
	}

	vt.activeScreen = vt.primaryScreen
	vt.mode.smcup = false
	vt.mode.altScroll = false
	vt.scrollOffset = 0
	if mode == 1049 && wasAlt {
		vt.decrc()
	}
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
	case 25:
		ps = modeReportState(vt.mode.dectcem)
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
	case 1006:
		ps = modeReportState(vt.mode.mouseSGR)
	case 1004:
		ps = modeReportState(vt.mode.focusEvents)
	case 1007:
		ps = modeReportState(vt.mode.altScroll)
	case 47, 1047, 1049:
		ps = modeReportState(vt.mode.smcup)
	case 1048:
		ps = modeReportState(vt.mode.saveCursor)
	case 2004:
		ps = modeReportState(vt.mode.paste)
	case 2031:
		ps = modeReportState(vt.mode.colorScheme)
	}
	vt.enqueueReplyString(fmt.Sprintf("\x1B[?%d;%d$y", pd, ps))
}

func modeReportState(enabled bool) int {
	if enabled {
		return 1
	}
	return 2
}
