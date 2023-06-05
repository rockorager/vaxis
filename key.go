package rtk

import (
	"bytes"
	"fmt"
	"unicode"

	"git.sr.ht/~rockorager/rtk/ansi"
)

type Key struct {
	Codepoint rune
	Modifiers ModifierMask
	EventType EventType
}

type ModifierMask int

const (
	// Values equivalent to kitty keyboard protocol
	ModShift ModifierMask = 1 << iota
	ModAlt
	ModCtrl
	ModSuper
	ModHyper
	ModMeta
	ModCapsLock
	ModNumLock
)

// EventType is an input event type (press, repeat, release, etc)
type EventType int

const (
	// The event type could not be determined
	EventUnknown EventType = iota
	// The key / button was pressed
	EventPress
	// The key / button was repeated
	EventRepeat
	// The key / button was released
	EventRelease
	// A mouse motion event (with no button pressed)
	EventMotion
)

// Modified keys will always have prefixes in this order:
//
//	<num-caps-meta-hyper-super-c-a-s-{key}>
func (k Key) String() string {
	buf := &bytes.Buffer{}

	if k.EventType != EventRelease {
		// if k.Modifiers&ModNumLock != 0 {
		// 	buf.WriteString("num-")
		// }
		// if k.Modifiers&ModCapsLock != 0 {
		// 	buf.WriteString("caps-")
		// }
		if k.Modifiers&ModMeta != 0 {
			buf.WriteString("Meta+")
		}
		if k.Modifiers&ModHyper != 0 {
			buf.WriteString("Hyper+")
		}
		if k.Modifiers&ModSuper != 0 {
			buf.WriteString("Super+")
		}
		if k.Modifiers&ModCtrl != 0 {
			buf.WriteString("Ctrl+")
		}
		if k.Modifiers&ModAlt != 0 {
			buf.WriteString("Alt+")
		}
		if k.Modifiers&ModShift != 0 {
			buf.WriteString("Shift+")
		}
	}

	switch {
	case k.Codepoint == KeyTab:
		// Handle further down
	case k.Codepoint == KeySpace:
		// Handle further down
	case k.Codepoint == KeyEsc:
		// Handle further down
	case k.Codepoint == KeyBackspace:
		// handle further down
	case k.Codepoint == KeyEnter:
		// Handle further down
	case k.Codepoint < 0x00:
		return "invalid"
	case k.Codepoint < 0x20:
		var val rune
		switch {
		case k.Codepoint == 0x00:
			val = '@'
		case k.Codepoint < 0x1A:
			// normalize these to lowercase runes
			val = k.Codepoint + 0x60
		case k.Codepoint < 0x20:
			val = k.Codepoint + 0x40
		}
		return fmt.Sprintf("Ctrl+%c", val)
	case k.Codepoint <= unicode.MaxRune:
		buf.WriteRune(k.Codepoint)
	}

	switch k.Codepoint {
	case KeyUp:
		buf.WriteString("Up")
	case KeyRight:
		buf.WriteString("Right")
	case KeyDown:
		buf.WriteString("Down")
	case KeyLeft:
		buf.WriteString("Left")
	case KeyInsert:
		buf.WriteString("Insert")
	case KeyDelete:
		buf.WriteString("Delete")
	case KeyBackspace:
		buf.WriteString("BackSpace")
	case KeyPgDown:
		buf.WriteString("Page_Down")
	case KeyPgUp:
		buf.WriteString("Page_Up")
	case KeyHome:
		buf.WriteString("Home")
	case KeyEnd:
		buf.WriteString("End")
	case KeyF00:
		buf.WriteString("F0")
	case KeyF01:
		buf.WriteString("F1")
	case KeyF02:
		buf.WriteString("F2")
	case KeyF03:
		buf.WriteString("F3")
	case KeyF04:
		buf.WriteString("F4")
	case KeyF05:
		buf.WriteString("F5")
	case KeyF06:
		buf.WriteString("F6")
	case KeyF07:
		buf.WriteString("F7")
	case KeyF08:
		buf.WriteString("F8")
	case KeyF09:
		buf.WriteString("F9")
	case KeyF10:
		buf.WriteString("F10")
	case KeyF11:
		buf.WriteString("F11")
	case KeyF12:
		buf.WriteString("F12")
	case KeyF13:
		buf.WriteString("F13")
	case KeyF14:
		buf.WriteString("F14")
	case KeyF15:
		buf.WriteString("F15")
	case KeyF16:
		buf.WriteString("F16")
	case KeyF17:
		buf.WriteString("F17")
	case KeyF18:
		buf.WriteString("F18")
	case KeyF19:
		buf.WriteString("F19")
	case KeyF20:
		buf.WriteString("F20")
	case KeyF21:
		buf.WriteString("F21")
	case KeyF22:
		buf.WriteString("F22")
	case KeyF23:
		buf.WriteString("F23")
	case KeyF24:
		buf.WriteString("F24")
	case KeyF25:
		buf.WriteString("F25")
	case KeyF26:
		buf.WriteString("F26")
	case KeyF27:
		buf.WriteString("F27")
	case KeyF28:
		buf.WriteString("F28")
	case KeyF29:
		buf.WriteString("F29")
	case KeyF30:
		buf.WriteString("F30")
	case KeyF31:
		buf.WriteString("F31")
	case KeyF32:
		buf.WriteString("F32")
	case KeyF33:
		buf.WriteString("F33")
	case KeyF34:
		buf.WriteString("F34")
	case KeyF35:
		buf.WriteString("F35")
	case KeyF36:
		buf.WriteString("F36")
	case KeyF37:
		buf.WriteString("F37")
	case KeyF38:
		buf.WriteString("F38")
	case KeyF39:
		buf.WriteString("F39")
	case KeyF40:
		buf.WriteString("F40")
	case KeyF41:
		buf.WriteString("F41")
	case KeyF42:
		buf.WriteString("F42")
	case KeyF43:
		buf.WriteString("F43")
	case KeyF44:
		buf.WriteString("F44")
	case KeyF45:
		buf.WriteString("F45")
	case KeyF46:
		buf.WriteString("F46")
	case KeyF47:
		buf.WriteString("F47")
	case KeyF48:
		buf.WriteString("F48")
	case KeyF49:
		buf.WriteString("F49")
	case KeyF50:
		buf.WriteString("F50")
	case KeyF51:
		buf.WriteString("F51")
	case KeyF52:
		buf.WriteString("F52")
	case KeyF53:
		buf.WriteString("F53")
	case KeyF54:
		buf.WriteString("F54")
	case KeyF55:
		buf.WriteString("F55")
	case KeyF56:
		buf.WriteString("F56")
	case KeyF57:
		buf.WriteString("F57")
	case KeyF58:
		buf.WriteString("F58")
	case KeyF59:
		buf.WriteString("F59")
	case KeyF60:
		buf.WriteString("F60")
	case KeyF61:
		buf.WriteString("F61")
	case KeyF62:
		buf.WriteString("F62")
	case KeyF63:
		buf.WriteString("F63")
	case KeyEnter:
		buf.WriteString("Enter")
	case KeyClear:
		buf.WriteString("Clear")
	case KeyDownLeft:
		buf.WriteString("DownLeft")
	case KeyDownRight:
		buf.WriteString("DownRight")
	case KeyUpLeft:
		buf.WriteString("UpLeft")
	case KeyUpRight:
		buf.WriteString("UpRight")
	case KeyCenter:
		buf.WriteString("Center")
	case KeyBegin:
		buf.WriteString("Begin")
	case KeyCancel:
		buf.WriteString("Cancel")
	case KeyClose:
		buf.WriteString("Close")
	case KeyCommand:
		buf.WriteString("Cmd")
	case KeyCopy:
		buf.WriteString("Copy")
	case KeyExit:
		buf.WriteString("Exit")
	case KeyPrint:
		buf.WriteString("Print")
	case KeyRefresh:
		buf.WriteString("Refresh")
		// notcurses says these are only avaialbe in kitty kbp:
	case KeyCapsLock:
		buf.WriteString("Caps_Lock")
	case KeyScrollLock:
		buf.WriteString("Scroll_Lock")
	case KeyNumlock:
		buf.WriteString("Num_Lock")
	case KeyPrintScreen:
		buf.WriteString("Print")
	case KeyPause:
		buf.WriteString("Pause")
	case KeyMenu:
		buf.WriteString("Menu")
		// Media keys, also generally only kitty kbp:
	case KeyMediaPlay:
		buf.WriteString("Media_Play")
	case KeyMediaPause:
		buf.WriteString("Media_Pause")
	case KeyMediaPlayPause:
		buf.WriteString("Media_Play_Pause")
	case KeyMediaRev:
		buf.WriteString("Media_Reverse")
	case KeyMediaStop:
		buf.WriteString("Media_Stop")
	case KeyMediaFF:
		buf.WriteString("Media_Fast_Forward")
	case KeyMediaRewind:
		buf.WriteString("Media_Rewind")
	case KeyMediaNext:
		buf.WriteString("Media_Track_Next")
	case KeyMediaPrev:
		buf.WriteString("Media_Track_Previous")
	case KeyMediaRecord:
		buf.WriteString("Media_Record")
	case KeyMediaVolDown:
		buf.WriteString("Lower_Volume")
	case KeyMediaVolUp:
		buf.WriteString("Raise_Volume")
	case KeyMediaMute:
		buf.WriteString("Mute_Volume")
	// Modifiers, when pressed by themselves
	case KeyLeftShift:
		buf.WriteString("Shift_L")
	case KeyLeftControl:
		buf.WriteString("Control_L")
	case KeyLeftAlt:
		buf.WriteString("Alt_L")
	case KeyLeftSuper:
		buf.WriteString("Super_L")
	case KeyLeftHyper:
		buf.WriteString("Hyper_L")
	case KeyLeftMeta:
		buf.WriteString("Meta_L")
	case KeyRightShift:
		buf.WriteString("Shift_R")
	case KeyRightControl:
		buf.WriteString("Control_R")
	case KeyRightAlt:
		buf.WriteString("Alt_R")
	case KeyRightSuper:
		buf.WriteString("Super_R")
	case KeyRightHyper:
		buf.WriteString("Hyper_R")
	case KeyRightMeta:
		buf.WriteString("Meta_R")
	case KeyL3Shift:
		buf.WriteString("ISO_Level3_Shift")
	case KeyL5Shift:
		buf.WriteString("ISO_Level5_Shift")
	// Aliases
	case KeyTab:
		buf.WriteString("Tab")
	case KeyEsc:
		buf.WriteString("Escape")
	case KeySpace:
		buf.WriteString("Space")
	}

	return buf.String()
}

