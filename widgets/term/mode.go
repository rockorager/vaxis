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

	// xterm
	//
	// Use alternate screen
	smcup bool
	// Bracketed paste
	paste bool
	// vt220 mouse
	mouseButtons bool
	// vt220 + drag
	mouseDrag bool
	// vt220 + all motion
	mouseMotion bool
	// Mouse SGR mode
	mouseSGR bool
	// Alternate scroll
	altScroll bool
	// Focus event tracking
	focusEvents bool
	// Unsolicited color scheme change notifications
	colorScheme bool
}

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
		vt.ed(2)
	}

	if enabled {
		vt.activeScreen = vt.altScreen
		vt.mode.smcup = true
		// Enable altScroll in the alt screen. This is only used if the
		// application doesn't enable mouse.
		vt.mode.altScroll = true
		vt.scrollOffset = 0
		if mode == 1049 {
			vt.ed(2)
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

func (vt *Model) decrqm(pd int) {
	ps := 0
	switch pd {
	case 1:
		switch vt.mode.decckm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 2:
		switch vt.mode.decanm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 3:
		switch vt.mode.deccolm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 4:
		switch vt.mode.decsclm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 5:
	case 6:
		switch vt.mode.decom {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 7:
		switch vt.mode.decawm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 8:
		switch vt.mode.decarm {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 25:
		switch vt.mode.dectcem {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1000:
		switch vt.mode.mouseButtons {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1002:
		switch vt.mode.mouseDrag {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1003:
		switch vt.mode.mouseMotion {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1006:
		switch vt.mode.mouseSGR {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1004:
		switch vt.mode.focusEvents {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 1007:
		switch vt.mode.altScroll {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 47, 1047, 1049:
		switch vt.mode.smcup {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 2004:
		switch vt.mode.paste {
		case true:
			ps = 1
		case false:
			ps = 2
		}
	case 2031:
		if vt.mode.colorScheme {
			ps = 1
		} else {
			ps = 2
		}
	}
	vt.enqueueReplyString(fmt.Sprintf("\x1B[?%d;%d$y", pd, ps))
}
