package term

import (
	"fmt"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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

func TestModeReportDECANMUnknownLikeGhostty(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('h', []uint32{2}, '?'))
	vt.update(testCSI('p', []uint32{2}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?2;0$y")), "\x1B[?2;0$y"; got != want {
		t.Fatalf("DECANM mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('s', []uint32{2}, '?'))
	vt.update(testCSI('l', []uint32{2}, '?'))
	vt.update(testCSI('r', []uint32{2}, '?'))
	vt.update(testCSI('p', []uint32{2}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?2;0$y")), "\x1B[?2;0$y"; got != want {
		t.Fatalf("restored DECANM mode report = %q, want %q", got, want)
	}
}

func TestModeReportRecognizedDECModeDefaults(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	tests := []struct {
		mode  uint32
		state int
	}{
		{1, 2},
		{3, 2},
		{4, 2},
		{5, 2},
		{6, 2},
		{7, 1},
		{8, 2},
		{9, 2},
		{12, 2},
		{25, 1},
		{40, 2},
		{45, 2},
		{47, 2},
		{66, 2},
		{67, 2},
		{69, 2},
		{80, 2},
		{1000, 2},
		{1002, 2},
		{1003, 2},
		{1004, 2},
		{1005, 2},
		{1006, 2},
		{1007, 1},
		{1015, 2},
		{1016, 2},
		{1035, 1},
		{1036, 1},
		{1039, 2},
		{1045, 2},
		{1047, 2},
		{1048, 2},
		{1049, 2},
		{2004, 2},
		{2026, 2},
		{2027, 2},
		{2031, 2},
		{2048, 2},
		{8452, 2},
	}

	for _, tt := range tests {
		vt.update(testCSI('p', []uint32{tt.mode}, '?', '$'))
		want := fmt.Sprintf("\x1B[?%d;%d$y", tt.mode, tt.state)
		if got := readReply(t, r, len(want)); got != want {
			t.Fatalf("default DEC mode %d report = %q, want %q", tt.mode, got, want)
		}
	}
}

func TestModeReportCursorBlinkingFollowsDECSCUSR(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{12}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?12;2$y")), "\x1B[?12;2$y"; got != want {
		t.Fatalf("initial cursor blinking mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('q', []uint32{5}, ' '))
	vt.update(testCSI('p', []uint32{12}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?12;1$y")), "\x1B[?12;1$y"; got != want {
		t.Fatalf("blinking cursor style mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('q', []uint32{6}, ' '))
	vt.update(testCSI('p', []uint32{12}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?12;2$y")), "\x1B[?12;2$y"; got != want {
		t.Fatalf("steady cursor style mode report = %q, want %q", got, want)
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

func TestANSIModeReportKeyboardActionMode(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{2}, '$'))
	if got, want := readReply(t, r, len("\x1B[2;2$y")), "\x1B[2;2$y"; got != want {
		t.Fatalf("keyboard action mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{2}))
	vt.update(testCSI('p', []uint32{2}, '$'))
	if got, want := readReply(t, r, len("\x1B[2;1$y")), "\x1B[2;1$y"; got != want {
		t.Fatalf("keyboard action mode report = %q, want %q", got, want)
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

func TestANSIModeReportLinefeedMode(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{20}, '$'))
	if got, want := readReply(t, r, len("\x1B[20;2$y")), "\x1B[20;2$y"; got != want {
		t.Fatalf("linefeed mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('h', []uint32{20}))
	vt.update(testCSI('p', []uint32{20}, '$'))
	if got, want := readReply(t, r, len("\x1B[20;1$y")), "\x1B[20;1$y"; got != want {
		t.Fatalf("linefeed mode report = %q, want %q", got, want)
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

func TestModeReportKeyEncodingDefaults(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('p', []uint32{1035}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1035;1$y")), "\x1B[?1035;1$y"; got != want {
		t.Fatalf("ignore-keypad mode report = %q, want %q", got, want)
	}
	vt.update(testCSI('p', []uint32{1036}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1036;1$y")), "\x1B[?1036;1$y"; got != want {
		t.Fatalf("alt-esc-prefix mode report = %q, want %q", got, want)
	}

	vt.update(testCSI('l', []uint32{1035, 1036}, '?'))
	vt.update(testCSI('p', []uint32{1035}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1035;2$y")), "\x1B[?1035;2$y"; got != want {
		t.Fatalf("ignore-keypad mode report = %q, want %q", got, want)
	}
	vt.update(testCSI('p', []uint32{1036}, '?', '$'))
	if got, want := readReply(t, r, len("\x1B[?1036;2$y")), "\x1B[?1036;2$y"; got != want {
		t.Fatalf("alt-esc-prefix mode report = %q, want %q", got, want)
	}
}

func TestModeReportRecognizedNoOpModes(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	for _, mode := range []uint32{5, 12, 40, 1039, 2026, 2027, 2031, 2048} {
		vt.update(testCSI('p', []uint32{mode}, '?', '$'))
		want := fmt.Sprintf("\x1B[?%d;2$y", mode)
		if got := readReply(t, r, len(want)); got != want {
			t.Fatalf("recognized no-op mode %d report = %q, want %q", mode, got, want)
		}

		vt.update(testCSI('h', []uint32{mode}, '?'))
		vt.update(testCSI('p', []uint32{mode}, '?', '$'))
		want = fmt.Sprintf("\x1B[?%d;1$y", mode)
		if got := readReply(t, r, len(want)); got != want {
			t.Fatalf("recognized no-op mode %d report = %q, want %q", mode, got, want)
		}
	}
}

func TestReverseVideoModeAppliesAtRenderTime(t *testing.T) {
	vt := New()
	cell := vaxis.Cell{Character: vaxis.Character{Grapheme: "A", Width: 1}}

	if got := vt.renderCell(cell); got.Attribute&vaxis.AttrReverse != 0 {
		t.Fatalf("rendered cell attribute = %v, want no reverse", got.Attribute)
	}

	vt.update(testCSI('h', []uint32{5}, '?'))
	rendered := vt.renderCell(cell)
	if rendered.Attribute&vaxis.AttrReverse == 0 {
		t.Fatalf("rendered cell attribute = %v, want reverse", rendered.Attribute)
	}
	if cell.Attribute&vaxis.AttrReverse != 0 {
		t.Fatal("reverse video mutated stored cell")
	}
}

func TestReverseVideoModeComposesWithSGRReverse(t *testing.T) {
	vt := New()
	cell := vaxis.Cell{
		Character: vaxis.Character{Grapheme: "A", Width: 1},
		Style: vaxis.Style{
			Attribute: vaxis.AttrReverse,
		},
	}

	vt.update(testCSI('h', []uint32{5}, '?'))
	rendered := vt.renderCell(cell)
	if rendered.Attribute&vaxis.AttrReverse != 0 {
		t.Fatalf("rendered cell attribute = %v, want screen reverse to cancel SGR reverse", rendered.Attribute)
	}
}

func TestModeReportAltScreenAndReportingModesSetReset(t *testing.T) {
	tests := []uint32{47, 1047, 1048, 1049, 1006, 2031, 2048}

	for _, mode := range tests {
		t.Run(fmt.Sprintf("%d", mode), func(t *testing.T) {
			vt, r := newReplyTestModel(t)
			vt.resize(80, 24)

			vt.update(testCSI('h', []uint32{mode}, '?'))
			vt.update(testCSI('p', []uint32{mode}, '?', '$'))
			want := fmt.Sprintf("\x1B[?%d;1$y", mode)
			if got := readReply(t, r, len(want)); got != want {
				t.Fatalf("set mode %d report = %q, want %q", mode, got, want)
			}

			vt.update(testCSI('l', []uint32{mode}, '?'))
			vt.update(testCSI('p', []uint32{mode}, '?', '$'))
			want = fmt.Sprintf("\x1B[?%d;2$y", mode)
			if got := readReply(t, r, len(want)); got != want {
				t.Fatalf("reset mode %d report = %q, want %q", mode, got, want)
			}
		})
	}
}

func TestModeReportMouseFormats(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	for _, mode := range []uint32{1005, 1006, 1015, 1016} {
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

func TestInBandSizeReportOnEnable(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})

	vt.update(testCSI('h', []uint32{2048}, '?'))

	if got, want := readReply(t, r, len("\x1B[48;24;80;432;720t")), "\x1B[48;24;80;432;720t"; got != want {
		t.Fatalf("in-band size report = %q, want %q", got, want)
	}
}

func TestInBandSizeReportDoesNotRequirePixelSize(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24})

	vt.update(testCSI('h', []uint32{2048}, '?'))

	if got, want := readReply(t, r, len("\x1B[48;24;80;0;0t")), "\x1B[48;24;80;0;0t"; got != want {
		t.Fatalf("in-band size report with zero pixels = %q, want %q", got, want)
	}
}

func TestInBandSizeReportOnResizeUpdate(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})
	vt.update(testCSI('h', []uint32{2048}, '?'))
	if got, want := readReply(t, r, len("\x1B[48;24;80;432;720t")), "\x1B[48;24;80;432;720t"; got != want {
		t.Fatalf("initial in-band size report = %q, want %q", got, want)
	}

	vt.Update(vaxis.Resize{Cols: 100, Rows: 30, XPixel: 1000, YPixel: 600})

	if got, want := readReply(t, r, len("\x1B[48;30;100;600;1000t")), "\x1B[48;30;100;600;1000t"; got != want {
		t.Fatalf("resize in-band size report = %q, want %q", got, want)
	}
}

func TestInBandSizeReportEmitsZeroPixels(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24})

	vt.update(testCSI('h', []uint32{2048}, '?'))

	if got, want := readReply(t, r, len("\x1B[48;24;80;0;0t")), "\x1B[48;24;80;0;0t"; got != want {
		t.Fatalf("in-band size report with zero pixels = %q, want %q", got, want)
	}

	vt.Update(vaxis.Resize{Cols: 100, Rows: 30})

	if got, want := readReply(t, r, len("\x1B[48;30;100;0;0t")), "\x1B[48;30;100;0;0t"; got != want {
		t.Fatalf("resize in-band size report with zero pixels = %q, want %q", got, want)
	}
}

func TestInBandSizeReportOnCellOnlyResizeUsesKnownCellPixels(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)
	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})
	vt.update(testCSI('h', []uint32{2048}, '?'))
	if got, want := readReply(t, r, len("\x1B[48;24;80;432;720t")), "\x1B[48;24;80;432;720t"; got != want {
		t.Fatalf("initial in-band size report = %q, want %q", got, want)
	}

	vt.Resize(100, 30)

	if got, want := readReply(t, r, len("\x1B[48;30;100;540;900t")), "\x1B[48;30;100;540;900t"; got != want {
		t.Fatalf("resize in-band size report = %q, want %q", got, want)
	}
}

func TestInBandSizeReportDisabledOnResizeUpdate(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.Update(vaxis.Resize{Cols: 80, Rows: 24, XPixel: 720, YPixel: 432})

	assertNoReply(t, r)
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
