package term

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func testDECRQSS(data string) ansi.DCS {
	return ansi.DCS{
		Intermediate:    [ansi.MaxIntermediate]rune{'$'},
		NumIntermediate: 1,
		Final:           'q',
		Data:            []rune(data),
	}
}

type recordingImage struct {
	destroyed int
}

func (img *recordingImage) Draw(vaxis.Window) {
}

func (img *recordingImage) Destroy() {
	img.destroyed += 1
}

func (img *recordingImage) Resize(int, int) {
}

func (img *recordingImage) CellSize() (int, int) {
	return 0, 0
}

func TestUnknownDCSIgnored(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(ansi.DCS{Final: 'x', Data: []rune("ignored")})

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
	assertNoReply(t, r)
}

func TestXTGETTCAPRepliesToKnownCapabilities(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Intermediate:    [ansi.MaxIntermediate]rune{'+'},
		NumIntermediate: 1,
		Final:           'q',
		Data:            []rune("536d756c78;524742;5463;4D73;5365"),
	})

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}

	want := "\x1BP1+r536D756C78=5C455B343A25703125646D\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP Smulx reply = %q, want %q", got, want)
	}
	want = "\x1BP1+r524742=38\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP RGB reply = %q, want %q", got, want)
	}
	want = "\x1BP1+r5463\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP Tc reply = %q, want %q", got, want)
	}
	want = "\x1BP1+r4D73=5C455D35323B25703125733B25703225735C303037\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP Ms reply = %q, want %q", got, want)
	}
	want = "\x1BP1+r5365=5C455B302071\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP Se reply = %q, want %q", got, want)
	}
}

func TestXTGETTCAPIgnoresUnknownCapabilities(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Intermediate:    [ansi.MaxIntermediate]rune{'+'},
		NumIntermediate: 1,
		Final:           'q',
		Data:            []rune("77686F"),
	})

	assertNoReply(t, r)
}

func TestXTGETTCAPSkipsInvalidKeysAndContinues(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Intermediate:    [ansi.MaxIntermediate]rune{'+'},
		NumIntermediate: 1,
		Final:           'q',
		Data:            []rune("who;536d756C78"),
	})

	want := "\x1BP1+r536D756C78=5C455B343A25703125646D\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("XTGETTCAP Smulx reply = %q, want %q", got, want)
	}
	assertNoReply(t, r)
}

func TestDCSQWithIntermediateDoesNotDecodeSixel(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Intermediate:    [ansi.MaxIntermediate]rune{'+'},
		NumIntermediate: 1,
		Final:           'q',
		Data:            []rune("#0;2;100;0;0#0@"),
	})

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
	assertNoReply(t, r)
}

func TestDECRQSSDollarQDoesNotDecodeSixel(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testDECRQSS("#0;2;100;0;0#0@"))

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
	assertNoReply(t, r)
}

func TestDECRQSSReportsSGR(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{1, 2, 3, 4, 5, 7, 8, 9, 38, 2, 100, 200, 255, 48, 2, 101, 102, 103}))
	vt.update(testDECRQSS("m"))

	want := "\x1BP1$r0;1;2;3;4;5;7;8;9;38:2::100:200:255;48:2::101:102:103m\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS SGR reply = %q, want %q", got, want)
	}
}

func TestDECRQSSReportsSGRIndexedColors(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testCSI('m', []uint32{32, 102}))
	vt.update(testDECRQSS("m"))

	want := "\x1BP1$r0;32;102m\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS indexed SGR reply = %q, want %q", got, want)
	}

	vt.update(testCSI('m', []uint32{0, 38, 5, 200, 48, 5, 201}))
	vt.update(testDECRQSS("m"))

	want = "\x1BP1$r0;38:5:200;48:5:201m\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS 256-color SGR reply = %q, want %q", got, want)
	}
}