const (
	extended rune = 1 << 30
)

const (
	KeyUp rune = extended + 1 + iota
	KeyRight
	KeyDown
	KeyLeft
	KeyInsert
	KeyDelete
	KeyBackspace
	KeyPgDown
	KeyPgUp
	KeyHome
	KeyEnd
	KeyF00
	KeyF01
	KeyF02
	KeyF03
	KeyF04
	KeyF05
	KeyF06
	KeyF07
	KeyF08
	KeyF09
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24
	KeyF25
	KeyF26
	KeyF27
	KeyF28
	KeyF29
	KeyF30
	KeyF31
	KeyF32
	KeyF33
	KeyF34
	KeyF35
	KeyF36
	KeyF37
	KeyF38
	KeyF39
	KeyF40
	KeyF41
	KeyF42
	KeyF43
	KeyF44
	KeyF45
	KeyF46
	KeyF47
	KeyF48
	KeyF49
	KeyF50
	KeyF51
	KeyF52
	KeyF53
	KeyF54
	KeyF55
	KeyF56
	KeyF57
	KeyF58
	KeyF59
	KeyF60
	KeyF61
	KeyF62
	KeyF63 // F63 is max defined in terminfo
	KeyClear
	KeyDownLeft
	KeyDownRight
	KeyUpLeft
	KeyUpRight
	KeyCenter
	KeyBegin
	KeyCancel
	KeyClose
	KeyCommand
	KeyCopy
	KeyExit
	KeyPrint
	KeyRefresh
	// notcurses says these are only avaialbe in kitty kbp
	KeyCapsLock
	KeyScrollLock
	KeyNumlock
	KeyPrintScreen
	KeyPause
	KeyMenu
	// Media keys, also generally only kitty kbp
	KeyMediaPlay
	KeyMediaPause
	KeyMediaPlayPause
	KeyMediaRev
	KeyMediaStop
	KeyMediaFF
	KeyMediaRewind
	KeyMediaNext
	KeyMediaPrev
	KeyMediaRecord
	KeyMediaVolDown
	KeyMediaVolUp
	KeyMediaMute
	// Modifiers, when pressed by themselves
	KeyLeftShift
	KeyLeftControl
	KeyLeftAlt
	KeyLeftSuper
	KeyLeftHyper
	KeyLeftMeta
	KeyRightShift
	KeyRightControl
	KeyRightAlt
	KeyRightSuper
	KeyRightHyper
	KeyRightMeta
	KeyL3Shift
	KeyL5Shift

	// Aliases
	KeyEnter  = 0x0D
	KeyReturn = KeyEnter
	KeyTab    = 0x09
	KeyEsc    = 0x1B
	KeySpace  = 0x20
)

