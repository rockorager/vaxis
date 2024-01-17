package vaxis

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

// Key is a key event. Codepoint can be either the literal codepoint of the
// keypress, or a value set by Vaxis to indicate special keys. Special keys have
// their codepoints outside of the valid unicode range
type Key struct {
	// Text is text that the keypress generated
	Text string
	// Keycode is our primary key press. In alternate layouts, this will be
	// the lowercase value of the unicode point
	Keycode rune
	// The shifted keycode of this key event. This will only be non-zero if
	// the shift-modifier was used to generate the event
	ShiftedCode rune
	// BaseLayoutCode is the keycode that would have been generated on a
	// standard PC-101 layout
	BaseLayoutCode rune
	// Modifiers are any key modifier used to generate the event
	Modifiers ModifierMask
	// EventType is the type of key event this was (press, release, repeat,
	// or paste)
	EventType EventType
}

// Matches returns true if there is any way for the passed key and mods to match
// the [Key]. Before matching, ModCapsLock and ModNumLock are removed from the
// modifier mask. Returns true if any of the following are true
//
// 1. Keycode and Modifiers are exact matches
// 2. Text and Modifiers are exact matches
// 3. ShiftedCode and Modifiers (with ModShift removed) are exact matches
// 4. BaseLayoutCode and Modifiers are exact matches
//
// If key is not a letter, but still a graphic: (so we can match ':', which is
// shift+; but we don't want to match "shift+tab" as the same as "tab")
//
// 5. Keycode and Modifers (without ModShift) are exact matches
// 6. Shifted Keycode and Modifers (without ModShift) are exact matches
//
// If key is lowercase or not a letter and mods includes ModShift, uppercase
// Key, remove ModShift and continue
//
// 6. Text and Modifiers are exact matches
func (k Key) Matches(key rune, modifiers ...ModifierMask) bool {
	var mods ModifierMask
	for _, mod := range modifiers {
		mods |= mod
	}
	mods = mods &^ ModCapsLock
	mods = mods &^ ModNumLock
	kMods := k.Modifiers &^ ModCapsLock
	kMods = kMods &^ ModNumLock
	unshiftedkMods := kMods &^ ModShift
	unshiftedMods := mods &^ ModShift

	// Rule 1
	if k.Keycode == key && mods == kMods {
		return true
	}

	// Rule 2
	if k.Text == string(key) && mods == kMods {
		return true
	}

	// Rule 3
	if k.ShiftedCode == key && mods == unshiftedkMods {
		return true
	}

	// Rule 4
	if k.BaseLayoutCode == key && mods == kMods {
		return true
	}

	// Rule 5
	if !unicode.IsLetter(key) && unicode.IsGraphic(key) {
		if k.Keycode == key && unshiftedkMods == unshiftedMods {
			return true
		}
		if k.ShiftedCode == key && unshiftedkMods == unshiftedMods {
			return true
		}
	}

	// Rule 6
	if mods&ModShift != 0 && unicode.IsLower(key) {
		key = unicode.ToUpper(key)
		if k.Text == string(key) && unshiftedMods == unshiftedkMods {
			return true
		}
	}

	return false
}

// MatchString parses a string and matches to the Key event. The syntax for
// strings is: <modifier>[+<modifer>]+<key>. For example:
//
//	Ctrl+p
//	Shift+Alt+Up
//
// All modifiers will be matched lowercase
func (k Key) MatchString(tgt string) bool {
	if tgt == "" {
		return false
	}
	if r, n := utf8.DecodeRuneInString(tgt); n == len(tgt) {
		// fast path if the 'tgt' is a single utf8 codepoint
		return k.Matches(r)
	}
	vals := strings.Split(tgt, "+")
	mods := vals[0 : len(vals)-1]
	key := vals[len(vals)-1]

	var mask ModifierMask
	for _, m := range mods {
		switch strings.ToLower(m) {
		case "shift":
			mask |= ModShift
		case "alt":
			mask |= ModAlt
		case "ctrl":
			mask |= ModCtrl
		case "super":
			mask |= ModSuper
		case "meta":
			mask |= ModMeta
		case "caps":
			mask |= ModCapsLock
		case "num":
			mask |= ModNumLock
		}
	}
	if r, n := utf8.DecodeRuneInString(key); n == len(key) {
		// fast path if the 'key' is unicode
		return k.Matches(r, mask)
	}
	for _, kn := range keyNames {
		if !strings.EqualFold(kn.name, key) {
			continue
		}
		return k.Matches(kn.key, mask)
	}

	// maybe it's a multi-byte, non special character. Grab the first rune
	// and try matching
	for _, r := range key {
		return k.Matches(r, mask)
	}
	// not a match
	return false
}

