package vaxis

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func testCSI(final rune, params []int, colonAfter ...int) ansi.CSI {
	seq := ansi.CSI{
		Final:         final,
		NumParameters: len(params),
	}
	if len(params) <= ansi.InlineCSIParams {
		for i, param := range params {
			seq.Parameters[i] = uint32(param)
		}
	} else {
		seq.ExtraParameters = make([]uint32, len(params))
		for i, param := range params {
			seq.ExtraParameters[i] = uint32(param)
		}
	}
	for _, idx := range colonAfter {
		seq.ColonSeparators |= 1 << uint(idx)
	}
	return seq
}

func TestKey(t *testing.T) {
	tests := []struct {
		name      string
		key       Key
		matchRune rune
		matchMods ModifierMask
	}{
		{
			name:      "j",
			key:       Key{Keycode: 'j'},
			matchRune: 'j',
		},
		{
			name:      "Ctrl+@",
			key:       Key{Keycode: '@', Modifiers: ModCtrl},
			matchRune: '@',
			matchMods: ModCtrl,
		},
		{
			name:      "Ctrl+a",
			key:       Key{Keycode: 'a', Modifiers: ModCtrl},
			matchRune: 'a',
			matchMods: ModCtrl,
		},
		{
			name:      "Alt+a",
			key:       Key{Keycode: 'a', Modifiers: ModAlt},
			matchRune: 'a',
			matchMods: ModAlt,
		},
		{
			name:      "F1",
			key:       Key{Keycode: KeyF01},
			matchRune: KeyF01,
		},
		{
			name:      "Shift+F1",
			key:       Key{Keycode: KeyF01, Modifiers: ModShift},
			matchRune: KeyF01,
			matchMods: ModShift,
		},
		{
			name:      "Shift+Tab",
			key:       Key{Keycode: KeyTab, Modifiers: ModShift},
			matchRune: KeyTab,
			matchMods: ModShift,
		},
		{
			name:      "Escape",
			key:       Key{Keycode: KeyEsc},
			matchRune: KeyEsc,
		},
		{
			name:      "space",
			key:       Key{Keycode: KeySpace},
			matchRune: ' ',
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.key.String(); got != test.name {
				t.Fatalf("String() = %q, want %q", got, test.name)
			}
			if !test.key.Matches(test.matchRune, test.matchMods) {
				t.Fatalf("Matches(%q, %v) = false, want true", test.matchRune, test.matchMods)
			}
		})
	}
}

