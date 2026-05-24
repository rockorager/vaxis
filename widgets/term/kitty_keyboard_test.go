package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestKittyKeyboardQueryDisabledByDefault(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('u', nil, '?'))

	assertNoReply(t, r)
}

func TestWithVaxisDisablesKittyKeyboardWithoutHostSupport(t *testing.T) {
	vx := &vaxis.Vaxis{}
	vt, r := newReplyTestModel(t, WithVaxis(vx))
	vt.resize(80, 24)

	if vt.vx != vx {
		t.Fatal("WithVaxis did not attach host Vaxis")
	}
	if vt.EnableKittyKeyboard {
		t.Fatal("WithVaxis enabled Kitty keyboard without host support")
	}

	vt.update(testCSI('u', nil, '?'))

	assertNoReply(t, r)
}

func TestKittyKeyboardControlsDefaultTERM(t *testing.T) {
	if got, want := New().defaultTERM(), "xterm-256color"; got != want {
		t.Fatalf("default TERM without Kitty keyboard = %q, want %q", got, want)
	}
	if got, want := New(WithKittyKeyboard(true)).defaultTERM(), "xterm-kitty"; got != want {
		t.Fatalf("default TERM with Kitty keyboard = %q, want %q", got, want)
	}
	if got, want := New(WithVaxis(&vaxis.Vaxis{})).defaultTERM(), "xterm-256color"; got != want {
		t.Fatalf("default TERM without host Kitty keyboard support = %q, want %q", got, want)
	}
}