// keyMap is built from terminfo entries
var keyMap = map[string]Key{}

var kittyKeyMap = map[string]rune{
	"27u":    KeyEsc,
	"13u":    KeyEnter,
	"9u":     KeyTab,
	"127u":   KeyBackspace,
	"2~":     KeyInsert,
	"3~":     KeyDelete,
	"1D":     KeyLeft,
	"1C":     KeyRight,
	"1B":     KeyDown,
	"1A":     KeyUp,
	"5~":     KeyPgUp,
	"6~":     KeyPgDown,
	"1F":     KeyEnd,
	"8~":     KeyEnd,
	"1H":     KeyHome,
	"7~":     KeyHome,
	"57358u": KeyCapsLock,
	"57359u": KeyScrollLock,
	"57360u": KeyNumlock,
	"57361u": KeyPrintScreen,
	"57362u": KeyPause,
	"57363u": KeyMenu,
	"1P":     KeyF01,
	"11~":    KeyF01,
	"1Q":     KeyF02,
	"12~":    KeyF02,
	"13~":    KeyF03,
	"1S":     KeyF04,
	"14~":    KeyF04,
	"15~":    KeyF05,
	"17~":    KeyF06,
	"18~":    KeyF07,
	"19~":    KeyF08,
	"20~":    KeyF09,
	"21~":    KeyF10,
	"23~":    KeyF11,
	"24~":    KeyF12,
	"57376u": KeyF13,
	"57377u": KeyF14,
	"57378u": KeyF15,
	"57379u": KeyF16,
	"57380u": KeyF17,
	"57381u": KeyF18,
	"57382u": KeyF19,
	"57383u": KeyF20,
	"57384u": KeyF21,
	"57385u": KeyF22,
	"57386u": KeyF23,
	"57387u": KeyF24,
	"57388u": KeyF25,
	"57389u": KeyF26,
	"57390u": KeyF27,
	"57391u": KeyF28,
	"57392u": KeyF29,
	"57393u": KeyF30,
	"57394u": KeyF31,
	"57395u": KeyF32,
	"57396u": KeyF33,
	"57397u": KeyF34,
	"57398u": KeyF35,
	// Skip the keypad keys
	"57428u": KeyMediaPlay,
	"57429u": KeyMediaPause,
	"57430u": KeyMediaPlayPause,
	"57431u": KeyMediaRev,
	"57432u": KeyMediaStop,
	"57433u": KeyMediaFF,
	"57434u": KeyMediaRewind,
	"57435u": KeyMediaNext,
	"57436u": KeyMediaPrev,
	"57437u": KeyMediaRecord,
	"57438u": KeyMediaVolDown,
	"57439u": KeyMediaVolUp,
	"57440u": KeyMediaMute,
	"57441u": KeyLeftShift,
	"57442u": KeyLeftControl,
	"57443u": KeyLeftAlt,
	"57444u": KeyLeftSuper,
	"57445u": KeyLeftHyper,
	"57446u": KeyLeftMeta,
	"57447u": KeyRightShift,
	"57448u": KeyRightControl,
	"57449u": KeyRightAlt,
	"57450u": KeyRightSuper,
	"57451u": KeyRightHyper,
	"57452u": KeyRightMeta,
	"57453u": KeyL3Shift,
	"57454u": KeyL5Shift,
}

