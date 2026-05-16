package term

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

const (
	kittyKeyboardFlagReportEvents     = 1 << 1
	kittyKeyboardFlagReportAlternates = 1 << 2
	kittyKeyboardFlagReportAll        = 1 << 3
	kittyKeyboardFlagAssociatedText   = 1 << 4
)

func encodeXterm(key vaxis.Key, deckpam bool, decckm bool, decbkm bool, ignoreKeypadWithNumLock bool, altEscPrefix bool) string {
	// ignore any kitty mods
	xtermMods := key.Modifiers & vaxis.ModShift
	xtermMods |= key.Modifiers & vaxis.ModAlt
	xtermMods |= key.Modifiers & vaxis.ModCtrl

	if key.Keycode == vaxis.KeyBackspace {
		seq := "\x7f"
		if decbkm {
			seq = "\x08"
		}
		if xtermMods&vaxis.ModCtrl != 0 {
			if decbkm {
				seq = "\x7f"
			} else {
				seq = "\x08"
			}
		}
		if xtermMods&vaxis.ModAlt != 0 {
			return "\x1b" + seq
		}
		return seq
	}

	if xtermMods == 0 {
		// function keys
		if val, ok := keymap[key.Keycode]; ok {
			return val
		}
		switch decckm {
		case true:
			if val, ok := cursorKeysApplicationMode[key.Keycode]; ok {
				return val
			}
		case false:
			if val, ok := cursorKeysNormalMode[key.Keycode]; ok {
				return val
			}
		}

		if ignoreKeypadWithNumLock {
			deckpam = false
		}
		switch deckpam {
		case true:
			// Special keys
			if val, ok := applicationKeymap[key.Keycode]; ok {
				return val
			}
		case false:
			// Special keys
			if val, ok := numericKeymap[key.Keycode]; ok {
				return val
			}
		}

		if key.Keycode < unicode.MaxRune {
			// Unicode keys
			return string(key.Keycode)
		}
	}

	if val, ok := xtermKeymap[key.Keycode]; ok {
		return fmt.Sprintf("\x1B[%d;%d%c", val.number, int(xtermMods)+1, val.final)
	}

	if key.Text != "" && key.Modifiers&vaxis.ModCtrl == 0 && key.Modifiers&vaxis.ModAlt == 0 {
		return key.Text
	}

	buf := bytes.NewBuffer(nil)
	if key.Keycode < unicode.MaxRune {
		if xtermMods&vaxis.ModAlt != 0 && altEscPrefix {
			buf.WriteRune('\x1b')
		}
		if xtermMods&vaxis.ModCtrl != 0 {
			if unicode.IsLower(key.Keycode) {
				buf.WriteRune(key.Keycode - 0x60)
				return buf.String()
			}
			switch key.Keycode {
			case '1':
				buf.WriteRune('1')
			case '2':
				buf.WriteRune(0x00)
			case '3':
				buf.WriteRune(0x1b)
			case '4':
				buf.WriteRune(0x1c)
			case '5':
				buf.WriteRune(0x1d)
			case '6':
				buf.WriteRune(0x1e)
			case '7':
				buf.WriteRune(0x1f)
			case '8':
				buf.WriteRune(0x7f)
			case '9':
			default:
				buf.WriteRune(key.Keycode - 0x40)
			}
			return buf.String()
		}
		if xtermMods&vaxis.ModShift != 0 {
			if key.ShiftedCode > 0 {
				buf.WriteRune(key.ShiftedCode)
			} else {
				buf.WriteRune(key.Keycode)
			}
			return buf.String()
		}
		buf.WriteRune(key.Keycode)
		return buf.String()
	}
	return ""
}

func (vt *Model) encodeKey(key vaxis.Key) string {
	flags := uint8(0)
	if vt.EnableKittyKeyboard {
		flags = vt.activeKittyKeyboard().current()
	}
	if seq := encodeKitty(key, flags); seq != "" {
		return seq
	}
	if key.EventType == vaxis.EventRepeat || key.EventType == vaxis.EventRelease {
		return ""
	}
	if vt.mode.modifyOtherKeys2 {
		if seq := encodeModifyOtherKeys(key); seq != "" {
			return seq
		}
	}
	return encodeXterm(key, vt.mode.deckpam, vt.mode.decckm, vt.mode.decbkm, vt.mode.ignoreKeypadWithNumLock, vt.mode.altEscPrefix)
}

