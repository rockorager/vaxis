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

func TestDeviceStatusReportRequiresOneParameter(t *testing.T) {
	tests := []struct {
		name         string
		intermediate []rune
		params       []uint32
		theme        vaxis.ColorThemeMode
	}{
		{name: "public none", params: nil},
		{name: "public multiple", params: []uint32{5, 6}},
		{name: "private none", intermediate: []rune{'?'}, params: nil, theme: vaxis.DarkMode},
		{name: "private multiple", intermediate: []rune{'?'}, params: []uint32{996, 1}, theme: vaxis.DarkMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)
			if tt.theme != 0 {
				vt.Update(vaxis.ColorThemeUpdate{Mode: tt.theme})
			}

			vt.update(testCSI('n', tt.params, tt.intermediate...))

			assertNoReply(t, r)
		})
	}
}

func TestDeviceStatusReportIgnoresUnknownRequests(t *testing.T) {
	tests := []struct {
		name         string
		intermediate []rune
		params       []uint32
		theme        vaxis.ColorThemeMode
	}{
		{name: "public unknown", params: []uint32{999}},
		{name: "private cursor position", intermediate: []rune{'?'}, params: []uint32{6}},
		{name: "public color scheme", params: []uint32{996}, theme: vaxis.DarkMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)
			if tt.theme != 0 {
				vt.Update(vaxis.ColorThemeUpdate{Mode: tt.theme})
			}

			vt.update(testCSI('n', tt.params, tt.intermediate...))

			assertNoReply(t, r)
		})
	}
}