func TestDECRQSSReportsMarginsAndCursorStyle(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(10, 5)

	vt.update(testCSI('r', []uint32{2, 4}))
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{3, 8}))
	vt.cursor.style = vaxis.CursorBeam
	vt.mode.cursorBlinking = true

	vt.update(testDECRQSS("r"))
	want := "\x1BP1$r2;4r\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS DECSTBM reply = %q, want %q", got, want)
	}

	vt.update(testDECRQSS("s"))
	want = "\x1BP1$r3;8s\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS DECSLRM reply = %q, want %q", got, want)
	}

	vt.update(testDECRQSS(" q"))
	want = "\x1BP1$r5 q\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS DECSCUSR reply = %q, want %q", got, want)
	}
}

func TestDECRQSSReportsDefaultCursorStyleAsSteadyBlock(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(10, 5)

	vt.update(testDECRQSS(" q"))

	want := "\x1BP1$r2 q\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS default DECSCUSR reply = %q, want %q", got, want)
	}
}

func TestDECRQSSReportsFailureForDECSLRMWhenModeDisabled(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(10, 5)

	vt.update(testCSI('s', []uint32{3, 8}))
	vt.update(testDECRQSS("s"))

	want := "\x1BP0$r\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS disabled DECSLRM reply = %q, want %q", got, want)
	}
}

func TestDECRQSSReportsFailureForUnsupportedSetting(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testDECRQSS("z"))

	want := "\x1BP0$r\x1B\\"
	if got := readReply(t, r, len(want)); got != want {
		t.Fatalf("DECRQSS failure reply = %q, want %q", got, want)
	}
}

func TestDECRQSSIgnoresInvalidLongRequest(t *testing.T) {
	vt, r := newReplyTestModel(t)
	vt.resize(80, 24)

	vt.update(testDECRQSS("\" q"))

	assertNoReply(t, r)
}

func TestSixelDCSStillSupported(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	if got := vt.graphics[0].img.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("sixel bounds = %v, want 1x1", got)
	}
}

func TestParameterizedSixelDCSStillSupported(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Parameters:    [ansi.InlineCSIParams]uint32{0, 0, 8},
		NumParameters: 3,
		Final:         'q',
		Data:          []rune("#0;2;100;0;0#0@"),
	})

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	if got := vt.graphics[0].img.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("sixel bounds = %v, want 1x1", got)
	}
}

func TestParsedSixelDCSStillSupported(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	parser := ansi.NewParser(strings.NewReader("\x1bP0;0;8q#0;2;100;0;0#0@\x1b\\"), ansi.ParserModeOutput)
	vt.parser = parser
	for {
		seq := <-parser.Next()
		if _, ok := seq.(ansi.EOF); ok {
			break
		}
		vt.update(seq)
	}

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	if got := vt.graphics[0].img.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("sixel bounds = %v, want 1x1", got)
	}
}

func TestSixelMovesCursorToLastImageRow(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@-@"),
	})

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	img := vt.graphics[0]
	if img.origin.row != 1 || img.origin.col != 2 {
		t.Fatalf("sixel origin = %d,%d, want 1,2", img.origin.row, img.origin.col)
	}
	if img.rows != 2 || img.cols != 1 {
		t.Fatalf("sixel cells = %dx%d, want 1x2", img.cols, img.rows)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 2 {
		t.Fatalf("cursor = %d,%d, want 2,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestSixelAtBottomScrollsAndStaysOnLastImageRow(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@-@"),
	})

	if got, want := vt.primaryScreen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback len = %d, want %d", got, want)
	}
	img := vt.graphics[0]
	if img.origin.row != 1 || img.origin.col != 1 {
		t.Fatalf("sixel origin = %d,%d, want 1,1", img.origin.row, img.origin.col)
	}
	if got, want := img.sourceRow, 2; got != want {
		t.Fatalf("sixel source row = %d, want %d", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 1 {
		t.Fatalf("cursor = %d,%d, want 2,1", vt.cursor.row, vt.cursor.col)
	}
}

func TestSixelDisplayModePlacesAtOriginAndLeavesCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 2
	vt.cursor.col = 3
	vt.setDECMode(80, true)

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@-@"),
	})

	img := vt.graphics[0]
	if img.origin.row != 0 || img.origin.col != 0 {
		t.Fatalf("sixel origin = %d,%d, want 0,0", img.origin.row, img.origin.col)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 3 {
		t.Fatalf("cursor = %d,%d, want 2,3", vt.cursor.row, vt.cursor.col)
	}
}