func encodeModifyOtherKeys(key vaxis.Key) string {
	codepoint, ok := modifyOtherKeysCodepoint(key)
	if !ok {
		return ""
	}
	mods := key.Modifiers & (vaxis.ModShift | vaxis.ModAlt | vaxis.ModCtrl | vaxis.ModSuper)
	if !shouldModifyOtherKey(codepoint, mods) {
		return ""
	}
	return fmt.Sprintf("\x1B[27;%d;%d~", int(mods)+1, codepoint)
}

func modifyOtherKeysCodepoint(key vaxis.Key) (rune, bool) {
	if key.Text != "" {
		r, size := utf8.DecodeRuneInString(key.Text)
		if r != utf8.RuneError && size == len(key.Text) {
			return r, true
		}
	}
	if key.Modifiers&vaxis.ModShift != 0 && key.ShiftedCode > 0 {
		return key.ShiftedCode, true
	}
	if key.Keycode < unicode.MaxRune {
		return key.Keycode, true
	}
	return 0, false
}

func shouldModifyOtherKey(codepoint rune, mods vaxis.ModifierMask) bool {
	if codepoint >= 0x40 && codepoint <= 0x7F {
		return true
	}
	if mods&^vaxis.ModShift != 0 {
		return true
	}
	return mods == vaxis.ModShift && codepoint == ' '
}

func (vt *Model) modifyKeyFormat(seq ansi.CSI) {
	switch seq.NumParameters {
	case 0:
		vt.mode.modifyOtherKeys2 = false
	case 1:
		switch seq.Parameters[0] {
		case 0, 1, 2, 4:
			vt.mode.modifyOtherKeys2 = false
		}
	case 2:
		switch seq.Parameters[0] {
		case 0, 1, 2:
			vt.mode.modifyOtherKeys2 = false
		case 4:
			vt.mode.modifyOtherKeys2 = seq.Parameters[1] == 2
		}
	}
}

func encodeKitty(key vaxis.Key, flags uint8) string {
	if flags == 0 {
		return ""
	}
	if key.EventType == vaxis.EventRepeat || key.EventType == vaxis.EventRelease {
		if flags&kittyKeyboardFlagReportEvents == 0 {
			return ""
		}
	}

	if val, ok := xtermKeymap[key.Keycode]; ok {
		if flags&kittyKeyboardFlagReportEvents != 0 {
			return fmt.Sprintf("\x1B[%d;%d:%d%c", val.number, int(key.Modifiers)+1, int(key.EventType)+1, val.final)
		}
		return ""
	}

	if key.Keycode >= unicode.MaxRune {
		return ""
	}

	needsKitty := flags&kittyKeyboardFlagReportAll != 0 ||
		key.Modifiers != 0 ||
		key.EventType == vaxis.EventRepeat ||
		key.EventType == vaxis.EventRelease ||
		key.Keycode == vaxis.KeyEsc ||
		key.Keycode == vaxis.KeyEnter ||
		key.Keycode == vaxis.KeyTab ||
		key.Keycode == vaxis.KeyBackspace
	if !needsKitty {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "\x1B[%d", key.Keycode)
	if flags&kittyKeyboardFlagReportAlternates != 0 {
		switch {
		case key.ShiftedCode > 0 && key.BaseLayoutCode > 0:
			fmt.Fprintf(buf, ":%d:%d", key.ShiftedCode, key.BaseLayoutCode)
		case key.ShiftedCode > 0:
			fmt.Fprintf(buf, ":%d", key.ShiftedCode)
		case key.BaseLayoutCode > 0:
			fmt.Fprintf(buf, ":0:%d", key.BaseLayoutCode)
		}
	}

	fmt.Fprintf(buf, ";%d", int(key.Modifiers)+1)
	if flags&kittyKeyboardFlagReportEvents != 0 {
		fmt.Fprintf(buf, ":%d", int(key.EventType)+1)
	}
	if flags&kittyKeyboardFlagAssociatedText != 0 && key.Text != "" {
		for _, r := range key.Text {
			fmt.Fprintf(buf, ";%d", r)
		}
	}
	buf.WriteRune('u')
	return buf.String()
}

type keycode struct {
	number int
	final  rune
}

