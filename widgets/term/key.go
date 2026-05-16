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
	xtermMods |= key.Modifiers & vaxis.ModSuper

	if key.Keycode == vaxis.KeyBackspace {
		return encodeLegacyBackspace(xtermMods, decbkm)
	}

	if key.Keycode == vaxis.KeyTab && xtermMods != 0 {
		switch xtermMods {
		case vaxis.ModShift:
			return "\x1B[Z"
		case vaxis.ModAlt:
			return "\x1B\t"
		case vaxis.ModShift | vaxis.ModAlt:
			return "\x1B[27;4;9~"
		default:
			if seq := modifyKeySequence(key.Keycode, xtermMods); seq != "" {
				return seq
			}
		}
	}

	if key.Keycode == vaxis.KeyEnter && xtermMods != 0 {
		switch xtermMods {
		case vaxis.ModAlt:
			return "\x1B\r"
		case vaxis.ModShift:
			return "\x1B[27;2;13~"
		case vaxis.ModShift | vaxis.ModAlt:
			return "\x1B[27;4;13~"
		default:
			if seq := modifyKeySequence(key.Keycode, xtermMods); seq != "" {
				return seq
			}
		}
	}

	if key.Keycode == vaxis.KeyEsc && xtermMods != 0 {
		switch xtermMods {
		case vaxis.ModAlt:
			return "\x1B\x1B"
		case vaxis.ModShift:
			return "\x1B[27;2;27~"
		case vaxis.ModShift | vaxis.ModAlt:
			return "\x1B[27;4;27~"
		default:
			if seq := modifyKeySequence(key.Keycode, xtermMods); seq != "" {
				return seq
			}
		}
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

		if key.Text != "" {
			return key.Text
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
		if xtermMods&vaxis.ModCtrl != 0 {
			if code, ok := controlKeyCode(key); ok {
				if xtermMods&vaxis.ModAlt != 0 {
					buf.WriteRune('\x1b')
				}
				buf.WriteRune(code)
				return buf.String()
			}
			if seq := encodeCSIu(key); seq != "" {
				return seq
			}
		}
		if xtermMods&vaxis.ModAlt != 0 && altEscPrefix {
			buf.WriteRune('\x1b')
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

func encodeLegacyBackspace(mods vaxis.ModifierMask, decbkm bool) string {
	switch mods {
	case 0:
		if decbkm {
			return "\x08"
		}
		return "\x7f"
	case vaxis.ModCtrl:
		if decbkm {
			return "\x7f"
		}
		return "\x08"
	case vaxis.ModShift,
		vaxis.ModSuper,
		vaxis.ModShift | vaxis.ModSuper:
		return "\x7f"
	case vaxis.ModAlt,
		vaxis.ModAlt | vaxis.ModShift,
		vaxis.ModAlt | vaxis.ModSuper,
		vaxis.ModAlt | vaxis.ModShift | vaxis.ModSuper:
		return "\x1b\x7f"
	case vaxis.ModCtrl | vaxis.ModShift,
		vaxis.ModCtrl | vaxis.ModSuper,
		vaxis.ModCtrl | vaxis.ModShift | vaxis.ModSuper:
		return "\x08"
	case vaxis.ModAlt | vaxis.ModCtrl,
		vaxis.ModAlt | vaxis.ModCtrl | vaxis.ModSuper,
		vaxis.ModAlt | vaxis.ModCtrl | vaxis.ModShift | vaxis.ModSuper:
		return "\x1b\x08"
	default:
		return ""
	}
}

func modifyKeySequence(keycode rune, mods vaxis.ModifierMask) string {
	if mods == 0 {
		return ""
	}
	return fmt.Sprintf("\x1B[27;%d;%d~", int(mods)+1, keycode)
}

func controlKeyCode(key vaxis.Key) (rune, bool) {
	ch := key.Keycode
	shiftedText := false
	if key.Text != "" {
		r, size := utf8.DecodeRuneInString(key.Text)
		if r != utf8.RuneError && size == len(key.Text) {
			ch = r
			shiftedText = key.Modifiers&vaxis.ModShift != 0
		}
	} else if key.Modifiers&vaxis.ModShift != 0 && key.ShiftedCode > 0 {
		ch = key.ShiftedCode
		shiftedText = true
	}
	if shiftedText && ch >= 'A' && ch <= 'Z' {
		return 0, false
	}
	if shiftedText && ch == '@' {
		return 0, false
	}
	if ch >= 'A' && ch <= 'Z' {
		ch += 'a' - 'A'
	}

	switch ch {
	case ' ':
		return 0, true
	case '/':
		return 31, true
	case '0':
		return '0', true
	case '1':
		return '1', true
	case '2':
		return 0, true
	case '3':
		return 27, true
	case '4':
		return 28, true
	case '5':
		return 29, true
	case '6':
		return 30, true
	case '7':
		return 31, true
	case '8':
		return 127, true
	case '9':
		return '9', true
	case '?':
		return 127, true
	case '@':
		return 0, true
	case '\\':
		return 28, true
	case ']':
		return 29, true
	case '^':
		return 30, true
	case '_':
		return 31, true
	case '~':
		return 30, true
	}
	switch ch {
	case 'i', 'm', '[':
		return 0, false
	}
	if ch >= 'a' && ch <= 'z' {
		return ch - 0x60, true
	}
	return 0, false
}

func encodeCSIu(key vaxis.Key) string {
	codepoint, ok := csiuCodepoint(key)
	if !ok {
		return ""
	}
	mods := key.Modifiers & (vaxis.ModShift | vaxis.ModAlt | vaxis.ModCtrl)
	if key.Modifiers&vaxis.ModShift != 0 && key.Keycode > 0 && key.Keycode != codepoint {
		if codepoint >= 'A' && codepoint <= 'Z' && key.Keycode == codepoint+'a'-'A' {
			codepoint = key.Keycode
		} else {
			mods &^= vaxis.ModShift
		}
	}
	return fmt.Sprintf("\x1B[%d;%du", codepoint, int(mods)+1)
}

func csiuCodepoint(key vaxis.Key) (rune, bool) {
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

func (vt *Model) encodeKey(key vaxis.Key) string {
	flags := uint8(0)
	if vt.EnableKittyKeyboard {
		flags = vt.activeKittyKeyboard().current()
	}
	if flags != 0 && key.EventType != vaxis.EventRelease && key.Text != "" && !kittyTextIsSingleControl(key.Text) && key.Keycode == vaxis.KeyBackspace {
		return ""
	}
	if seq := encodeKitty(key, flags); seq != "" {
		return seq
	}
	if key.EventType == vaxis.EventRelease {
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
	if codepoint == vaxis.KeyBackspace {
		return mods != 0 && mods != vaxis.ModCtrl
	}
	if codepoint >= 0x40 && codepoint <= 0x7F {
		return true
	}
	if (codepoint == vaxis.KeyTab || codepoint == vaxis.KeyEnter) && mods != 0 {
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
	if key.EventType == vaxis.EventRelease {
		if flags&kittyKeyboardFlagReportEvents == 0 {
			return ""
		}
		if flags&kittyKeyboardFlagReportAll == 0 {
			switch key.Keycode {
			case vaxis.KeyEnter, vaxis.KeyBackspace, vaxis.KeyTab:
				return ""
			}
		}
	}

	if key.EventType != vaxis.EventRelease && key.Text != "" && !kittyTextIsSingleControl(key.Text) && key.Keycode == vaxis.KeyEnter {
		return key.Text
	}

	if flags&kittyKeyboardFlagReportAll == 0 && key.EventType != vaxis.EventRelease && key.Modifiers == 0 {
		switch key.Keycode {
		case vaxis.KeyEnter:
			return "\r"
		case vaxis.KeyBackspace:
			return "\x7f"
		case vaxis.KeyTab:
			return "\t"
		}
	}

	if key.Keycode == 0 && key.Text != "" && key.EventType != vaxis.EventRelease && key.Modifiers == 0 {
		return key.Text
	}

	if flags&kittyKeyboardFlagReportAll == 0 && key.EventType != vaxis.EventRelease && kittyTextIsPlainInput(key) {
		return key.Text
	}

	if val, ok := kittyKeymap[key.Keycode]; ok {
		if val.modifier && flags&kittyKeyboardFlagReportAll == 0 {
			return ""
		}
		return encodeKittySequence(key, val.number, val.final, flags)
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

	return encodeKittySequence(key, int(key.Keycode), 'u', flags)
}

func encodeKittySequence(key vaxis.Key, code int, final rune, flags uint8) string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "\x1B[%d", code)
	if flags&kittyKeyboardFlagReportAlternates != 0 && !unicode.IsControl(key.Keycode) {
		switch {
		case key.ShiftedCode > 0 && key.BaseLayoutCode > 0:
			fmt.Fprintf(buf, ":%d:%d", key.ShiftedCode, key.BaseLayoutCode)
		case key.ShiftedCode > 0:
			fmt.Fprintf(buf, ":%d", key.ShiftedCode)
		case key.BaseLayoutCode > 0:
			fmt.Fprintf(buf, "::%d", key.BaseLayoutCode)
		}
	}

	mods := int(key.Modifiers) + 1
	emitPrior := false
	if flags&kittyKeyboardFlagReportEvents != 0 && key.EventType != vaxis.EventPress {
		fmt.Fprintf(buf, ";%d:%d", mods, int(key.EventType)+1)
		emitPrior = true
	} else if mods > 1 {
		fmt.Fprintf(buf, ";%d", mods)
		emitPrior = true
	}
	if flags&kittyKeyboardFlagAssociatedText != 0 && key.EventType != vaxis.EventRelease && key.Text != "" && !kittyModifiersPreventText(key.Modifiers) {
		count := 0
		for _, r := range key.Text {
			if unicode.IsControl(r) {
				continue
			}
			if count == 0 {
				if !emitPrior {
					buf.WriteRune(';')
				}
				buf.WriteRune(';')
			} else {
				buf.WriteRune(':')
			}
			fmt.Fprintf(buf, "%d", r)
			count++
		}
	}
	buf.WriteRune(final)
	return buf.String()
}

func kittyTextIsSingleControl(text string) bool {
	return len(text) == 1 && text[0] < 0x80 && unicode.IsControl(rune(text[0]))
}

func kittyTextIsPrintable(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func kittyTextIsPlainInput(key vaxis.Key) bool {
	if !kittyTextIsPrintable(key.Text) {
		return false
	}
	if key.Modifiers == 0 {
		return true
	}
	if key.Modifiers&^vaxis.ModShift != 0 {
		return false
	}
	return key.ShiftedCode > 0 && key.Text == string(key.ShiftedCode)
}

func kittyModifiersPreventText(mods vaxis.ModifierMask) bool {
	const prevents = vaxis.ModAlt | vaxis.ModCtrl | vaxis.ModSuper | vaxis.ModHyper | vaxis.ModMeta
	return mods&prevents != 0
}

type keycode struct {
	number int
	final  rune
}

type kittyKeycode struct {
	number   int
	final    rune
	modifier bool
}

var kittyKeymap = map[rune]kittyKeycode{
	vaxis.KeyF13:             {57376, 'u', false},
	vaxis.KeyF14:             {57377, 'u', false},
	vaxis.KeyF15:             {57378, 'u', false},
	vaxis.KeyF16:             {57379, 'u', false},
	vaxis.KeyF17:             {57380, 'u', false},
	vaxis.KeyF18:             {57381, 'u', false},
	vaxis.KeyF19:             {57382, 'u', false},
	vaxis.KeyF20:             {57383, 'u', false},
	vaxis.KeyF21:             {57384, 'u', false},
	vaxis.KeyF22:             {57385, 'u', false},
	vaxis.KeyF23:             {57386, 'u', false},
	vaxis.KeyF24:             {57387, 'u', false},
	vaxis.KeyF25:             {57388, 'u', false},
	vaxis.KeyF26:             {57389, 'u', false},
	vaxis.KeyF27:             {57390, 'u', false},
	vaxis.KeyF28:             {57391, 'u', false},
	vaxis.KeyF29:             {57392, 'u', false},
	vaxis.KeyF30:             {57393, 'u', false},
	vaxis.KeyF31:             {57394, 'u', false},
	vaxis.KeyF32:             {57395, 'u', false},
	vaxis.KeyF33:             {57396, 'u', false},
	vaxis.KeyF34:             {57397, 'u', false},
	vaxis.KeyF35:             {57398, 'u', false},
	vaxis.KeyKeyPad0:         {57399, 'u', false},
	vaxis.KeyKeyPad1:         {57400, 'u', false},
	vaxis.KeyKeyPad2:         {57401, 'u', false},
	vaxis.KeyKeyPad3:         {57402, 'u', false},
	vaxis.KeyKeyPad4:         {57403, 'u', false},
	vaxis.KeyKeyPad5:         {57404, 'u', false},
	vaxis.KeyKeyPad6:         {57405, 'u', false},
	vaxis.KeyKeyPad7:         {57406, 'u', false},
	vaxis.KeyKeyPad8:         {57407, 'u', false},
	vaxis.KeyKeyPad9:         {57408, 'u', false},
	vaxis.KeyKeyPadDecimal:   {57409, 'u', false},
	vaxis.KeyKeyPadDivide:    {57410, 'u', false},
	vaxis.KeyKeyPadMultiply:  {57411, 'u', false},
	vaxis.KeyKeyPadSubtract:  {57412, 'u', false},
	vaxis.KeyKeyPadAdd:       {57413, 'u', false},
	vaxis.KeyKeyPadEnter:     {57414, 'u', false},
	vaxis.KeyKeyPadEqual:     {57415, 'u', false},
	vaxis.KeyKeyPadSeparator: {57416, 'u', false},
	vaxis.KeyKeyPadLeft:      {57417, 'u', false},
	vaxis.KeyKeyPadRight:     {57418, 'u', false},
	vaxis.KeyKeyPadUp:        {57419, 'u', false},
	vaxis.KeyKeyPadDown:      {57420, 'u', false},
	vaxis.KeyKeyPadPageUp:    {57421, 'u', false},
	vaxis.KeyKeyPadPageDown:  {57422, 'u', false},
	vaxis.KeyKeyPadHome:      {57423, 'u', false},
	vaxis.KeyKeyPadEnd:       {57424, 'u', false},
	vaxis.KeyKeyPadInsert:    {57425, 'u', false},
	vaxis.KeyKeyPadDelete:    {57426, 'u', false},
	vaxis.KeyKeyPadBegin:     {57427, 'u', false},
	vaxis.KeyLeftShift:       {57441, 'u', true},
	vaxis.KeyRightShift:      {57447, 'u', true},
	vaxis.KeyLeftControl:     {57442, 'u', true},
	vaxis.KeyRightControl:    {57448, 'u', true},
	vaxis.KeyLeftSuper:       {57444, 'u', true},
	vaxis.KeyRightSuper:      {57450, 'u', true},
	vaxis.KeyLeftAlt:         {57443, 'u', true},
	vaxis.KeyRightAlt:        {57449, 'u', true},
}

var xtermKeymap = map[rune]keycode{
	vaxis.KeyUp:             {1, 'A'},
	vaxis.KeyDown:           {1, 'B'},
	vaxis.KeyRight:          {1, 'C'},
	vaxis.KeyLeft:           {1, 'D'},
	vaxis.KeyEnd:            {1, 'F'},
	vaxis.KeyHome:           {1, 'H'},
	vaxis.KeyInsert:         {2, '~'},
	vaxis.KeyDelete:         {3, '~'},
	vaxis.KeyPgUp:           {5, '~'},
	vaxis.KeyPgDown:         {6, '~'},
	vaxis.KeyKeyPadUp:       {1, 'A'},
	vaxis.KeyKeyPadDown:     {1, 'B'},
	vaxis.KeyKeyPadRight:    {1, 'C'},
	vaxis.KeyKeyPadLeft:     {1, 'D'},
	vaxis.KeyKeyPadBegin:    {1, 'E'},
	vaxis.KeyKeyPadHome:     {1, 'H'},
	vaxis.KeyKeyPadEnd:      {1, 'F'},
	vaxis.KeyKeyPadInsert:   {2, '~'},
	vaxis.KeyKeyPadDelete:   {3, '~'},
	vaxis.KeyKeyPadPageUp:   {5, '~'},
	vaxis.KeyKeyPadPageDown: {6, '~'},
	vaxis.KeyF01:            {1, 'P'},
	vaxis.KeyF02:            {1, 'Q'},
	vaxis.KeyF03:            {13, '~'},
	vaxis.KeyF04:            {1, 'S'},
	vaxis.KeyF05:            {15, '~'},
	vaxis.KeyF06:            {17, '~'},
	vaxis.KeyF07:            {18, '~'},
	vaxis.KeyF08:            {19, '~'},
	vaxis.KeyF09:            {20, '~'},
	vaxis.KeyF10:            {21, '~'},
	vaxis.KeyF11:            {23, '~'},
	vaxis.KeyF12:            {24, '~'},
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
