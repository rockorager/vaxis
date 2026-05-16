package term

import (
	"fmt"
	"testing"
)

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

func TestANSIModeReportInsertMode(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{4}, '$'))
	if got, want := readReply(t, r, len("\x1B[4;2$y")), "\x1B[4;2$y"; got != want {
		t.Fatalf("insert mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{4}))
	vt.update(testCSI('p', []uint32{4}, '$'))
	if got, want := readReply(t, r, len("\x1B[4;1$y")), "\x1B[4;1$y"; got != want {
		t.Fatalf("insert mode report = %q, want %q", got, want)
	}
}

func TestANSIModeReportSendReceiveDefaultsSet(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{12}, '$'))
	if got, want := readReply(t, r, len("\x1B[12;1$y")), "\x1B[12;1$y"; got != want {
		t.Fatalf("send/receive mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{12}))
	vt.update(testCSI('p', []uint32{12}, '$'))
	if got, want := readReply(t, r, len("\x1B[12;2$y")), "\x1B[12;2$y"; got != want {
		t.Fatalf("send/receive mode report = %q, want %q", got, want)
	}
}

func TestANSIModeReportUnknown(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{9999}, '$'))

	if got, want := readReply(t, r, len("\x1B[9999;0$y")), "\x1B[9999;0$y"; got != want {
		t.Fatalf("unknown ANSI mode report = %q, want %q", got, want)
	}
}

func TestModeReportRequiresSingleParameter(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', nil, '?', '$'))
	vt.update(testCSI('p', []uint32{7, 8}, '?', '$'))
	vt.update(testCSI('p', nil, '$'))
	vt.update(testCSI('p', []uint32{4, 20}, '$'))

	assertNoReply(t, r)
}

func TestModeReportAlternateScrollDefaultsSet(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{1007}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1007;1$y")), "\x1B[?1007;1$y"; got != want {
		t.Fatalf("alternate scroll mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1007}, '?'))
	vt.update(testCSI('p', []uint32{1007}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1007;2$y")), "\x1B[?1007;2$y"; got != want {
		t.Fatalf("alternate scroll mode report = %q, want %q", got, want)
	}
}

func TestModeReportMouseFormats(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	for _, mode := range []uint32{1005, 1015, 1016} {
		vt.update(testCSI('p', []uint32{mode}, '?', '$'))
		want := fmt.Sprintf("\x1B[?%d;2$y", mode)
		if got := readReply(t, r, len(want)); got != want {
			t.Fatalf("mouse format mode %d report = %q, want %q", mode, got, want)
		}

		vt.update(testCSI('h', []uint32{mode}, '?'))
		vt.update(testCSI('p', []uint32{mode}, '?', '$'))
		want = fmt.Sprintf("\x1B[?%d;1$y", mode)
		if got := readReply(t, r, len(want)); got != want {
			t.Fatalf("mouse format mode %d report = %q, want %q", mode, got, want)
		}
	}
}

func TestModeReportApplicationKeypad(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{66}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?66;2$y")), "\x1B[?66;2$y"; got != want {
		t.Fatalf("application keypad mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{66}, '?'))
	vt.update(testCSI('p', []uint32{66}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?66;1$y")), "\x1B[?66;1$y"; got != want {
		t.Fatalf("application keypad mode report = %q, want %q", got, want)
	}
}

func TestModeReportBackarrowKeyMode(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{67}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?67;2$y")), "\x1B[?67;2$y"; got != want {
		t.Fatalf("backarrow key mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{67}, '?'))
	vt.update(testCSI('p', []uint32{67}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?67;1$y")), "\x1B[?67;1$y"; got != want {
		t.Fatalf("backarrow key mode report = %q, want %q", got, want)
	}
}

func TestAlternateScrollModeSurvivesScreenSwitch(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.update(testCSI('l', []uint32{1007}, '?'))
	vt.update(testCSI('h', []uint32{1049}, '?'))
	if vt.mode.altScroll {
		t.Fatal("alternate scroll was re-enabled entering alternate screen")
	}

	vt.update(testCSI('l', []uint32{1049}, '?'))
	if vt.mode.altScroll {
		t.Fatal("alternate scroll was re-enabled leaving alternate screen")
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

func TestModeReportReverseWrap(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{45}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?45;2$y")), "\x1B[?45;2$y"; got != want {
		t.Fatalf("reverse-wrap mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{45}, '?'))
	vt.update(testCSI('p', []uint32{45}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?45;1$y")), "\x1B[?45;1$y"; got != want {
		t.Fatalf("reverse-wrap mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{45}, '?'))
	vt.update(testCSI('p', []uint32{45}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?45;2$y")), "\x1B[?45;2$y"; got != want {
		t.Fatalf("reverse-wrap mode report = %q, want %q", got, want)
	}
}

func TestReverseWrapModeSaveRestore(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.update(testCSI('h', []uint32{45}, '?'))
	vt.update(testCSI('s', []uint32{45}, '?'))
	vt.update(testCSI('l', []uint32{45}, '?'))
	vt.update(testCSI('r', []uint32{45}, '?'))

	if !vt.mode.reverseWrap {
		t.Fatal("reverse-wrap mode was not restored")
	}
}
