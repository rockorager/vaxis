package vaxis

import (
	"fmt"
)

const (
	// Queries
	// Device Status Report - Cursor Position Report
	dsrcpr = "\x1b[6n"
	// Generic DSR
	dsr = "\x1b[?%dn"
	// Device primary attributes
	primaryAttributes  = "\x1b[c"
	tertiaryAttributes = "\x1b[=c"
	// Device Status Report - XTVERSION
	xtversion = "\x1b[>0q"
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
	clear        = "\x1b[H\x1b[2J"
	cup          = "\x1B[%d;%dH"
	osc8         = "\x1b]8;%s;%s\x1b\\"
	osc52put     = "\x1b]52;c;%s\x1b\\"
	osc52pop     = "\x1b]52;c;?\x1b\\"
	osc9notify   = "\x1b]9;%s\x1b\\"
	osc777notify = "\x1b]777;notify;%s;%s\x1b\\"
	setTitle     = "\x1b]2;%s\x1b\\"
	mouseShape   = "\x1b]22;%s\x1b\\"

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
	unicodeCore        = 2027
	colorThemeUpdates  = 2031
	sixelScrolling     = 8452

	// dsr requests/responses
	colorThemeReq  = 996
	colorThemeResp = 997

	// screen size, always requested pixels first and characters second
	textAreaSize = "\x1b[14t\x1b[18t"
)

func decset(mode int) string {
	return fmt.Sprintf("\x1B[?%dh", mode)
}

func decrst(mode int) string {
	return fmt.Sprintf("\x1B[?%dl", mode)
}

func decrqm(mode int) string {
	return fmt.Sprintf("\x1B[?%d$p", mode)
}

func tparm(s string, args ...any) string {
	return fmt.Sprintf(s, args...)
}

// xtgettcap prepares a query of a given terminfo capability
func xtgettcap(cap string) string {
	return "\x1bP+q" + hexEncode(cap) + "\x1b\\"
}

func hexEncode(cap string) string {
	return fmt.Sprintf("%X", cap)
}