// ModifierMask is a bitmask for which modifier keys were held when a key was
// pressed
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
	// The key / button was pressed
	EventPress EventType = iota
	// The key / button was repeated
	EventRepeat
	// The key / button was released
	EventRelease
	// A mouse motion event (with or without a button press)
	EventMotion
	// The key resulted from a paste
	EventPaste
)

// String returns a human-readable description of the keypress, suitable for use
// in matching ("Ctrl+c")
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
	case k.Keycode == KeyTab:
	case k.Keycode == KeySpace:
	case k.Keycode == KeyEsc:
	case k.Keycode == KeyBackspace:
	case k.Keycode == KeyEnter:
	case k.Keycode == 0x08:
		k.Keycode = KeyBackspace
	case k.Keycode < 0x00:
		return "invalid"
	case k.Keycode < 0x20:
		var val rune
		switch {
		case k.Keycode == 0x00:
			val = '@'
		case k.Keycode < 0x1A:
			// normalize these to lowercase runes
			val = k.Keycode + 0x60
		case k.Keycode < 0x20:
			val = k.Keycode + 0x40
		}
		return fmt.Sprintf("Ctrl+%c", val)
	case k.Keycode <= unicode.MaxRune:
		if k.Modifiers&ModCapsLock != 0 {
			buf.WriteRune(unicode.ToUpper(k.Keycode))
		} else {
			buf.WriteRune(k.Keycode)
		}
	}

	for _, kn := range keyNames {
		if kn.key != k.Keycode {
			continue
		}
		buf.WriteString(kn.name)
		break
	}

	return buf.String()
}

