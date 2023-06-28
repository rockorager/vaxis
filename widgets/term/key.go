package term

import (
	"bytes"
	"fmt"
	"unicode"

	"git.sr.ht/~rockorager/rtk"
)

// TODO we assume it's always application keys. Add in the right modes and
// encode properly
func encodeXterm(key rtk.Key, applicationMode bool) string {
	// function keys
	if val, ok := keymap[key.Codepoint]; ok {
		return val
	}

	// ignore any kitty mods
	xtermMods := key.Modifiers & rtk.ModShift
	xtermMods |= key.Modifiers & rtk.ModAlt
	xtermMods |= key.Modifiers & rtk.ModCtrl
	if xtermMods == 0 {
		switch applicationMode {
		case true:
			// Special keys
			if val, ok := normalKeymap[key.Codepoint]; ok {
				return val
			}
		case false:
			// Special keys
			if val, ok := applicationKeymap[key.Codepoint]; ok {
				return val
			}
		}

		if key.Codepoint < unicode.MaxRune {
			// Unicode keys
			return string(key.Codepoint)
		}
	}

	if val, ok := xtermKeymap[key.Codepoint]; ok {
		return fmt.Sprintf("\x1B[%d;%d%c", val.number, int(xtermMods)+1, val.final)
	}

	buf := bytes.NewBuffer(nil)
	if key.Codepoint < unicode.MaxRune {
		if xtermMods&rtk.ModAlt != 0 {
			buf.WriteRune('\x1b')
		}
		if xtermMods&rtk.ModCtrl != 0 {
			if unicode.IsLower(key.Codepoint) {
				buf.WriteRune(key.Codepoint - 0x60)
				return buf.String()
			}
			buf.WriteRune(key.Codepoint - 0x40)
			return buf.String()
		}
		if xtermMods&rtk.ModShift != 0 {
			if unicode.IsLower(key.Codepoint) {
				buf.WriteRune(unicode.ToUpper(key.Codepoint))
				return buf.String()
			}
		}
		buf.WriteRune(key.Codepoint)
		return buf.String()
	}
	return ""
}

type keycode struct {
	number int
	final  rune
}

var xtermKeymap = map[rune]keycode{
	rtk.KeyUp:     {1, 'A'},
	rtk.KeyDown:   {1, 'B'},
	rtk.KeyRight:  {1, 'C'},
	rtk.KeyLeft:   {1, 'D'},
	rtk.KeyEnd:    {1, 'F'},
	rtk.KeyHome:   {1, 'H'},
	rtk.KeyInsert: {2, '~'},
	rtk.KeyDelete: {3, '~'},
	rtk.KeyPgUp:   {5, '~'},
	rtk.KeyPgDown: {6, '~'},
}

var normalKeymap = map[rune]string{
	rtk.KeyUp:     "\x1B[A",
	rtk.KeyDown:   "\x1B[B",
	rtk.KeyRight:  "\x1B[C",
	rtk.KeyLeft:   "\x1B[D",
	rtk.KeyEnd:    "\x1B[F",
	rtk.KeyHome:   "\x1B[H",
	rtk.KeyInsert: "\x1B[2~",
	rtk.KeyDelete: "\x1B[3~",
	rtk.KeyPgUp:   "\x1B[5~",
	rtk.KeyPgDown: "\x1B[6~",
}

var applicationKeymap = map[rune]string{
	rtk.KeyUp:     "\x1BOA",
	rtk.KeyDown:   "\x1BOB",
	rtk.KeyRight:  "\x1BOC",
	rtk.KeyLeft:   "\x1BOD",
	rtk.KeyEnd:    "\x1BOF",
	rtk.KeyHome:   "\x1BOH",
	rtk.KeyInsert: "\x1B[2~",
	rtk.KeyDelete: "\x1B[3~",
	rtk.KeyPgUp:   "\x1B[5~",
	rtk.KeyPgDown: "\x1B[6~",
}

var keymap = map[rune]string{
	rtk.KeyF01: "\x1BOP",
	rtk.KeyF02: "\x1BOQ",
	rtk.KeyF03: "\x1BOR",
	rtk.KeyF04: "\x1BOS",
	rtk.KeyF05: "\x1B[15~",
	rtk.KeyF06: "\x1B[17~",
	rtk.KeyF07: "\x1B[18~",
	rtk.KeyF08: "\x1B[19~",
	rtk.KeyF09: "\x1B[20~",
	rtk.KeyF10: "\x1B[21~",
	rtk.KeyF11: "\x1B[23~",
	rtk.KeyF12: "\x1B[24~",
	rtk.KeyF13: "\x1B[1;2P",
	rtk.KeyF14: "\x1B[1;2Q",
	rtk.KeyF15: "\x1B[1;2R",
	rtk.KeyF16: "\x1B[1;2S",
	rtk.KeyF17: "\x1B[15;2~",
	rtk.KeyF18: "\x1B[17;2~",
	rtk.KeyF19: "\x1B[18;2~",
	rtk.KeyF20: "\x1B[19;2~",
	rtk.KeyF21: "\x1B[20;2~",
	rtk.KeyF22: "\x1B[21;2~",
	rtk.KeyF23: "\x1B[23;2~",
	rtk.KeyF24: "\x1B[24;2~",
	rtk.KeyF25: "\x1B[1;5P",
	rtk.KeyF26: "\x1B[1;5Q",
	rtk.KeyF27: "\x1B[1;5R",
	rtk.KeyF28: "\x1B[1;5S",
	rtk.KeyF29: "\x1B[15;5~",
	rtk.KeyF30: "\x1B[17;5~",
	rtk.KeyF31: "\x1B[18;5~",
	rtk.KeyF32: "\x1B[19;5~",
	rtk.KeyF33: "\x1B[20;5~",
	rtk.KeyF34: "\x1B[21;5~",
	rtk.KeyF35: "\x1B[23;5~",
	rtk.KeyF36: "\x1B[24;5~",
	rtk.KeyF37: "\x1B[1;6P",
	rtk.KeyF38: "\x1B[1;6Q",
	rtk.KeyF39: "\x1B[1;6R",
	rtk.KeyF40: "\x1B[1;6S",
	// TODO add in the rest
}