func TestSixelCursorRightMode(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 1
	vt.cursor.col = 1
	vt.setDECMode(8452, true)

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0~~"),
	})

	if vt.cursor.row != 1 || vt.cursor.col != 3 {
		t.Fatalf("cursor = %d,%d, want 1,3", vt.cursor.row, vt.cursor.col)
	}
}

func TestSixelScrollsWithText(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@-@"),
	})
	vt.ind()

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	img := vt.graphics[0]
	if img.sourceRow != 1 || img.origin.col != 1 {
		t.Fatalf("sixel source/origin = %d,%d, want 1,1", img.sourceRow, img.origin.col)
	}
	visible := vt.visibleGraphics()
	if len(visible) != 1 {
		t.Fatalf("visible graphics len = %d, want 1", len(visible))
	}
	if visible[0].row != 0 || visible[0].col != 1 {
		t.Fatalf("visible sixel = %d,%d, want 0,1", visible[0].row, visible[0].col)
	}
}

func TestSixelHiddenWhenScrolledOffscreenThenVisibleInScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@-@"),
	})
	vt.ind()
	vt.ind()

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	if visible := vt.visibleGraphics(); len(visible) != 0 {
		t.Fatalf("visible graphics len = %d, want 0", len(visible))
	}

	vt.scrollOffset = 1
	visible := vt.visibleGraphics()
	if len(visible) != 1 {
		t.Fatalf("visible scrollback graphics len = %d, want 1", len(visible))
	}
	if visible[0].row != 0 || visible[0].col != 1 {
		t.Fatalf("visible scrollback sixel = %d,%d, want 0,1", visible[0].row, visible[0].col)
	}
}

func TestSixelReflowsWithWrappedText(t *testing.T) {
	vt := New()
	vt.resize(4, 3)
	vt.size = vaxis.Resize{Cols: 4, Rows: 3, XPixel: 4, YPixel: 18}
	printText(vt, "abcd")
	vt.cr()
	vt.lf()
	printText(vt, "efgh")
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})

	vt.resize(2, 3)

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len = %d, want 1", len(vt.graphics))
	}
	img := vt.graphics[0]
	if img.sourceRow != 3 || img.origin.col != 0 {
		t.Fatalf("sixel source/origin = %d,%d, want 3,0", img.sourceRow, img.origin.col)
	}
}

func TestSixelDroppedWhenReflowMovesBeyondRightEdge(t *testing.T) {
	vt := New()
	vt.resize(4, 3)
	vt.size = vaxis.Resize{Cols: 4, Rows: 3, XPixel: 4, YPixel: 18}
	vt.cursor.row = 1
	vt.cursor.col = 3

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0~~"),
	})

	vt.resize(2, 3)

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
}

func TestSixelDroppedOnNoReflowWidthChange(t *testing.T) {
	vt := New()
	vt.resize(4, 3)
	vt.size = vaxis.Resize{Cols: 4, Rows: 3, XPixel: 4, YPixel: 18}
	vt.mode.decawm = false
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})

	vt.resize(2, 3)

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
}

