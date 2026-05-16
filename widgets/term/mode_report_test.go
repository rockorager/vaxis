package term

import "testing"

func TestModeReportWraparound(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{7}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?7;1$y")), "\x1B[?7;1$y"; got != want {
		t.Fatalf("wraparound mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{7}, '?'))
	vt.update(testCSI('p', []uint32{7}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?7;2$y")), "\x1B[?7;2$y"; got != want {
		t.Fatalf("wraparound mode report = %q, want %q", got, want)
	}
}

func TestModeReportUnknown(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{9999}, '?', '$'))

	if got, want := readReply(t, r, len("\x1B[?9999;0$y")), "\x1B[?9999;0$y"; got != want {
		t.Fatalf("unknown mode report = %q, want %q", got, want)
	}
}

func TestModeReportSaveCursor(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{1048}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1048;2$y")), "\x1B[?1048;2$y"; got != want {
		t.Fatalf("save-cursor mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{1048}, '?'))
	vt.update(testCSI('p', []uint32{1048}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1048;1$y")), "\x1B[?1048;1$y"; got != want {
		t.Fatalf("save-cursor mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1048}, '?'))
	vt.update(testCSI('p', []uint32{1048}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1048;2$y")), "\x1B[?1048;2$y"; got != want {
		t.Fatalf("save-cursor mode report = %q, want %q", got, want)
	}
}
