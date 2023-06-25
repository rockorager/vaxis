package term

import (
	"fmt"
	"unicode"

	"git.sr.ht/~rockorager/rtk"
)

func normalKeyCode(msg rtk.Key) string {
	if msg.Modifiers == 0 {
		kc, ok := normalKeymap[msg.Codepoint]
		if ok {
			return kc
		}
	}
	return keyCode(msg)
}

func applicationKeyCode(msg rtk.Key) string {
	if msg.Modifiers == 0 {
		kc, ok := applicationKeymap[msg.Codepoint]
		if ok {
			return kc
		}
	}
	return keyCode(msg)
}

func keyCode(msg rtk.Key) string {
	if msg.Codepoint < unicode.MaxRune {
		switch msg.Modifiers {
		case 0:
			return string(msg.Codepoint)
		case rtk.ModAlt:
			return fmt.Sprintf("\x1B%c", msg.Codepoint)
		case rtk.ModCtrl:
			if unicode.IsLower(msg.Codepoint) {
				return fmt.Sprintf("%c", msg.Codepoint-0x60)
			}
			return fmt.Sprintf("%c", msg.Codepoint-0x40)
		}
	}
	if kc, ok := xtermKeymap[msg.Codepoint]; ok {
		if msg.Modifiers == 0 {
			return fmt.Sprintf("\x1B[%d%c", kc.number, kc.final)
		}
		return fmt.Sprintf("\x1B[%d;%d%c", kc.number, int(msg.Modifiers)+1, kc.final)
	}
	if kc, ok := keymap[msg.Codepoint]; ok {
		return kc
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
	rtk.KeyF01:    {1, 'P'},
	rtk.KeyF02:    {1, 'Q'},
	rtk.KeyF03:    {1, 'R'},
	rtk.KeyF04:    {1, 'S'},
	rtk.KeyF05:    {15, '~'},
	rtk.KeyF06:    {17, '~'},
	rtk.KeyF07:    {18, '~'},
	rtk.KeyF08:    {19, '~'},
	rtk.KeyF09:    {20, '~'},
	rtk.KeyF10:    {21, '~'},
	rtk.KeyF11:    {23, '~'},
	rtk.KeyF12:    {24, '~'},
}

var normalKeymap = map[rune]string{
	rtk.KeyUp:    "\x1B[A",
	rtk.KeyDown:  "\x1B[B",
	rtk.KeyRight: "\x1B[C",
	rtk.KeyLeft:  "\x1B[D",
	rtk.KeyEnd:   "\x1B[F",
	rtk.KeyHome:  "\x1B[H",
}

var applicationKeymap = map[rune]string{
	rtk.KeyUp:    "\x1BOA",
	rtk.KeyDown:  "\x1BOB",
	rtk.KeyRight: "\x1BOC",
	rtk.KeyLeft:  "\x1BOD",
	rtk.KeyEnd:   "\x1BOF",
	rtk.KeyHome:  "\x1BOH",
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
}