func parseKittyKbp(seq ansi.CSI) Key {
	key := Key{}
	switch seq.Final {
	case 'u', '~', 'A', 'B', 'C', 'D', 'E', 'F', 'H', 'P', 'Q', 'S':
	default:
		return key
	}

	switch len(seq.Parameters) {
	case 0:
		seq.Parameters = [][]int{
			{1},
			{1, 1},
		}
	case 1:
		seq.Parameters = append(seq.Parameters, []int{1, 1})
	}

	for i, pm := range seq.Parameters {
		switch i {
		case 0:
			// unicode-key-code
			// This will always be length of 1. We haven't requested
			// alternate-keys, which would make the length
			// longer...we don't care about those. We translate this
			// codepoint to an internal key below
			base := fmt.Sprintf("%d%c", pm[0], seq.Final)
			var ok bool
			key.Codepoint, ok = kittyKeyMap[base]
			if !ok {
				key.Codepoint = rune(pm[0])
			}
		case 1:
			// Kitty keyboard protocol reports these as their
			// bitmask + 1, so that an unmodified key has a value of
			// 1. We subtract one to normalize to our internal
			// representation
			key.Modifiers = ModifierMask(pm[0] - 1)
			if len(pm) > 1 {
				key.EventType = EventType(pm[1])
			}
		case 2:
			// text-as-codepoint
			key.Codepoint = rune(pm[0])
			key.Modifiers = 0
		}
	}
	return key
}