func decodeKey(seq ansi.Sequence) Key {
	key := Key{}
	switch seq := seq.(type) {
	case ansi.Print:
		// For decoding keys, we take the first rune
		var raw rune
		for _, r := range seq.Grapheme {
			raw = r
			break
		}
		key.Keycode = raw
		if unicode.IsUpper(raw) {
			key.Keycode = unicode.ToLower(raw)
			key.ShiftedCode = raw
			// It's a shifted character
			key.Modifiers = ModShift
		}
		key.Text = seq.Grapheme
		// NOTE: we don't set baselayout code on printed keys. In legacy
		// encodings, this is meaningless. In kitty, this is best used to map
		// keybinds and we should only get ansi.Print types when a paste occurs
	case ansi.C0:
		switch rune(seq) {
		case 0x08:
			key.Keycode = KeyBackspace
		case 0x09:
			key.Keycode = KeyTab
		case 0x0D:
			key.Keycode = KeyEnter
		case 0x1B:
			key.Keycode = KeyEsc
		default:
			key.Modifiers = ModCtrl
			switch {
			case rune(seq) == 0x00:
				key.Keycode = '@'
			case rune(seq) < 0x1A:
				// normalize these to lowercase runes
				key.Keycode = rune(seq) + 0x60
			case rune(seq) < 0x20:
				key.Keycode = rune(seq) + 0x40
			}
		}
	case ansi.ESC:
		key.Keycode = seq.Final
		key.Modifiers = ModAlt
	case ansi.SS3:
		switch rune(seq) {
		case 'A':
			key.Keycode = KeyUp
		case 'B':
			key.Keycode = KeyDown
		case 'C':
			key.Keycode = KeyRight
		case 'D':
			key.Keycode = KeyLeft
		case 'F':
			key.Keycode = KeyEnd
		case 'H':
			key.Keycode = KeyHome
		case 'P':
			key.Keycode = KeyF01
		case 'Q':
			key.Keycode = KeyF02
		case 'R':
			key.Keycode = KeyF03
		case 'S':
			key.Keycode = KeyF04
		}
	case ansi.CSI:
		if len(seq.Parameters) == 0 {
			seq.Parameters = [][]int{
				{1},
			}
		}
		for i, pm := range seq.Parameters {
			switch i {
			case 0:
				for j, ps := range pm {
					switch j {
					case 0:
						// our keycode
						// unicode-key-code
						// This will always be length of at least 1
						sk := specialKey{rune(ps), seq.Final}
						if sk.keycode == 1 && sk.final == 'Z' {
							key.Keycode = KeyTab
							key.Modifiers = ModShift
							continue
						}
						var ok bool
						key.Keycode, ok = specialsKeys[sk]
						if !ok {
							key.Keycode = rune(ps)
						}
					case 1:
						// Shifted keycode
						key.ShiftedCode = rune(ps)
					case 2:
						// Base layout code
						key.BaseLayoutCode = rune(ps)
					}
				}
			case 1:
				// Kitty keyboard protocol reports these as their
				// bitmask + 1, so that an unmodified key has a value of
				// 1. We subtract one to normalize to our internal
				// representation
				for j, ps := range pm {
					switch j {
					case 0:
						// Modifiers
						key.Modifiers = ModifierMask(pm[0] - 1)
						if key.Modifiers < 0 {
							key.Modifiers = 0
						}
					case 1:
						// event type
						//
						key.EventType = EventType(ps) - 1
					}
				}
			case 2:
				// text-as-codepoint
				for _, p := range pm {
					key.Text += string(rune(p))
				}
			}
		}
	}
	return key
}

type specialKey struct {
	keycode rune
	final   rune
}

const (
	extended = unicode.MaxRune + 1
)

const (
	KeyUp rune = extended + 1 + iota
	KeyRight
	KeyDown
	KeyLeft
	KeyInsert
	KeyDelete
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
	KeyEnter     = 0x0D
	KeyReturn    = KeyEnter
	KeyTab       = 0x09
	KeyEsc       = 0x1B
	KeySpace     = 0x20
	KeyBackspace = 0x7F
)

