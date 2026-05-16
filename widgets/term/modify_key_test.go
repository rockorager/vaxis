package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestModifyOtherKeysStateTwoEncodesModifiedText(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.Update(vaxis.Key{
		Keycode:     'h',
		ShiftedCode: 'H',
		Text:        "H",
		Modifiers:   vaxis.ModShift | vaxis.ModCtrl,
		EventType:   vaxis.EventPress,
	})

	if got, want := readReply(t, r, len("\x1B[27;6;72~")), "\x1B[27;6;72~"; got != want {
		t.Fatalf("modifyOtherKeys encoded key = %q, want %q", got, want)
	}
}

func TestModifyOtherKeysStateTwoEncodesAltDigit(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.Update(vaxis.Key{
		Keycode:   '8',
		Text:      "8",
		Modifiers: vaxis.ModAlt,
		EventType: vaxis.EventPress,
	})

	if got, want := readReply(t, r, len("\x1B[27;3;56~")), "\x1B[27;3;56~"; got != want {
		t.Fatalf("modifyOtherKeys encoded digit = %q, want %q", got, want)
	}
}

func TestModifyOtherKeysStateTwoEncodesPlainASCIIText(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.Update(vaxis.Key{Keycode: 'x', Text: "x", EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\x1B[27;1;120~")), "\x1B[27;1;120~"; got != want {
		t.Fatalf("plain ASCII key with modifyOtherKeys = %q, want %q", got, want)
	}
}

func TestModifyOtherKeysResetFallsBackToLegacyEncoding(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	vt.update(testCSI('m', nil, '>'))
	vt.Update(vaxis.Key{Keycode: 'j', Modifiers: vaxis.ModCtrl, EventType: vaxis.EventPress})

	if got, want := readReply(t, r, len("\n")), "\n"; got != want {
		t.Fatalf("legacy key after modifyOtherKeys reset = %q, want %q", got, want)
	}
}

func TestModifyOtherKeysInvalidRequestIgnored(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{4, 2}, '>'))
	if !vt.mode.modifyOtherKeys2 {
		t.Fatal("modifyOtherKeys state 2 was not enabled")
	}

	vt.update(testCSI('m', []uint32{9}, '>'))
	vt.update(testCSI('m', []uint32{4, 2, 1}, '>'))
	if !vt.mode.modifyOtherKeys2 {
		t.Fatal("invalid modifyOtherKeys request changed state")
	}

	vt.update(testCSI('n', nil, '>'))
	if vt.mode.modifyOtherKeys2 {
		t.Fatal("CSI > n did not reset modifyOtherKeys state")
	}
}