var xtermKeymap = map[rune]keycode{
	vaxis.KeyUp:     {1, 'A'},
	vaxis.KeyDown:   {1, 'B'},
	vaxis.KeyRight:  {1, 'C'},
	vaxis.KeyLeft:   {1, 'D'},
	vaxis.KeyEnd:    {1, 'F'},
	vaxis.KeyHome:   {1, 'H'},
	vaxis.KeyInsert: {2, '~'},
	vaxis.KeyDelete: {3, '~'},
	vaxis.KeyPgUp:   {5, '~'},
	vaxis.KeyPgDown: {6, '~'},
	vaxis.KeyF01:    {1, 'P'},
	vaxis.KeyF02:    {1, 'Q'},
	vaxis.KeyF03:    {1, 'R'},
	vaxis.KeyF04:    {1, 'S'},
	vaxis.KeyF05:    {15, '~'},
	vaxis.KeyF06:    {17, '~'},
	vaxis.KeyF07:    {18, '~'},
	vaxis.KeyF08:    {19, '~'},
	vaxis.KeyF09:    {20, '~'},
	vaxis.KeyF10:    {21, '~'},
	vaxis.KeyF11:    {23, '~'},
	vaxis.KeyF12:    {24, '~'},
}

var cursorKeysApplicationMode = map[rune]string{
	vaxis.KeyUp:    "\x1BOA",
	vaxis.KeyDown:  "\x1BOB",
	vaxis.KeyRight: "\x1BOC",
	vaxis.KeyLeft:  "\x1BOD",
	vaxis.KeyEnd:   "\x1BOF",
	vaxis.KeyHome:  "\x1BOH",
}

var cursorKeysNormalMode = map[rune]string{
	vaxis.KeyUp:    "\x1B[A",
	vaxis.KeyDown:  "\x1B[B",
	vaxis.KeyRight: "\x1B[C",
	vaxis.KeyLeft:  "\x1B[D",
	vaxis.KeyEnd:   "\x1B[F",
	vaxis.KeyHome:  "\x1B[H",
}

// TODO are these needed? can we even detect this from the host? I guess we can
// with kitty keyboard enabled on host but not in subterm. Double check keypad
// arrows in application mode vs other arrows (CSI vs SS3?)
var numericKeymap = map[rune]string{
	vaxis.KeyInsert:         "\x1B[2~",
	vaxis.KeyDelete:         "\x1B[3~",
	vaxis.KeyPgUp:           "\x1B[5~",
	vaxis.KeyPgDown:         "\x1B[6~",
	vaxis.KeyKeyPad0:        "0",
	vaxis.KeyKeyPad1:        "1",
	vaxis.KeyKeyPad2:        "2",
	vaxis.KeyKeyPad3:        "3",
	vaxis.KeyKeyPad4:        "4",
	vaxis.KeyKeyPad5:        "5",
	vaxis.KeyKeyPad6:        "6",
	vaxis.KeyKeyPad7:        "7",
	vaxis.KeyKeyPad8:        "8",
	vaxis.KeyKeyPad9:        "9",
	vaxis.KeyKeyPadDecimal:  ".",
	vaxis.KeyKeyPadDivide:   "/",
	vaxis.KeyKeyPadMultiply: "*",
	vaxis.KeyKeyPadSubtract: "-",
	vaxis.KeyKeyPadAdd:      "+",
	vaxis.KeyKeyPadEnter:    "\r",
	vaxis.KeyKeyPadUp:       "\x1B[A",
	vaxis.KeyKeyPadDown:     "\x1B[B",
	vaxis.KeyKeyPadRight:    "\x1B[C",
	vaxis.KeyKeyPadLeft:     "\x1B[D",
	vaxis.KeyKeyPadBegin:    "\x1B[E",
	vaxis.KeyKeyPadHome:     "\x1B[H",
	vaxis.KeyKeyPadEnd:      "\x1B[F",
	vaxis.KeyKeyPadInsert:   "\x1B[2~",
	vaxis.KeyKeyPadDelete:   "\x1B[3~",
	vaxis.KeyKeyPadPageUp:   "\x1B[5~",
	vaxis.KeyKeyPadPageDown: "\x1B[6~",
}