func TestSixelSourceRowsShiftWhenSameWidthResizeTrimsScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.primaryScreen.state.scrollbackLimit = 0
	vt.size = vaxis.Resize{Cols: 5, Rows: 3, XPixel: 5, YPixel: 18}
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len before resize = %d, want 1", len(vt.graphics))
	}
	if got, want := vt.graphics[0].sourceRow, 2; got != want {
		t.Fatalf("sixel source row before resize = %d, want %d", got, want)
	}

	vt.resize(5, 1)

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len after resize = %d, want 1", len(vt.graphics))
	}
	img := vt.graphics[0]
	if img.sourceRow != 0 || img.origin.col != 1 {
		t.Fatalf("sixel source/origin = %d,%d, want 0,1", img.sourceRow, img.origin.col)
	}
	visible := vt.visibleGraphics()
	if len(visible) != 1 {
		t.Fatalf("visible graphics len = %d, want 1", len(visible))
	}
	if visible[0].row != 0 || visible[0].col != 1 {
		t.Fatalf("visible sixel = %d,%d, want 0,1", visible[0].row, visible[0].col)
	}
}

func TestSixelSourceRowsShiftWhenReflowResizeTrimsScrollback(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.primaryScreen.state.scrollbackLimit = 0
	vt.size = vaxis.Resize{Cols: 4, Rows: 2, XPixel: 4, YPixel: 12}
	setScreenLine(vt.primaryScreen, 0, "abcd")
	setScreenLine(vt.primaryScreen, 1, "x")
	vt.cursor.row = 1
	vt.cursor.col = 0

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len before resize = %d, want 1", len(vt.graphics))
	}
	if got, want := vt.graphics[0].sourceRow, 1; got != want {
		t.Fatalf("sixel source row before resize = %d, want %d", got, want)
	}

	vt.resize(2, 2)

	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len after resize = %d, want 1", len(vt.graphics))
	}
	img := vt.graphics[0]
	if img.sourceRow != 1 || img.origin.col != 0 {
		t.Fatalf("sixel source/origin = %d,%d, want 1,0", img.sourceRow, img.origin.col)
	}
	visible := vt.visibleGraphics()
	if len(visible) != 1 {
		t.Fatalf("visible graphics len = %d, want 1", len(visible))
	}
	if visible[0].row != 1 || visible[0].col != 0 {
		t.Fatalf("visible sixel = %d,%d, want 1,0", visible[0].row, visible[0].col)
	}
}

func TestClearGraphicsDestroysCachedImages(t *testing.T) {
	vxImage := &recordingImage{}
	vt := New()
	vt.graphics = []*Image{{
		vaxii: []*vaxisImage{{
			vx:      &vaxis.Vaxis{},
			vxImage: vxImage,
		}},
	}}

	vt.clearGraphicsLocked()

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len = %d, want 0", len(vt.graphics))
	}
	if got, want := vxImage.destroyed, 1; got != want {
		t.Fatalf("destroy calls = %d, want %d", got, want)
	}
}

func TestRISClearsSixelGraphics(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.update(ansi.DCS{
		Final: 'q',
		Data:  []rune("#0;2;100;0;0#0@"),
	})
	if len(vt.graphics) != 1 {
		t.Fatalf("graphics len before RIS = %d, want 1", len(vt.graphics))
	}

	vt.update(testESC('c'))

	if len(vt.graphics) != 0 {
		t.Fatalf("graphics len after RIS = %d, want 0", len(vt.graphics))
	}
}

func TestEraseDisplayCompleteClearsSixelGraphics(t *testing.T) {
	tests := []struct {
		name string
		ps   int
	}{
		{name: "display", ps: 2},
		{name: "display and scrollback", ps: 22},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(80, 24)

			vt.update(ansi.DCS{
				Final: 'q',
				Data:  []rune("#0;2;100;0;0#0@"),
			})
			if len(vt.graphics) != 1 {
				t.Fatalf("graphics len before ED = %d, want 1", len(vt.graphics))
			}

			vt.ed(tt.ps, false)

			if len(vt.graphics) != 0 {
				t.Fatalf("graphics len after ED %d = %d, want 0", tt.ps, len(vt.graphics))
			}
		})
	}
}
