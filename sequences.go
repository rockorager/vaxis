package vaxis

import (
	"bytes"
	"fmt"
)

const (
	// Queries
	// Device Status Report - Cursor Position Report
	dsrcpr = "\x1b[6n"
	// Device primary attributes
	primaryAttributes  = "\x1b[c"
	tertiaryAttributes = "\x1b[=c"
	// Device Status Report - XTVERSION
	xtversion = "\x1b[>0q"
	// Synchronized Update Mode
	sumQuery = "\x1b[?2026$p"
	// kitty keyboard protocol
	kittyKBQuery  = "\x1b[?u"
	kittyKBEnable = "\x1b[=%du"
	kittyKBPush   = "\x1b[>1u"
	kittyKBPop    = "\x1b[<u"
	// kitty graphics protocol
	kittyGquery = "\x1b_Gi=1,a=q\x1b\\"
	// sixel query XTSMGRAPHICS
	xtsmSixelGeom = "\x1b[?2;1;0S"

	// Misc
	clear      = "\x1b[H\x1b[2J"
	cup        = "\x1B[%d;%dH"
	osc8WithID = "\x1b]8;id=%s;%s\x1b\\"
	osc8       = "\x1b]8;;%s\x1b\\"
	osc8End    = "\x1b]8;;\x1b\\"
	osc52put   = "\x1b]52;c;%s\x1b\\"
	osc52pop   = "\x1b]52;c;?\x1b\\"
	notify     = "\x1b]9;%s\x1b\\"
	setTitle   = "\x1b]2;%s\x1b\\"

	// SGR
	sgrReset           = "\x1b[m"
	boldSet            = "\x1b[1m"
	dimSet             = "\x1b[2m"
	italicSet          = "\x1b[3m"
	underlineSet       = "\x1b[4m"
	blinkSet           = "\x1b[5m"
	reverseSet         = "\x1b[7m"
	hiddenSet          = "\x1b[8m"
	strikethroughSet   = "\x1b[9m"
	boldDimReset       = "\x1b[22m"
	italicReset        = "\x1b[23m"
	underlineReset     = "\x1b[24m"
	blinkReset         = "\x1b[25m"
	reverseReset       = "\x1b[27m"
	hiddenReset        = "\x1b[28m"
	strikethroughReset = "\x1b[29m"
	fgReset            = "\x1b[39m"
	bgReset            = "\x1b[49m"
	ulColorReset       = "\x1b[59m"

	// SGR Parameterized
	fgSet       = "\x1b[3%dm"
	fgBrightSet = "\x1b[9%dm"
	fgIndexSet  = "\x1b[38:5:%dm"
	fgRGBSet    = "\x1b[38:2:%d:%d:%dm"
	bgSet       = "\x1b[4%dm"
	bgBrightSet = "\x1b[10%dm"
	bgIndexSet  = "\x1b[48:5:%dm"
	bgRGBSet    = "\x1b[48:2:%d:%d:%dm"
	ulIndexSet  = "\x1b[58:5:%dm"
	ulRGBSet    = "\x1b[58:2:%d:%d:%dm"
	ulStyleSet  = "\x1b[4:%dm"

	// bracketed paste signals. All terminals are using the same sequences.
	// We only check terminfo for support. If supported, we turn it on and
	// we'll see these on pastes
	ps = "\x1b[200~" // paste started
	pe = "\x1b[201~" // paste ended

	// cursor styles
	cursorStyleSet   = "\x1b[%d q"
	cursorStyleReset = "\x1b[ q"

	// keypad
	applicationMode = "\x1b="
	numericMode     = "\x1b>"

	// Private Modes
	cursorKeys         = 1
	cursorVisibility   = 25
	mouseAllEvents     = 1003
	mouseFocusEvents   = 1004
	mouseSGR           = 1006
	alternateScreen    = 1049
	bracketedPaste     = 2004
	synchronizedUpdate = 2026
)

func decset(mode int) string {
	return fmt.Sprintf("\x1B[?%dh", mode)
}

func decrst(mode int) string {
	return fmt.Sprintf("\x1B[?%dl", mode)
}

func tparm(s string, args ...any) string {
	return fmt.Sprintf(s, args...)
}

// xtgettcap prepares a query of a given terminfo capability
func xtgettcap(cap string) string {
	out := bytes.NewBuffer(nil)
	out.WriteString("\x1bP+q")
	out.WriteString(hexEncode(cap))
	out.WriteString("\x1b\\")
	return out.String()
}

func hexEncode(cap string) string {
	return fmt.Sprintf("%X", cap)
}
