package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDeviceStatusReportOperatingStatus(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('n', []uint32{5}))

	if got, want := readReply(t, r, len("\x1B[0n")), "\x1B[0n"; got != want {
		t.Fatalf("device status reply = %q, want %q", got, want)
	}
}

func TestDeviceStatusReportCursorPosition(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('n', []uint32{6}))
	if got, want := readReply(t, r, len("\x1B[1;1R")), "\x1B[1;1R"; got != want {
		t.Fatalf("cursor position reply = %q, want %q", got, want)
	}

	vt.update(testCSI('H', []uint32{5, 10}))
	vt.update(testCSI('n', []uint32{6}))
	if got, want := readReply(t, r, len("\x1B[5;10R")), "\x1B[5;10R"; got != want {
		t.Fatalf("cursor position reply = %q, want %q", got, want)
	}
}

func TestDeviceStatusReportCursorPositionOriginMode(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('r', []uint32{5, 20}))
	vt.update(testCSI('h', []uint32{6}, '?'))
	vt.update(testCSI('H', []uint32{3, 5}))
	vt.update(testCSI('n', []uint32{6}))

	if got, want := readReply(t, r, len("\x1B[3;5R")), "\x1B[3;5R"; got != want {
		t.Fatalf("origin-mode cursor position reply = %q, want %q", got, want)
	}
}

func TestDeviceStatusReportColorScheme(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.Update(vaxis.ColorThemeUpdate{Mode: vaxis.DarkMode})
	vt.update(testCSI('n', []uint32{996}, '?'))

	if got, want := readReply(t, r, len("\x1B[?997;1n")), "\x1B[?997;1n"; got != want {
		t.Fatalf("color scheme reply = %q, want %q", got, want)
	}
}

func TestDeviceStatusReportColorSchemeUnknown(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('n', []uint32{996}, '?'))

	assertNoReply(t, r)
}