func TestKittyKeyboardQueryReportsCurrentFlags(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?0u")), "\x1B[?0u"; got != want {
		t.Fatalf("default Kitty keyboard flags = %q, want %q", got, want)
	}

	vt.update(testCSI('u', []uint32{5}, '>'))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?5u")), "\x1B[?5u"; got != want {
		t.Fatalf("pushed Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardSetOrNotAndPop(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '='))
	vt.update(testCSI('u', []uint32{4, 2}, '='))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?5u")), "\x1B[?5u"; got != want {
		t.Fatalf("or-ed Kitty keyboard flags = %q, want %q", got, want)
	}

	vt.update(testCSI('u', []uint32{1, 3}, '='))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?4u")), "\x1B[?4u"; got != want {
		t.Fatalf("masked Kitty keyboard flags = %q, want %q", got, want)
	}

	vt.update(testCSI('u', []uint32{7}, '>'))
	vt.update(testCSI('u', []uint32{1}, '<'))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?4u")), "\x1B[?4u"; got != want {
		t.Fatalf("popped Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardPopNoParamsDefaultsToOne(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.update(testCSI('u', []uint32{2}, '>'))
	vt.update(testCSI('u', nil, '<'))
	vt.update(testCSI('u', nil, '?'))

	if got, want := readReply(t, r, len("\x1B[?1u")), "\x1B[?1u"; got != want {
		t.Fatalf("popped Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardPopLargeCountResetsStack(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.update(testCSI('u', []uint32{2}, '>'))
	vt.update(testCSI('u', []uint32{100}, '<'))
	vt.update(testCSI('u', nil, '?'))

	if got, want := readReply(t, r, len("\x1B[?0u")), "\x1B[?0u"; got != want {
		t.Fatalf("large pop Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardPushOverflowUsesFixedRing(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	for i := uint32(1); i <= kittyKeyboardStackLen+1; i++ {
		vt.update(testCSI('u', []uint32{i}, '>'))
	}
	vt.update(testCSI('u', nil, '?'))

	if got, want := readReply(t, r, len("\x1B[?9u")), "\x1B[?9u"; got != want {
		t.Fatalf("overflowed Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardStateIsPerScreen(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.update(testCSI('h', []uint32{1049}, '?'))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?0u")), "\x1B[?0u"; got != want {
		t.Fatalf("alternate screen Kitty keyboard flags = %q, want %q", got, want)
	}

	vt.update(testCSI('u', []uint32{2}, '>'))
	vt.update(testCSI('l', []uint32{1049}, '?'))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?1u")), "\x1B[?1u"; got != want {
		t.Fatalf("primary screen Kitty keyboard flags = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{1049}, '?'))
	vt.update(testCSI('u', nil, '?'))
	if got, want := readReply(t, r, len("\x1B[?2u")), "\x1B[?2u"; got != want {
		t.Fatalf("restored alternate screen Kitty keyboard flags = %q, want %q", got, want)
	}
}

func TestRISClearsAlternateScreenKittyKeyboardState(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('h', []uint32{1049}, '?'))
	vt.update(testCSI('u', []uint32{15}, '>'))
	vt.update(testCSI('l', []uint32{1049}, '?'))

	vt.update(testESC('c'))
	vt.update(testCSI('h', []uint32{1049}, '?'))
	vt.update(testCSI('u', nil, '?'))

	if got, want := readReply(t, r, len("\x1B[?0u")), "\x1B[?0u"; got != want {
		t.Fatalf("alternate screen Kitty keyboard flags after RIS = %q, want %q", got, want)
	}
}

func TestKittyKeyboardInvalidFlagsIgnored(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{3}, '>'))
	vt.update(testCSI('u', []uint32{32}, '>'))
	vt.update(testCSI('u', []uint32{32}, '='))
	vt.update(testCSI('u', nil, '?'))

	if got, want := readReply(t, r, len("\x1B[?3u")), "\x1B[?3u"; got != want {
		t.Fatalf("Kitty keyboard flags after invalid requests = %q, want %q", got, want)
	}
}

func TestKittyKeyboardInputFallsBackUntilChildEnablesIt(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.Update(vaxis.Key{Keycode: 'j', Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\n")), "\n"; got != want {
		t.Fatalf("input before Kitty keyboard negotiation = %q, want %q", got, want)
	}

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.Update(vaxis.Key{Keycode: 'j', Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\x1B[106;5u")), "\x1B[106;5u"; got != want {
		t.Fatalf("input after Kitty keyboard negotiation = %q, want %q", got, want)
	}
}

func TestKittyKeyboardInputIgnoredWhenHostSupportDisabled(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.Update(vaxis.Key{Keycode: 'j', Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\n")), "\n"; got != want {
		t.Fatalf("input with disabled host Kitty keyboard support = %q, want %q", got, want)
	}
}

func TestKittyKeyboardInputReportsReleaseEventsWhenEnabled(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{3}, '>'))
	vt.Update(vaxis.Key{Keycode: 'x', EventType: vaxis.EventRelease})

	if got, want := readReply(t, r, len("\x1B[120;1:3u")), "\x1B[120;1:3u"; got != want {
		t.Fatalf("Kitty keyboard release event = %q, want %q", got, want)
	}
}

func TestKittyKeyboardPlainTextUsesTextInput(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.Update(vaxis.Key{Keycode: 'a', Text: "abcd", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("abcd")), "abcd"; got != want {
		t.Fatalf("Kitty keyboard plain text = %q, want %q", got, want)
	}
}

func TestKittyKeyboardRepeatPlainTextUsesTextInput(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.Update(vaxis.Key{Keycode: 'a', Text: "a", EventType: vaxis.EventRepeat})

	if got, want := readReply(t, r, len("a")), "a"; got != want {
		t.Fatalf("Kitty keyboard repeat text = %q, want %q", got, want)
	}
}

func TestKittyKeyboardComposedTextWithReportAllUsesTextInput(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{31}, '>'))
	vt.Update(vaxis.Key{Text: "û", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("û")), "û"; got != want {
		t.Fatalf("Kitty keyboard composed text = %q, want %q", got, want)
	}
}

func TestKittyKeyboardEnterWithAllFlagsOmitsDefaultFields(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{31}, '>'))
	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\x1B[13u")), "\x1B[13u"; got != want {
		t.Fatalf("Kitty keyboard enter with all flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardAssociatedTextFormatting(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{25}, '>'))
	vt.Update(vaxis.Key{Keycode: 'j', Text: "j\n", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\x1B[106;;106u")), "\x1B[106;;106u"; got != want {
		t.Fatalf("Kitty keyboard associated text = %q, want %q", got, want)
	}
}

func TestKittyKeyboardAssociatedTextSuppressedByModifiersAndRelease(t *testing.T) {
	t.Run("ctrl", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{25}, '>'))
		vt.Update(vaxis.Key{Keycode: 'j', Text: "j", Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress})

		if got, want := readReply(t, r, len("\x1B[106;5u")), "\x1B[106;5u"; got != want {
			t.Fatalf("Kitty keyboard associated text with ctrl = %q, want %q", got, want)
		}
	})

	t.Run("release", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{27}, '>'))
		vt.Update(vaxis.Key{Keycode: 'j', Text: "j", Modifiers: vaxis.ModShift, EventType: vaxis.EventRelease})

		if got, want := readReply(t, r, len("\x1B[106;2:3u")), "\x1B[106;2:3u"; got != want {
			t.Fatalf("Kitty keyboard associated text on release = %q, want %q", got, want)
		}
	})
}

func TestKittyKeyboardReportAlternates(t *testing.T) {
	tests := []struct {
		name string
		key  vaxis.Key
		want string
	}{
		{
			name: "shifted printable",
			key:  vaxis.Key{Keycode: 'j', Text: "J", ShiftedCode: 'J', Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[106:74;2;74u",
		},
		{
			name: "shifted semicolon",
			key:  vaxis.Key{Keycode: ';', Text: ":", ShiftedCode: ':', Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want: "\x1B[59:58;2;58u",
		},
		{
			name: "caps lock printable",
			key:  vaxis.Key{Keycode: 'j', Text: "J", Modifiers: vaxis.ModCapsLock, EventType: vaxis.EventPress},
			want: "\x1B[106;65;74u",
		},
		{
			name: "base layout",
			key:  vaxis.Key{Keycode: 1095, Text: "ч", BaseLayoutCode: ';', EventType: vaxis.EventPress},
			want: "\x1B[1095::59;;1095u",
		},
		{
			name: "control key omits alternates",
			key:  vaxis.Key{Keycode: vaxis.KeyBackspace, ShiftedCode: 'X', BaseLayoutCode: 'Y', EventType: vaxis.EventPress},
			want: "\x1B[127u",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
			vt.resize(80, 24)

			vt.update(testCSI('u', []uint32{29}, '>'))
			vt.Update(tt.key)

			if got := readReply(t, r, len(tt.want)); got != tt.want {
				t.Fatalf("Kitty keyboard alternates %s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestKittyKeyboardShiftSemicolonWithNeovimFlags(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{3}, '>'))
	vt.Update(vaxis.Key{Keycode: ';', Text: ":", ShiftedCode: ':', Modifiers: vaxis.ModShift, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len(":")), ":"; got != want {
		t.Fatalf("Kitty keyboard shift+semicolon with Neovim flags = %q, want %q", got, want)
	}
}

func TestKittyKeyboardFunctionalKeysUseKittyCodes(t *testing.T) {
	tests := []struct {
		name  string
		flags uint32
		key   vaxis.Key
		want  string
	}{
		{
			name:  "keypad number report all with associated text",
			flags: 31,
			key:   vaxis.Key{Keycode: vaxis.KeyKeyPad1, Text: "1", EventType: vaxis.EventPress},
			want:  "\x1B[57400;;49u",
		},
		{
			name:  "keypad number without report all stays text",
			flags: 1,
			key:   vaxis.Key{Keycode: vaxis.KeyKeyPad1, Text: "1", EventType: vaxis.EventPress},
			want:  "1",
		},
		{
			name:  "keypad navigation uses private code",
			flags: 1,
			key:   vaxis.Key{Keycode: vaxis.KeyKeyPadUp, EventType: vaxis.EventPress},
			want:  "\x1B[57419u",
		},
		{
			name:  "f13 report all uses private code",
			flags: 9,
			key:   vaxis.Key{Keycode: vaxis.KeyF13, EventType: vaxis.EventPress},
			want:  "\x1B[57376u",
		},
		{
			name:  "f35 report all uses private code",
			flags: 9,
			key:   vaxis.Key{Keycode: vaxis.KeyF35, EventType: vaxis.EventPress},
			want:  "\x1B[57398u",
		},
		{
			name:  "modifier suppressed without report all",
			flags: 1,
			key:   vaxis.Key{Keycode: vaxis.KeyLeftShift, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want:  "",
		},
		{
			name:  "modifier report all",
			flags: 9,
			key:   vaxis.Key{Keycode: vaxis.KeyLeftShift, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress},
			want:  "\x1B[57441;2u",
		},
		{
			name:  "modifier release",
			flags: 11,
			key:   vaxis.Key{Keycode: vaxis.KeyLeftControl, Modifiers: vaxis.ModCtrl, EventType: vaxis.EventRelease},
			want:  "\x1B[57442;5:3u",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
			vt.resize(80, 24)

			vt.update(testCSI('u', []uint32{tt.flags}, '>'))
			vt.Update(tt.key)

			if tt.want == "" {
				assertNoReply(t, r)
				return
			}
			if got := readReply(t, r, len(tt.want)); got != tt.want {
				t.Fatalf("Kitty keyboard functional key %s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestKittyKeyboardDeadKeyTextForEnterAndBackspace(t *testing.T) {
	t.Run("enter", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{31}, '>'))
		vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, Text: "A", EventType: vaxis.EventPress})

		if got, want := readReply(t, r, len("A")), "A"; got != want {
			t.Fatalf("Kitty keyboard enter with dead-key text = %q, want %q", got, want)
		}
	})

	t.Run("backspace", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{31}, '>'))
		vt.Update(vaxis.Key{Keycode: vaxis.KeyBackspace, Text: "A", EventType: vaxis.EventPress})

		assertNoReply(t, r)
	})
}

func TestKittyKeyboardPlainEnterTabBackspaceStayLegacyWithoutReportAll(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
	vt.resize(80, 24)

	vt.update(testCSI('u', []uint32{1}, '>'))
	vt.update(testCSI('h', []uint32{67}, '?'))

	vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\r")), "\r"; got != want {
		t.Fatalf("Kitty keyboard enter = %q, want %q", got, want)
	}

	vt.Update(vaxis.Key{Keycode: vaxis.KeyTab, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\t")), "\t"; got != want {
		t.Fatalf("Kitty keyboard tab = %q, want %q", got, want)
	}

	vt.Update(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventPress})
	if got, want := readReply(t, r, len("\x7f")), "\x7f"; got != want {
		t.Fatalf("Kitty keyboard backspace = %q, want %q", got, want)
	}
}

func TestKittyKeyboardShiftEnterTabBackspaceEmitCSIu(t *testing.T) {
	tests := []struct {
		name string
		key  rune
		want string
	}{
		{"enter", vaxis.KeyEnter, "\x1B[13;2u"},
		{"tab", vaxis.KeyTab, "\x1B[9;2u"},
		{"backspace", vaxis.KeyBackspace, "\x1B[127;2u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
			vt.resize(80, 24)

			vt.update(testCSI('u', []uint32{1}, '>'))
			vt.Update(vaxis.Key{Keycode: tt.key, Modifiers: vaxis.ModShift, EventType: vaxis.EventPress})

			if got := readReply(t, r, len(tt.want)); got != tt.want {
				t.Fatalf("Kitty keyboard shift+%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestKittyKeyboardBackspaceIgnoresDECBKMWithReportAll(t *testing.T) {
	tests := []struct {
		name   string
		decbkm bool
	}{
		{"reset", false},
		{"set", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
			vt.resize(80, 24)

			vt.update(testCSI('u', []uint32{31}, '>'))
			if tt.decbkm {
				vt.update(testCSI('h', []uint32{67}, '?'))
			}
			vt.Update(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventPress})

			if got, want := readReply(t, r, len("\x1B[127u")), "\x1B[127u"; got != want {
				t.Fatalf("Kitty keyboard backspace with DECBKM %s = %q, want %q", tt.name, got, want)
			}
		})
	}
}

func TestKittyKeyboardEnterTabBackspaceReleaseRequiresReportAll(t *testing.T) {
	t.Run("suppressed", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{3}, '>'))
		vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
		vt.Update(vaxis.Key{Keycode: vaxis.KeyTab, EventType: vaxis.EventRelease})
		vt.Update(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventRelease})
		assertNoReply(t, r)
	})

	t.Run("report all", func(t *testing.T) {
		vt, r := newReplyTestModel(t, WithKittyKeyboard(true))
		vt.resize(80, 24)

		vt.update(testCSI('u', []uint32{11}, '>'))
		vt.Update(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
		if got, want := readReply(t, r, len("\x1B[13;1:3u")), "\x1B[13;1:3u"; got != want {
			t.Fatalf("Kitty keyboard enter release = %q, want %q", got, want)
		}

		vt.Update(vaxis.Key{Keycode: vaxis.KeyTab, EventType: vaxis.EventRelease})
		if got, want := readReply(t, r, len("\x1B[9;1:3u")), "\x1B[9;1:3u"; got != want {
			t.Fatalf("Kitty keyboard tab release = %q, want %q", got, want)
		}

		vt.Update(vaxis.Key{Keycode: vaxis.KeyBackspace, EventType: vaxis.EventRelease})
		if got, want := readReply(t, r, len("\x1B[127;1:3u")), "\x1B[127;1:3u"; got != want {
			t.Fatalf("Kitty keyboard backspace release = %q, want %q", got, want)
		}
	})
}
