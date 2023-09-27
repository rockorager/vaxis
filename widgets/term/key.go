package term

import (
	"bytes"
	"fmt"
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
)

// TODO we assume it's always application keys. Add in the right modes and
// encode properly
func encodeXterm(key vaxis.Key, deckpam bool, decckm bool) string {
	// function keys
	if val, ok := keymap[key.Keycode]; ok {
		return val
	}

	// ignore any kitty mods
	xtermMods := key.Modifiers & vaxis.ModShift
	xtermMods |= key.Modifiers & vaxis.ModAlt
	xtermMods |= key.Modifiers & vaxis.ModCtrl
	if xtermMods == 0 {
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

	buf := bytes.NewBuffer(nil)
	if key.Keycode < unicode.MaxRune {
		if xtermMods&vaxis.ModAlt != 0 {
			buf.WriteRune('\x1b')
		}
		if xtermMods&vaxis.ModCtrl != 0 {
			if unicode.IsLower(key.Keycode) {
				buf.WriteRune(key.Keycode - 0x60)
				return buf.String()
			}
			buf.WriteRune(key.Keycode - 0x40)
			return buf.String()
		}
		if xtermMods&vaxis.ModShift != 0 {
			if unicode.IsLower(key.Keycode) {
				buf.WriteRune(unicode.ToUpper(key.Keycode))
				return buf.String()
			}
		}
		buf.WriteRune(key.Keycode)
		return buf.String()
	}
	return ""
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
	vaxis.KeyInsert: "\x1B[2~",
	vaxis.KeyDelete: "\x1B[3~",
	vaxis.KeyPgUp:   "\x1B[5~",
	vaxis.KeyPgDown: "\x1B[6~",
}

var applicationKeymap = map[rune]string{
	vaxis.KeyInsert: "\x1B[2~",
	vaxis.KeyDelete: "\x1B[3~",
	vaxis.KeyPgUp:   "\x1B[5~",
	vaxis.KeyPgDown: "\x1B[6~",
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
