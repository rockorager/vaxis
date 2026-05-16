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