var specialsKeys = map[specialKey]rune{
	{27, 'u'}:    KeyEsc,
	{13, 'u'}:    KeyEnter,
	{9, 'u'}:     KeyTab,
	{127, 'u'}:   KeyBackspace,
	{2, '~'}:     KeyInsert,
	{3, '~'}:     KeyDelete,
	{1, 'D'}:     KeyLeft,
	{1, 'C'}:     KeyRight,
	{1, 'B'}:     KeyDown,
	{1, 'A'}:     KeyUp,
	{5, '~'}:     KeyPgUp,
	{6, '~'}:     KeyPgDown,
	{1, 'F'}:     KeyEnd,
	{8, '~'}:     KeyEnd,
	{1, 'H'}:     KeyHome,
	{7, '~'}:     KeyHome,
	{57358, 'u'}: KeyCapsLock,
	{57359, 'u'}: KeyScrollLock,
	{57360, 'u'}: KeyNumlock,
	{57361, 'u'}: KeyPrintScreen,
	{57362, 'u'}: KeyPause,
	{57363, 'u'}: KeyMenu,
	{1, 'P'}:     KeyF01,
	{11, '~'}:    KeyF01,
	{1, 'Q'}:     KeyF02,
	{12, '~'}:    KeyF02,
	{1, 'R'}:     KeyF03,
	{13, '~'}:    KeyF03,
	{1, 'S'}:     KeyF04,
	{14, '~'}:    KeyF04,
	{15, '~'}:    KeyF05,
	{17, '~'}:    KeyF06,
	{18, '~'}:    KeyF07,
	{19, '~'}:    KeyF08,
	{20, '~'}:    KeyF09,
	{21, '~'}:    KeyF10,
	{23, '~'}:    KeyF11,
	{24, '~'}:    KeyF12,
	{57376, 'u'}: KeyF13,
	{57377, 'u'}: KeyF14,
	{57378, 'u'}: KeyF15,
	{57379, 'u'}: KeyF16,
	{57380, 'u'}: KeyF17,
	{57381, 'u'}: KeyF18,
	{57382, 'u'}: KeyF19,
	{57383, 'u'}: KeyF20,
	{57384, 'u'}: KeyF21,
	{57385, 'u'}: KeyF22,
	{57386, 'u'}: KeyF23,
	{57387, 'u'}: KeyF24,
	{57388, 'u'}: KeyF25,
	{57389, 'u'}: KeyF26,
	{57390, 'u'}: KeyF27,
	{57391, 'u'}: KeyF28,
	{57392, 'u'}: KeyF29,
	{57393, 'u'}: KeyF30,
	{57394, 'u'}: KeyF31,
	{57395, 'u'}: KeyF32,
	{57396, 'u'}: KeyF33,
	{57397, 'u'}: KeyF34,
	{57398, 'u'}: KeyF35,
	// Skip the keypad keys
	{57428, 'u'}: KeyMediaPlay,
	{57429, 'u'}: KeyMediaPause,
	{57430, 'u'}: KeyMediaPlayPause,
	{57431, 'u'}: KeyMediaRev,
	{57432, 'u'}: KeyMediaStop,
	{57433, 'u'}: KeyMediaFF,
	{57434, 'u'}: KeyMediaRewind,
	{57435, 'u'}: KeyMediaNext,
	{57436, 'u'}: KeyMediaPrev,
	{57437, 'u'}: KeyMediaRecord,
	{57438, 'u'}: KeyMediaVolDown,
	{57439, 'u'}: KeyMediaVolUp,
	{57440, 'u'}: KeyMediaMute,
	{57441, 'u'}: KeyLeftShift,
	{57442, 'u'}: KeyLeftControl,
	{57443, 'u'}: KeyLeftAlt,
	{57444, 'u'}: KeyLeftSuper,
	{57445, 'u'}: KeyLeftHyper,
	{57446, 'u'}: KeyLeftMeta,
	{57447, 'u'}: KeyRightShift,
	{57448, 'u'}: KeyRightControl,
	{57449, 'u'}: KeyRightAlt,
	{57450, 'u'}: KeyRightSuper,
	{57451, 'u'}: KeyRightHyper,
	{57452, 'u'}: KeyRightMeta,
	{57453, 'u'}: KeyL3Shift,
	{57454, 'u'}: KeyL5Shift,
}

type keyName struct {
	key  rune
	name string
}

