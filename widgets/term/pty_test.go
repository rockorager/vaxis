package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestInputWithoutPtyDoesNotPanic(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.mode.paste = true
	vt.mode.colorScheme = true

	withoutPanic(t, func() {
		vt.Update(vaxis.Key{Keycode: 'x', EventType: vaxis.EventPress})
		vt.Update(vaxis.PasteStartEvent{})
		vt.Update(vaxis.PasteEndEvent{})
		vt.Update(vaxis.ColorThemeUpdate{Mode: vaxis.DarkMode})
	})
}

func TestCloseWithoutPtyDoesNotPanic(t *testing.T) {
	vt := New()

	withoutPanic(t, func() {
		vt.Close()
		vt.Close()
	})
}

func TestKeyboardActionModeDoesNotSuppressInputByDefault(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.update(testCSI('h', []uint32{2}))
	vt.Update(vaxis.Key{Keycode: 'x', Text: "x", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("x")), "x"; got != want {
		t.Fatalf("KAM default input = %q, want %q", got, want)
	}
}

func TestKeyboardActionModeSuppressesInputWhenAllowed(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKeyboardActionMode(true))
	vt.mode.paste = true

	vt.update(testCSI('h', []uint32{2}))
	vt.Update(vaxis.Key{Keycode: 'x', Text: "x", EventType: vaxis.EventPress})
	vt.Update(vaxis.PasteStartEvent{})
	vt.Update(vaxis.PasteEndEvent{})

	assertNoReply(t, r)
}

func TestKeyboardActionModeResetAllowsInput(t *testing.T) {
	vt, r := newReplyTestModel(t, WithKeyboardActionMode(true))

	vt.update(testCSI('h', []uint32{2}))
	vt.update(testCSI('l', []uint32{2}))
	vt.Update(vaxis.Key{Keycode: 'x', Text: "x", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("x")), "x"; got != want {
		t.Fatalf("input after KAM reset = %q, want %q", got, want)
	}
}

func TestFocusWithoutPtyDoesNotPanic(t *testing.T) {
	vt := New()
	vt.mode.focusEvents = true

	withoutPanic(t, func() {
		vt.Focus()
		vt.Blur()
		vt.Update(vaxis.FocusIn{})
		vt.Update(vaxis.FocusOut{})
	})
}

func TestEnableFocusEventsReportsCurrentFocusedState(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.Focus()

	vt.update(testCSI('h', []uint32{1004}, '?'))

	if got, want := readReply(t, r, len("\x1B[I")), "\x1B[I"; got != want {
		t.Fatalf("focus report on enable = %q, want %q", got, want)
	}
}

func TestEnableFocusEventsReportsCurrentBlurredState(t *testing.T) {
	vt, r := newReplyTestModel(t)

	vt.update(testCSI('h', []uint32{1004}, '?'))

	if got, want := readReply(t, r, len("\x1B[O")), "\x1B[O"; got != want {
		t.Fatalf("blur report on enable = %q, want %q", got, want)
	}
}

func TestFocusEventsReportThroughUpdate(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.update(testCSI('h', []uint32{1004}, '?'))
	if got, want := readReply(t, r, len("\x1B[O")), "\x1B[O"; got != want {
		t.Fatalf("initial focus report = %q, want %q", got, want)
	}

	vt.Update(vaxis.FocusIn{})
	if got, want := readReply(t, r, len("\x1B[I")), "\x1B[I"; got != want {
		t.Fatalf("focus-in report = %q, want %q", got, want)
	}

	vt.Update(vaxis.FocusOut{})
	if got, want := readReply(t, r, len("\x1B[O")), "\x1B[O"; got != want {
		t.Fatalf("focus-out report = %q, want %q", got, want)
	}
}

func TestAltScrollWithoutPtyDoesNotPanic(t *testing.T) {
	vt := New()
	vt.mode.smcup = true
	vt.mode.altScroll = true

	withoutPanic(t, func() {
		vt.handleMouse(vaxis.Mouse{
			Button:    vaxis.MouseWheelUp,
			EventType: vaxis.EventPress,
		})
	})
}

func withoutPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}
