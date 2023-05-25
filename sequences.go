package rtk

import (
	"os"
)

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
	bold = info.Strings["bold"]
	dim = info.Strings["dim"]
	sitm = info.Strings["sitm"]
	smul = info.Strings["smul"]
	blink = info.Strings["blink"]
	rev = info.Strings["rev"]
	invis = info.Strings["invis"]
	smxx = info.Strings["smxx"]
	ritm = info.Strings["ritm"]
	rmul = info.Strings["rmul"]
	rmso = info.Strings["rmso"]
	rmxx = info.Strings["rmxx"]
	sgr0 = info.Strings["sgr0"]
	smcup = info.Strings["smcup"]
	rmcup = info.Strings["rmcup"]
	clear = info.Strings["clear"]
	civis = info.Strings["civis"]
	cvvis = info.Strings["cvvis"]

	// Now we map all of the extended keys.....
	capNameToExtended := map[string]Key{
		"kcuu1": {Codepoint: KeyUp},
		"kcuf1": {Codepoint: KeyRight},
		"kcud1": {Codepoint: KeyDown},
		"kcub1": {Codepoint: KeyLeft},
		"kich1": {Codepoint: KeyInsert},
		"kdch1": {Codepoint: KeyDelete},
		"kbs":   {Codepoint: KeyBackspace},
		"knp":   {Codepoint: KeyPgDown},
		"kpp":   {Codepoint: KeyPgUp},
		"khome": {Codepoint: KeyHome},
		"kend":  {Codepoint: KeyEnd},
		"kf0":   {Codepoint: KeyF00},
		"kf1":   {Codepoint: KeyF01},
		"kf2":   {Codepoint: KeyF02},
		"kf3":   {Codepoint: KeyF03},
		"kf4":   {Codepoint: KeyF04},
		"kf5":   {Codepoint: KeyF05},
		"kf6":   {Codepoint: KeyF06},
		"kf7":   {Codepoint: KeyF07},
		"kf8":   {Codepoint: KeyF08},
		"kf9":   {Codepoint: KeyF09},
		"kf10":  {Codepoint: KeyF10},
		"kf11":  {Codepoint: KeyF11},
		"kf12":  {Codepoint: KeyF12},
		"kf13":  {Codepoint: KeyF13},
		"kf14":  {Codepoint: KeyF14},
		"kf15":  {Codepoint: KeyF15},
		"kf16":  {Codepoint: KeyF16},
		"kf17":  {Codepoint: KeyF17},
		"kf18":  {Codepoint: KeyF18},
		"kf19":  {Codepoint: KeyF19},
		"kf20":  {Codepoint: KeyF20},
		"kf21":  {Codepoint: KeyF21},
		"kf22":  {Codepoint: KeyF22},
		"kf23":  {Codepoint: KeyF23},
		"kf24":  {Codepoint: KeyF24},
		"kf25":  {Codepoint: KeyF25},
		"kf26":  {Codepoint: KeyF26},
		"kf27":  {Codepoint: KeyF27},
		"kf28":  {Codepoint: KeyF28},
		"kf29":  {Codepoint: KeyF29},
		"kf30":  {Codepoint: KeyF30},
		"kf31":  {Codepoint: KeyF31},
		"kf32":  {Codepoint: KeyF32},
		"kf33":  {Codepoint: KeyF33},
		"kf34":  {Codepoint: KeyF34},
		"kf35":  {Codepoint: KeyF35},
		"kf36":  {Codepoint: KeyF36},
		"kf37":  {Codepoint: KeyF37},
		"kf38":  {Codepoint: KeyF38},
		"kf39":  {Codepoint: KeyF39},
		"kf40":  {Codepoint: KeyF40},
		"kf41":  {Codepoint: KeyF41},
		"kf42":  {Codepoint: KeyF42},
		"kf43":  {Codepoint: KeyF43},
		"kf44":  {Codepoint: KeyF44},
		"kf45":  {Codepoint: KeyF45},
		"kf46":  {Codepoint: KeyF46},
		"kf47":  {Codepoint: KeyF47},
		"kf48":  {Codepoint: KeyF48},
		"kf49":  {Codepoint: KeyF49},
		"kf50":  {Codepoint: KeyF50},
		"kf51":  {Codepoint: KeyF51},
		"kf52":  {Codepoint: KeyF52},
		"kf53":  {Codepoint: KeyF53},
		"kf54":  {Codepoint: KeyF54},
		"kf55":  {Codepoint: KeyF55},
		"kf56":  {Codepoint: KeyF56},
		"kf57":  {Codepoint: KeyF57},
		"kf58":  {Codepoint: KeyF58},
		"kf59":  {Codepoint: KeyF59},
		"kf60":  {Codepoint: KeyF60},
		"kf61":  {Codepoint: KeyF61},
		"kf62":  {Codepoint: KeyF62},
		"kf63":  {Codepoint: KeyF63},
		"kent":  {Codepoint: KeyEnter},
		"kclr":  {Codepoint: KeyClear},
		"kc1":   {Codepoint: KeyDownLeft},
		"kc3":   {Codepoint: KeyDownRight},
		"ka1":   {Codepoint: KeyUpLeft},
		"ka3":   {Codepoint: KeyUpRight},
		"kb2":   {Codepoint: KeyCenter},
		"kbeg":  {Codepoint: KeyBegin},
		"kcan":  {Codepoint: KeyCancel},
		"kclo":  {Codepoint: KeyClose},
		"kcmd":  {Codepoint: KeyCommand},
		"kcpy":  {Codepoint: KeyCopy},
		"kext":  {Codepoint: KeyExit},
		"kprt":  {Codepoint: KeyPrint},
		"krfr":  {Codepoint: KeyRefresh},
		"kBEG":  {Codepoint: KeyBegin, Modifiers: ModShift},
		"kBEG3": {Codepoint: KeyBegin, Modifiers: ModAlt},
		"kBEG4": {Codepoint: KeyBegin, Modifiers: ModAlt | ModShift},
		"kBEG5": {Codepoint: KeyBegin, Modifiers: ModCtrl},
		"kBEG6": {Codepoint: KeyBegin, Modifiers: ModCtrl | ModShift},
		"kBEG7": {Codepoint: KeyBegin, Modifiers: ModAlt | ModCtrl},
		"kDC":   {Codepoint: KeyDelete, Modifiers: ModShift},
		"kDC3":  {Codepoint: KeyDelete, Modifiers: ModAlt},
		"kDC4":  {Codepoint: KeyDelete, Modifiers: ModAlt | ModShift},
		"kDC5":  {Codepoint: KeyDelete, Modifiers: ModCtrl},
		"kDC6":  {Codepoint: KeyDelete, Modifiers: ModCtrl | ModShift},
		"kDC7":  {Codepoint: KeyDelete, Modifiers: ModAlt | ModCtrl},
		"kDN":   {Codepoint: KeyDown, Modifiers: ModShift},
		"kDN3":  {Codepoint: KeyDown, Modifiers: ModAlt},
		"kDN4":  {Codepoint: KeyDown, Modifiers: ModAlt | ModShift},
		"kDN5":  {Codepoint: KeyDown, Modifiers: ModCtrl},
		"kDN6":  {Codepoint: KeyDown, Modifiers: ModCtrl | ModShift},
		"kDN7":  {Codepoint: KeyDown, Modifiers: ModAlt | ModCtrl},
		"kEND":  {Codepoint: KeyEnd, Modifiers: ModShift},
		"kEND3": {Codepoint: KeyEnd, Modifiers: ModAlt},
		"kEND4": {Codepoint: KeyEnd, Modifiers: ModAlt | ModShift},
		"kEND5": {Codepoint: KeyEnd, Modifiers: ModCtrl},
		"kEND6": {Codepoint: KeyEnd, Modifiers: ModCtrl | ModShift},
		"kEND7": {Codepoint: KeyEnd, Modifiers: ModAlt | ModCtrl},
		"kHOM":  {Codepoint: KeyHome, Modifiers: ModShift},
		"kHOM3": {Codepoint: KeyHome, Modifiers: ModAlt},
		"kHOM4": {Codepoint: KeyHome, Modifiers: ModAlt | ModShift},
		"kHOM5": {Codepoint: KeyHome, Modifiers: ModCtrl},
		"kHOM6": {Codepoint: KeyHome, Modifiers: ModCtrl | ModShift},
		"kHOM7": {Codepoint: KeyHome, Modifiers: ModAlt | ModCtrl},
		"kIC":   {Codepoint: KeyInsert, Modifiers: ModShift},
		"kIC3":  {Codepoint: KeyInsert, Modifiers: ModAlt},
		"kIC4":  {Codepoint: KeyInsert, Modifiers: ModAlt | ModShift},
		"kIC5":  {Codepoint: KeyInsert, Modifiers: ModCtrl},
		"kIC6":  {Codepoint: KeyInsert, Modifiers: ModCtrl | ModShift},
		"kIC7":  {Codepoint: KeyInsert, Modifiers: ModAlt | ModCtrl},
		"kLFT":  {Codepoint: KeyLeft, Modifiers: ModShift},
		"kLFT3": {Codepoint: KeyLeft, Modifiers: ModAlt},
		"kLFT4": {Codepoint: KeyLeft, Modifiers: ModAlt | ModShift},
		"kLFT5": {Codepoint: KeyLeft, Modifiers: ModCtrl},
		"kLFT6": {Codepoint: KeyLeft, Modifiers: ModCtrl | ModShift},
		"kLFT7": {Codepoint: KeyLeft, Modifiers: ModAlt | ModCtrl},
		"kNXT":  {Codepoint: KeyPgDown, Modifiers: ModShift},
		"kNXT3": {Codepoint: KeyPgDown, Modifiers: ModAlt},
		"kNXT4": {Codepoint: KeyPgDown, Modifiers: ModAlt | ModShift},
		"kNXT5": {Codepoint: KeyPgDown, Modifiers: ModCtrl},
		"kNXT6": {Codepoint: KeyPgDown, Modifiers: ModCtrl | ModShift},
		"kNXT7": {Codepoint: KeyPgDown, Modifiers: ModAlt | ModCtrl},
		"kPRV":  {Codepoint: KeyPgUp, Modifiers: ModShift},
		"kPRV3": {Codepoint: KeyPgUp, Modifiers: ModAlt},
		"kPRV4": {Codepoint: KeyPgUp, Modifiers: ModAlt | ModShift},
		"kPRV5": {Codepoint: KeyPgUp, Modifiers: ModCtrl},
		"kPRV6": {Codepoint: KeyPgUp, Modifiers: ModCtrl | ModShift},
		"kPRV7": {Codepoint: KeyPgUp, Modifiers: ModAlt | ModCtrl},
		"kRIT":  {Codepoint: KeyRight, Modifiers: ModShift},
		"kRIT3": {Codepoint: KeyRight, Modifiers: ModAlt},
		"kRIT4": {Codepoint: KeyRight, Modifiers: ModAlt | ModShift},
		"kRIT5": {Codepoint: KeyRight, Modifiers: ModCtrl},
		"kRIT6": {Codepoint: KeyRight, Modifiers: ModCtrl | ModShift},
		"kRIT7": {Codepoint: KeyRight, Modifiers: ModAlt | ModCtrl},
		"kUP":   {Codepoint: KeyUp, Modifiers: ModShift},
		"kUP3":  {Codepoint: KeyUp, Modifiers: ModAlt},
		"kUP4":  {Codepoint: KeyUp, Modifiers: ModAlt | ModShift},
		"kUP5":  {Codepoint: KeyUp, Modifiers: ModCtrl},
		"kUP6":  {Codepoint: KeyUp, Modifiers: ModCtrl | ModShift},
		"kUP7":  {Codepoint: KeyUp, Modifiers: ModAlt | ModCtrl},
	}

	for name, ext := range capNameToExtended {
		val, ok := info.Strings[name]
		if !ok {
			continue
		}
		keyMap[val] = ext
	}

	return nil
}