var keyNames = []keyName{
	{KeyUp, "Up"},
	{KeyRight, "Right"},
	{KeyDown, "Down"},
	{KeyLeft, "Left"},
	{KeyInsert, "Insert"},
	{KeyDelete, "Delete"},
	{KeyBackspace, "BackSpace"},
	{KeyPgDown, "Page_Down"},
	{KeyPgUp, "Page_Up"},
	{KeyHome, "Home"},
	{KeyEnd, "End"},
	{KeyF00, "F0"},
	{KeyF01, "F1"},
	{KeyF02, "F2"},
	{KeyF03, "F3"},
	{KeyF04, "F4"},
	{KeyF05, "F5"},
	{KeyF06, "F6"},
	{KeyF07, "F7"},
	{KeyF08, "F8"},
	{KeyF09, "F9"},
	{KeyF10, "F10"},
	{KeyF11, "F11"},
	{KeyF12, "F12"},
	{KeyF13, "F13"},
	{KeyF14, "F14"},
	{KeyF15, "F15"},
	{KeyF16, "F16"},
	{KeyF17, "F17"},
	{KeyF18, "F18"},
	{KeyF19, "F19"},
	{KeyF20, "F20"},
	{KeyF21, "F21"},
	{KeyF22, "F22"},
	{KeyF23, "F23"},
	{KeyF24, "F24"},
	{KeyF25, "F25"},
	{KeyF26, "F26"},
	{KeyF27, "F27"},
	{KeyF28, "F28"},
	{KeyF29, "F29"},
	{KeyF30, "F30"},
	{KeyF31, "F31"},
	{KeyF32, "F32"},
	{KeyF33, "F33"},
	{KeyF34, "F34"},
	{KeyF35, "F35"},
	{KeyF36, "F36"},
	{KeyF37, "F37"},
	{KeyF38, "F38"},
	{KeyF39, "F39"},
	{KeyF40, "F40"},
	{KeyF41, "F41"},
	{KeyF42, "F42"},
	{KeyF43, "F43"},
	{KeyF44, "F44"},
	{KeyF45, "F45"},
	{KeyF46, "F46"},
	{KeyF47, "F47"},
	{KeyF48, "F48"},
	{KeyF49, "F49"},
	{KeyF50, "F50"},
	{KeyF51, "F51"},
	{KeyF52, "F52"},
	{KeyF53, "F53"},
	{KeyF54, "F54"},
	{KeyF55, "F55"},
	{KeyF56, "F56"},
	{KeyF57, "F57"},
	{KeyF58, "F58"},
	{KeyF59, "F59"},
	{KeyF60, "F60"},
	{KeyF61, "F61"},
	{KeyF62, "F62"},
	{KeyF63, "F63"},
	{KeyEnter, "Enter"},
	{KeyClear, "Clear"},
	{KeyDownLeft, "DownLeft"},
	{KeyDownRight, "DownRight"},
	{KeyUpLeft, "UpLeft"},
	{KeyUpRight, "UpRight"},
	{KeyCenter, "Center"},
	{KeyBegin, "Begin"},
	{KeyCancel, "Cancel"},
	{KeyClose, "Close"},
	{KeyCommand, "Cmd"},
	{KeyCopy, "Copy"},
	{KeyExit, "Exit"},
	{KeyPrint, "Print"},
	{KeyRefresh, "Refresh"},
	{KeyCapsLock, "Caps_Lock"},
	{KeyScrollLock, "Scroll_Lock"},
	{KeyNumlock, "Num_Lock"},
	{KeyPrintScreen, "Print"},
	{KeyPause, "Pause"},
	{KeyMenu, "Menu"},
	{KeyMediaPlay, "Media_Play"},
	{KeyMediaPause, "Media_Pause"},
	{KeyMediaPlayPause, "Media_Play_Pause"},
	{KeyMediaRev, "Media_Reverse"},
	{KeyMediaStop, "Media_Stop"},
	{KeyMediaFF, "Media_Fast_Forward"},
	{KeyMediaRewind, "Media_Rewind"},
	{KeyMediaNext, "Media_Track_Next"},
	{KeyMediaPrev, "Media_Track_Previous"},
	{KeyMediaRecord, "Media_Record"},
	{KeyMediaVolDown, "Lower_Volume"},
	{KeyMediaVolUp, "Raise_Volume"},
	{KeyMediaMute, "Mute_Volume"},
	{KeyLeftShift, "Shift_L"},
	{KeyLeftControl, "Control_L"},
	{KeyLeftAlt, "Alt_L"},
	{KeyLeftSuper, "Super_L"},
	{KeyLeftHyper, "Hyper_L"},
	{KeyLeftMeta, "Meta_L"},
	{KeyRightShift, "Shift_R"},
	{KeyRightControl, "Control_R"},
	{KeyRightAlt, "Alt_R"},
	{KeyRightSuper, "Super_R"},
	{KeyRightHyper, "Hyper_R"},
	{KeyRightMeta, "Meta_R"},
	{KeyL3Shift, "ISO_Level3_Shift"},
	{KeyL5Shift, "ISO_Level5_Shift"},
	{KeyTab, "Tab"},
	{KeyEsc, "Escape"},
	{KeySpace, "space"},
}