// Parses a raw sequence obtained from actual terminals into a key then tests
// the match function
func TestKeyMatches(t *testing.T) {
	shouldMatch := []struct {
		name        string
		sequence    string
		matchRune   rune
		matchMods   ModifierMask
		matchString string
	}{
		{
			name:        "Application: j",
			sequence:    "j",
			matchRune:   'j',
			matchString: "j",
		},
		{
			name:        "Kitty: j",
			sequence:    "\x1b[106;1:3u",
			matchRune:   'j',
			matchString: "j",
		},
		{
			name:        "Legacy: Ctrl+j",
			sequence:    "\n",
			matchRune:   'j',
			matchMods:   ModCtrl,
			matchString: "ctrl+j",
		},
		{
			name:        "Legacy: Ctrl+z",
			sequence:    "\x1a",
			matchRune:   'z',
			matchMods:   ModCtrl,
			matchString: "ctrl+z",
		},
		{
			name:      "Kitty: Ctrl+j",
			sequence:  "\x1b[106;5:3u",
			matchRune: 'j',
			matchMods: ModCtrl,
		},
		{
			name:        "Legacy: caps+j",
			sequence:    "J",
			matchRune:   'J',
			matchString: "caps+J",
		},
		{
			name:        "Kitty: caps+j",
			sequence:    "\x1b[106;65;74u",
			matchRune:   'J',
			matchString: "caps+j",
		},
		{
			name:        "Kitty: shift+j",
			sequence:    "\x1b[106;65;74u",
			matchRune:   'j',
			matchMods:   ModShift,
			matchString: "shift+j",
		},
		{
			name:        "Legacy: F1",
			sequence:    "\x1bOP",
			matchRune:   KeyF01,
			matchString: "f1",
		},
		{
			name:        "Kitty: F1",
			sequence:    "\x1b[P",
			matchRune:   KeyF01,
			matchString: "f1",
		},
		{
			name:        "Legacy: Shift+F1",
			sequence:    "\x1b[1;2P",
			matchRune:   KeyF01,
			matchMods:   ModShift,
			matchString: "shift+f1",
		},
		{
			name:        "Kitty: Shift+F1",
			sequence:    "\x1b[1;2P",
			matchRune:   KeyF01,
			matchMods:   ModShift,
			matchString: "shift+f1",
		},
		{
			name:        "Kitty: F35",
			sequence:    "\x1b[57398u",
			matchRune:   KeyF35,
			matchString: "F35",
		},
		{
			name:        "Kitty: Shift+F35",
			sequence:    "\x1b[57398;2u",
			matchRune:   KeyF35,
			matchMods:   ModShift,
			matchString: "sHiFt+f35",
		},
		{
			name:        "Legacy: ф",
			sequence:    "ф",
			matchRune:   'ф',
			matchString: "ф",
		},
		{
			name:        "Kitty: ф matched to 'ф'",
			sequence:    "\x1b[1092::97;;1092u",
			matchRune:   'ф',
			matchString: "ф",
		},
		{
			name:        "Kitty: ф matched to 'a'",
			sequence:    "\x1b[1092::97;;1092u",
			matchRune:   'a',
			matchString: "ф",
		},
		{
			name:        "Kitty: Ctrl+Shift+ф matched to Ctrl+Shift+'a'",
			sequence:    "\x1b[1092:1060:97;6:3u",
			matchRune:   'a',
			matchMods:   ModCtrl | ModShift,
			matchString: "ctrl+shift+ф",
		},
		{
			name:        "Kitty: ':' (shift + ';')",
			sequence:    "\x1b[59:58;2;58u",
			matchRune:   ':',
			matchMods:   ModShift,
			matchString: ":",
		},
		{
			name:        "legacy: 'tab'",
			sequence:    "\t",
			matchRune:   KeyTab,
			matchString: "tab",
		},
		{
			name:        "legacy: 'shift+tab'",
			sequence:    "\x1b[Z",
			matchRune:   KeyTab,
			matchMods:   ModShift,
			matchString: "shift+tab",
		},
		{
			name:        "Kitty: 'tab'",
			sequence:    "\x1b[9;1:1u",
			matchRune:   KeyTab,
			matchString: "tab",
		},
		{
			name:        "Kitty: 'shift+tab'",
			sequence:    "\x1b[9;2:1u",
			matchRune:   KeyTab,
			matchMods:   ModShift,
			matchString: "shift+tab",
		},
		{
			name:        "legacy: 'ctrl+shift+tab'",
			sequence:    "\x1b[27;6;9~",
			matchRune:   KeyTab,
			matchMods:   ModShift | ModCtrl,
			matchString: "ctrl+shift+tab",
		},
		{
			name:        "kitty: 'caps+p'",
			sequence:    "\x1b[112;65;80u", // actually the sequence for CAPS+p
			matchRune:   'p',
			matchMods:   ModCapsLock,
			matchString: "P",
		},
	}

	for _, test := range shouldMatch {
		t.Run(test.name, func(t *testing.T) {
			parser := ansi.NewParser(strings.NewReader(test.sequence))
			seq := <-parser.Next()
			key := decodeKey(seq)
			if !key.Matches(test.matchRune, test.matchMods) {
				t.Fatalf("got %s %#v", key.String(), key)
			}
			if test.matchString != "" {
				if !key.MatchString(test.matchString) {
					t.Fatalf("got %s %#v", key.String(), key)
				}
			}
		})
	}

	shouldNotMatch := []struct {
		name        string
		sequence    string
		matchRune   rune
		matchMods   ModifierMask
		matchString string
	}{
		{
			name:        "kitty: 'caps+p' is not 'ctrl+shift+p'",
			sequence:    "\x1b[112;65;80u", // actually the sequence for CAPS+p
			matchRune:   'p',
			matchMods:   ModCtrl | ModShift,
			matchString: "ctrl+shift+p",
		},
	}
	for _, test := range shouldNotMatch {
		t.Run(test.name, func(t *testing.T) {
			parser := ansi.NewParser(strings.NewReader(test.sequence))
			seq := <-parser.Next()
			key := decodeKey(seq)
			if key.Matches(test.matchRune, test.matchMods) {
				t.Fatalf("got %s %#v", key.String(), key)
			}
			if test.matchString != "" {
				if key.MatchString(test.matchString) {
					t.Fatalf("got %s %#v", key.String(), key)
				}
			}
		})
	}
}

func TestKeyDecode(t *testing.T) {
	tests := []struct {
		name     string
		sequence ansi.Sequence
		expected Key
	}{
		{
			name:     "legacy: j",
			sequence: ansi.Print{Grapheme: "j", Width: 1},
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name:     "legacy: Up",
			sequence: ansi.SS3('A'),
			expected: Key{Keycode: KeyUp},
		},
		{
			name:     "legacy: Up, normal keys",
			sequence: ansi.CSI{Final: 'A'},
			expected: Key{Keycode: KeyUp},
		},
		{
			name:     "legacy: shift+j",
			sequence: ansi.Print{Grapheme: "J", Width: 1},
			expected: Key{
				Keycode:     'j',
				ShiftedCode: 'J',
				Modifiers:   ModShift,
				Text:        "J",
			},
		},
		{
			name:     "kitty: j with event",
			sequence: testCSI('u', []int{106, 1, 1, 106}, 1),
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name:     "kitty: j with minimal",
			sequence: testCSI('u', []int{106, 0, 106}),
			expected: Key{
				Keycode: 'j',
				Text:    "j",
			},
		},
		{
			name:     "kitty: ф",
			sequence: testCSI('u', []int{1092, 0, 102, 0, 1092}, 0, 1),
			expected: Key{
				Keycode:        'ф',
				BaseLayoutCode: 'f',
				Text:           "ф",
			},
		},
		{
			name:     "kitty: multiple codepoints",
			sequence: testCSI('u', []int{106, 0, 127482, 127480}, 2),
			expected: Key{
				Keycode: 'j',
				Text:    "🇺🇸",
			},
		},
		{
			name:     "kitty: keypad begin",
			sequence: testCSI('u', []int{57427}),
			expected: Key{Keycode: KeyKeyPadBegin},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := decodeKey(test.sequence); got != test.expected {
				t.Fatalf("decodeKey() = %#v, want %#v", got, test.expected)
			}
		})
	}
}
