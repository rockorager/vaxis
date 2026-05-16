package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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

func TestFocusWithoutPtyDoesNotPanic(t *testing.T) {
	vt := New()
	vt.mode.focusEvents = true

	withoutPanic(t, func() {
		vt.Focus()
		vt.Blur()
	})
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
