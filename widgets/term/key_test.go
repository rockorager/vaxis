package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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

func TestKeyModesSaveRestore(t *testing.T) {
	vt := New()

	vt.update(testCSI('h', []uint32{66, 67}, '?'))
	vt.update(testCSI('s', []uint32{66, 67}, '?'))
	vt.update(testCSI('l', []uint32{66, 67}, '?'))
	vt.update(testCSI('r', []uint32{66, 67}, '?'))

	if !vt.mode.deckpam {
		t.Fatal("application keypad mode was not restored")
	}
	if !vt.mode.decbkm {
		t.Fatal("backarrow key mode was not restored")
	}
}
