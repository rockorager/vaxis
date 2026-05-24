package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestBackarrowKeyModeControlsBackspaceEncoding(t *testing.T) {
	vt := New()

	if got, want := vt.encodeKey(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventPress}), "\x7f"; got != want {
		t.Fatalf("backspace with DECBKM reset = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{67}, '?'))
	if got, want := vt.encodeKey(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventPress}), "\x08"; got != want {
		t.Fatalf("backspace with DECBKM set = %q, want %q", got, want)
	}
}

func TestControlBackspaceInvertsBackarrowKeyMode(t *testing.T) {
	vt := New()

	key := vaxis.Key{Keycode: vaxis.KeyBackspace, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress}
	if got, want := vt.encodeKey(key), "\x08"; got != want {
		t.Fatalf("ctrl+backspace with DECBKM reset = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{67}, '?'))
	if got, want := vt.encodeKey(key), "\x7f"; got != want {
		t.Fatalf("ctrl+backspace with DECBKM set = %q, want %q", got, want)
	}
}

func TestModifiedBackspaceMatchesGhosttyLegacyTable(t *testing.T) {
	tests := []struct {
		name   string
		mods   vaxis.ModifierMask
		decbkm bool
		want   string
	}{
		{"shift reset", vaxis.ModShift, false, "\x7f"},
		{"shift set", vaxis.ModShift, true, "\x7f"},
		{"alt reset", vaxis.ModAlt, false, "\x1b\x7f"},
		{"alt set", vaxis.ModAlt, true, "\x1b\x7f"},
		{"alt ctrl reset", vaxis.ModAlt | vaxis.ModCtrl, false, "\x1b\x08"},
		{"alt ctrl set", vaxis.ModAlt | vaxis.ModCtrl, true, "\x1b\x08"},
		{"ctrl shift reset", vaxis.ModCtrl | vaxis.ModShift, false, "\x08"},
		{"ctrl shift set", vaxis.ModCtrl | vaxis.ModShift, true, "\x08"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			if tt.decbkm {
				vt.update(testCSI('h', []uint32{67}, '?'))
			}

			key := vaxis.Key{Keycode: vaxis.KeyBackspace, Modifiers: tt.mods, EventType: vaxis.EventPress}
			if got := vt.encodeKey(key); got != tt.want {
				t.Fatalf("modified backspace = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestControlKeyEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "ctrl space",
			key:  vaxis.Key{Keycode: ' ', Text: " ", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x00",
		},
		{
			name: "ctrl slash",
			key:  vaxis.Key{Keycode: '/', Text: "/", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1F",
		},
		{
			name: "ctrl zero",
			key:  vaxis.Key{Keycode: '0', Text: "0", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "0",
		},
		{
			name: "ctrl nine",
			key:  vaxis.Key{Keycode: '9', Text: "9", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "9",
		},
		{
			name: "ctrl shift minus",
			key:  vaxis.Key{Keycode: '-', Text: "_", ShiftedCode: '_', Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1F",
		},
		{
			name: "ctrl question",
			key:  vaxis.Key{Keycode: '/', Text: "?", ShiftedCode: '?', Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x7F",
		},
		{
			name: "ctrl i",
			key:  vaxis.Key{Keycode: 'i', Text: "i", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[105;5u",
		},
		{
			name: "ctrl m",
			key:  vaxis.Key{Keycode: 'm', Text: "m", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[109;5u",
		},
		{
			name: "ctrl left bracket",
			key:  vaxis.Key{Keycode: '[', Text: "[", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[91;5u",
		},
		{
			name: "ctrl shift two",
			key:  vaxis.Key{Keycode: '2', Text: "@", ShiftedCode: '@', Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[64;5u",
		},
		{
			name: "ctrl shift letter",
			key:  vaxis.Key{Keycode: 'm', Text: "M", ShiftedCode: 'M', Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[109;6u",
		},
		{
			name: "ctrl alt c",
			key:  vaxis.Key{Keycode: 'c', Text: "c", Modifiers: vaxis.ModAlt | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B\x03",
		},
		{
			name: "ctrl alt i",
			key:  vaxis.Key{Keycode: 'i', Text: "i", Modifiers: vaxis.ModAlt | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[105;7u",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded key = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFunctionKeyEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "f3",
			key:  vaxis.Key{Keycode: vaxis.KeyF03, EventType: vaxis.EventPress},
			want: "\x1BOR",
		},
		{
			name: "ctrl f3",
			key:  vaxis.Key{Keycode: vaxis.KeyF03, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[13;5~",
		},
		{
			name: "super f1",
			key:  vaxis.Key{Keycode: vaxis.KeyF01, Modifiers: vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[1;9P",
		},
		{
			name: "ctrl super f5",
			key:  vaxis.Key{Keycode: vaxis.KeyF05, Modifiers: vaxis.ModCtrl | vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[15;13~",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded function key = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModifiedKeypadNavigationEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "shift keypad up",
			key:  vaxis.Key{Keycode: vaxis.KeyKeyPadUp, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[1;2A",
		},
		{
			name: "ctrl keypad begin",
			key:  vaxis.Key{Keycode: vaxis.KeyKeyPadBegin, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[1;5E",
		},
		{
			name: "alt keypad home",
			key:  vaxis.Key{Keycode: vaxis.KeyKeyPadHome, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B[1;3H",
		},
		{
			name: "ctrl keypad page down",
			key:  vaxis.Key{Keycode: vaxis.KeyKeyPadPageDown, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[6;5~",
		},
		{
			name: "super keypad left",
			key:  vaxis.Key{Keycode: vaxis.KeyKeyPadLeft, Modifiers: vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[1;9D",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded keypad navigation = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTabModifierEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "shift tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[Z",
		},
		{
			name: "alt tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\t",
		},
		{
			name: "ctrl tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;5;9~",
		},
		{
			name: "ctrl shift tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;6;9~",
		},
		{
			name: "super tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[27;9;9~",
		},
		{
			name: "ctrl super tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModCtrl | vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[27;13;9~",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded tab = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModifyOtherKeysStateTwoOverridesShiftTab(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.Update(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\x1B[27;2;9~")), "\x1B[27;2;9~"; got != want {
		t.Fatalf("modifyOtherKeys shift tab = %q, want %q", got, want)
	}
}

func TestEnterModifierEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "shift enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[27;2;13~",
		},
		{
			name: "alt enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\r",
		},
		{
			name: "ctrl enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;5;13~",
		},
		{
			name: "ctrl shift enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;6;13~",
		},
		{
			name: "super enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[27;9;13~",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded enter = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModifyOtherKeysStateTwoOverridesAltEnter(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\x1B[27;3;13~")), "\x1B[27;3;13~"; got != want {
		t.Fatalf("modifyOtherKeys alt enter = %q, want %q", got, want)
	}
}

func TestLinefeedModeExpandsCarriageReturnInput(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('h', []uint32{20}))
	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\r\n")), "\r\n"; got != want {
		t.Fatalf("linefeed-mode enter = %q, want %q", got, want)
	}

	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\x1B\r\n")), "\x1B\r\n"; got != want {
		t.Fatalf("linefeed-mode alt-enter = %q, want %q", got, want)
	}
}

func TestLinefeedModeResetStopsCarriageReturnExpansion(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('h', []uint32{20}))
	vt.update(testCSI('l', []uint32{20}))
	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\r")), "\r"; got != want {
		t.Fatalf("enter after linefeed-mode reset = %q, want %q", got, want)
	}
}

func TestEscapeModifierEncodingMatchesGhosttyTable(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "shift escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[27;2;27~",
		},
		{
			name: "alt escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\x1B",
		},
		{
			name: "ctrl escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;5;27~",
		},
		{
			name: "ctrl shift escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModShift | vaxis.ModCtrl, EventType: vaxis.EventPress},
			want: "\x1B[27;6;27~",
		},
		{
			name: "super escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModSuper, EventType: vaxis.EventPress},
			want: "\x1B[27;9;27~",
		},
	}

	vt := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded escape = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCursorKeyModeControlsCursorKeyEncoding(t *testing.T) {
	vt := New()
	key := vaxis.Key{Keycode: vaxis.KeyUp, EventType: vaxis.EventPress}

	if got, want := vt.encodeKey(key), "\x1B[A"; got != want {
		t.Fatalf("up key with DECCKM reset = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{1}, '?'))
	if got, want := vt.encodeKey(key), "\x1BOA"; got != want {
		t.Fatalf("up key with DECCKM set = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1}, '?'))
	if got, want := vt.encodeKey(key), "\x1B[A"; got != want {
		t.Fatalf("up key with DECCKM reset again = %q, want %q", got, want)
	}
}

func TestCursorKeyModeControlsAllCursorKeyEncodings(t *testing.T) {
	tests := []struct {
		name        string
		keycode     rune
		normal      string
		application string
	}{
		{"up", vaxis.KeyUp, "\x1B[A", "\x1BOA"},
		{"down", vaxis.KeyDown, "\x1B[B", "\x1BOB"},
		{"right", vaxis.KeyRight, "\x1B[C", "\x1BOC"},
		{"left", vaxis.KeyLeft, "\x1B[D", "\x1BOD"},
		{"home", vaxis.KeyHome, "\x1B[H", "\x1BOH"},
		{"end", vaxis.KeyEnd, "\x1B[F", "\x1BOF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			key := vaxis.Key{Keycode: tt.keycode, EventType: vaxis.EventPress}

			if got := vt.encodeKey(key); got != tt.normal {
				t.Fatalf("normal cursor key = %q, want %q", got, tt.normal)
			}

			vt.update(testCSI('h', []uint32{1}, '?'))
			if got := vt.encodeKey(key); got != tt.application {
				t.Fatalf("application cursor key = %q, want %q", got, tt.application)
			}
		})
	}
}

func TestApplicationKeypadModeCanBeSetByDECMode(t *testing.T) {
	vt := New()

	key := vaxis.Key{Keycode: vaxis.KeyKeyPad1, Text: "1", EventType: vaxis.EventPress}
	if got, want := vt.encodeKey(key), "1"; got != want {
		t.Fatalf("keypad 1 with DECKPAM reset = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{66}, '?'))
	if got, want := vt.encodeKey(key), "1"; got != want {
		t.Fatalf("keypad 1 with DECKPAM set and ignore-keypad mode = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1035}, '?'))
	if got, want := vt.encodeKey(key), "\x1BOq"; got != want {
		t.Fatalf("keypad 1 with DECKPAM set and ignore-keypad reset = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{66}, '?'))
	if got, want := vt.encodeKey(key), "1"; got != want {
		t.Fatalf("keypad 1 with DECKPAM reset again = %q, want %q", got, want)
	}
}

func TestAltEscPrefixModeControlsAltTextEncoding(t *testing.T) {
	vt := New()
	key := vaxis.Key{Keycode: 'x', Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress}

	if got, want := vt.encodeKey(key), "\x1Bx"; got != want {
		t.Fatalf("alt text with prefix mode set = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1036}, '?'))
	if got, want := vt.encodeKey(key), "x"; got != want {
		t.Fatalf("alt text with prefix mode reset = %q, want %q", got, want)
	}
}

func TestAltEscPrefixModeDoesNotDisableAltSpecialKeys(t *testing.T) {
	vt := New()
	vt.update(testCSI('l', []uint32{1036}, '?'))

	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "alt tab",
			key:  vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\t",
		},
		{
			name: "alt enter",
			key:  vaxis.Key{Keycode: vaxis.KeyEnter, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\r",
		},
		{
			name: "alt escape",
			key:  vaxis.Key{Keycode: vaxis.KeyEsc, Modifiers: vaxis.ModAlt, EventType: vaxis.EventPress},
			want: "\x1B\x1B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vt.encodeKey(tt.key); got != tt.want {
				t.Fatalf("encoded special key = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextInputPreferredOverKeycode(t *testing.T) {
	vt := New()

	key := vaxis.Key{Keycode: 'a', Text: "abcd", EventType: vaxis.EventPress}
	if got, want := vt.encodeKey(key), "abcd"; got != want {
		t.Fatalf("text input with keycode = %q, want %q", got, want)
	}
}

func TestRepeatTextInputEncodesLikePress(t *testing.T) {
	vt := New()

	key := vaxis.Key{Keycode: 'a', Text: "a", EventType: vaxis.EventRepeat}
	if got, want := vt.encodeKey(key), "a"; got != want {
		t.Fatalf("repeat text input = %q, want %q", got, want)
	}
}

func TestKeyModesSaveRestore(t *testing.T) {
	vt := New()

	vt.update(testCSI('h', []uint32{1, 66, 67}, '?'))
	vt.update(testCSI('s', []uint32{1, 66, 67}, '?'))
	vt.update(testCSI('l', []uint32{1, 66, 67}, '?'))
	vt.update(testCSI('r', []uint32{1, 66, 67}, '?'))

	if !vt.mode.decckm {
		t.Fatal("cursor key mode was not restored")
	}
	if !vt.mode.deckpam {
		t.Fatal("application keypad mode was not restored")
	}
	if !vt.mode.decbkm {
		t.Fatal("backarrow key mode was not restored")
	}
}
