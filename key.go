package vaxis

import (
	"bytes"
	"fmt"
	"unicode"

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

	switch k.Keycode {
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
		buf.WriteString("space")
	}

	return buf.String()
}

func decodeKey(seq ansi.Sequence) Key {
	key := Key{}
	switch seq := seq.(type) {
	case ansi.Print:
		raw := rune(seq)
		key.Keycode = raw
		if unicode.IsUpper(raw) {
			key.Keycode = unicode.ToLower(rune(seq))
			key.ShiftedCode = raw
			// It's a shifted character
			key.Modifiers = ModShift
		}
		key.Text = string(raw)
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
						if key.Keycode <= KeyF12 && key.Keycode >= KeyF01 {
							switch key.Modifiers {
							case 1:
								key.Keycode += 12
							case 2:
								key.Keycode += 48
							case 3:
								key.Keycode += 60
							case 4:
								key.Keycode += 24
							case 5:
								key.Keycode += 36
							}
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
