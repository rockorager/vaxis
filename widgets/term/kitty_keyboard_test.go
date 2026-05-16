package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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