var applicationKeymap = map[rune]string{
	vaxis.KeyInsert:         "\x1B[2~",
	vaxis.KeyDelete:         "\x1B[3~",
	vaxis.KeyPgUp:           "\x1B[5~",
	vaxis.KeyPgDown:         "\x1B[6~",
	vaxis.KeyKeyPad0:        "\x1BOp",
	vaxis.KeyKeyPad1:        "\x1BOq",
	vaxis.KeyKeyPad2:        "\x1BOr",
	vaxis.KeyKeyPad3:        "\x1BOs",
	vaxis.KeyKeyPad4:        "\x1BOt",
	vaxis.KeyKeyPad5:        "\x1BOu",
	vaxis.KeyKeyPad6:        "\x1BOv",
	vaxis.KeyKeyPad7:        "\x1BOw",
	vaxis.KeyKeyPad8:        "\x1BOx",
	vaxis.KeyKeyPad9:        "\x1BOy",
	vaxis.KeyKeyPadDecimal:  "\x1BOn",
	vaxis.KeyKeyPadDivide:   "\x1BOo",
	vaxis.KeyKeyPadMultiply: "\x1BOj",
	vaxis.KeyKeyPadSubtract: "\x1BOm",
	vaxis.KeyKeyPadAdd:      "\x1BOk",
	vaxis.KeyKeyPadEnter:    "\x1BOM",
	vaxis.KeyKeyPadUp:       "\x1BOA",
	vaxis.KeyKeyPadDown:     "\x1BOB",
	vaxis.KeyKeyPadRight:    "\x1BOC",
	vaxis.KeyKeyPadLeft:     "\x1BOD",
	vaxis.KeyKeyPadBegin:    "\x1BOE",
	vaxis.KeyKeyPadHome:     "\x1BOH",
	vaxis.KeyKeyPadEnd:      "\x1BOF",
	vaxis.KeyKeyPadInsert:   "\x1B[2~",
	vaxis.KeyKeyPadDelete:   "\x1B[3~",
	vaxis.KeyKeyPadPageUp:   "\x1B[5~",
	vaxis.KeyKeyPadPageDown: "\x1B[6~",
}

var keymap = map[rune]string{
	vaxis.KeyF01: "\x1BOP",
	vaxis.KeyF02: "\x1BOQ",
	vaxis.KeyF03: "\x1BOR",
	vaxis.KeyF04: "\x1BOS",
	vaxis.KeyF05: "\x1B[15~",
	vaxis.KeyF06: "\x1B[17~",
	vaxis.KeyF07: "\x1B[18~",
	vaxis.KeyF08: "\x1B[19~",
	vaxis.KeyF09: "\x1B[20~",
	vaxis.KeyF10: "\x1B[21~",
	vaxis.KeyF11: "\x1B[23~",
	vaxis.KeyF12: "\x1B[24~",
	vaxis.KeyF13: "\x1B[1;2P",
	vaxis.KeyF14: "\x1B[1;2Q",
	vaxis.KeyF15: "\x1B[1;2R",
	vaxis.KeyF16: "\x1B[1;2S",
	vaxis.KeyF17: "\x1B[15;2~",
	vaxis.KeyF18: "\x1B[17;2~",
	vaxis.KeyF19: "\x1B[18;2~",
	vaxis.KeyF20: "\x1B[19;2~",
	vaxis.KeyF21: "\x1B[20;2~",
	vaxis.KeyF22: "\x1B[21;2~",
	vaxis.KeyF23: "\x1B[23;2~",
	vaxis.KeyF24: "\x1B[24;2~",
	vaxis.KeyF25: "\x1B[1;5P",
	vaxis.KeyF26: "\x1B[1;5Q",
	vaxis.KeyF27: "\x1B[1;5R",
	vaxis.KeyF28: "\x1B[1;5S",
	vaxis.KeyF29: "\x1B[15;5~",
	vaxis.KeyF30: "\x1B[17;5~",
	vaxis.KeyF31: "\x1B[18;5~",
	vaxis.KeyF32: "\x1B[19;5~",
	vaxis.KeyF33: "\x1B[20;5~",
	vaxis.KeyF34: "\x1B[21;5~",
	vaxis.KeyF35: "\x1B[23;5~",
	vaxis.KeyF36: "\x1B[24;5~",
	vaxis.KeyF37: "\x1B[1;6P",
	vaxis.KeyF38: "\x1B[1;6Q",
	vaxis.KeyF39: "\x1B[1;6R",
	vaxis.KeyF40: "\x1B[1;6S",
	// TODO add in the rest
}
