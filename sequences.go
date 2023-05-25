package rtk

import "os"

const (
	// Device Status Report - Cursor Position Report
	// dsrcpr = "\x1b[6n"
	// Device Status Report - XTVERSION
	xtversion = "\x1b[>0q"

	// Synchronized Update Mode
	sumSet   = "\x1b[?2026h"
	sumReset = "\x1b[?2026l"
	sumQuery = "\x1b[?2026$p"

	// kitty keyboard protocol
	kkbpQuery = "\x1b[?u"

	// rgb. These usually aren't in terminfo in any way
	setrgbf    = "\x1b[38;2;%p1%d;%p2%d;%p3%dm"
	setrgbb    = "\x1b[48;2;%p1%d;%p2%d;%p3%dm"
	setrgbfgbg = "\x1b[38;2;%p1%d;%p2%d;%p3%d;48;2;%p4%d;%p5%d;%p6%dm"

	// These have no terminfo entry but they work everywhere so we hardcode
	// them
	setfDefault    = "\x1b[39m"
	setbDefault    = "\x1b[49m"
	boldDimReset   = "\x1b[22m"
	blinkReset     = "\x1b[25m"
	invisibleReset = "\x1b[28m"
)

// Below we pull from terminfo
var (
	clear  string
	civis  string
	cvvis  string
	cup    string
	dsrcpr string
	setaf  string
	setab  string
	smcup  string
	rmcup  string

	boldSet            string
	dimSet             string
	italicSet          string
	underlineSet       string
	blinkSet           string
	reverseSet         string
	invisibleSet       string
	strikethroughSet   string
	italicReset        string
	underlineReset     string
	reverseReset       string
	strikethroughReset string
	resetAll           string
)

func setupTermInfo() error {
	info, err := infocmp(os.Getenv("TERM"))
	if err != nil {
		return err
	}

	// Set our terminfo strings
	cup = info.Strings["cup"]
	setaf = info.Strings["setaf"]
	setab = info.Strings["setab"]
	dsrcpr = info.Strings["u7"]
	boldSet = info.Strings["bold"]
	dimSet = info.Strings["dim"]
	italicSet = info.Strings["sitm"]
	underlineSet = info.Strings["smul"]
	blinkSet = info.Strings["blink"]
	reverseSet = info.Strings["rev"]
	invisibleSet = info.Strings["invis"]
	strikethroughSet = info.Strings["smxx"]
	italicReset = info.Strings["ritm"]
	underlineReset = info.Strings["rmul"]
	reverseReset = info.Strings["rmso"]
	strikethroughReset = info.Strings["rmxx"]
	resetAll = info.Strings["sgr0"]
	smcup = info.Strings["smcup"]
	rmcup = info.Strings["rmcup"]
	clear = info.Strings["clear"]
	civis = info.Strings["civis"]
	cvvis = info.Strings["cvvis"]
	return nil
}
