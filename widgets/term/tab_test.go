package term

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestHorizontalTabs(t *testing.T) {
	vt := New()
	vt.resize(20, 5)

	printText(vt, "1")
	vt.ht()
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after first HT = %d, want %d", got, want)
	}

	vt.ht()
	if got, want := vt.cursor.col, column(16); got != want {
		t.Fatalf("cursor after second HT = %d, want %d", got, want)
	}

	vt.ht()
	if got, want := vt.cursor.col, column(19); got != want {
		t.Fatalf("cursor after HT at end = %d, want %d", got, want)
	}
	vt.ht()
	if got, want := vt.cursor.col, column(19); got != want {
		t.Fatalf("cursor after repeated HT at end = %d, want %d", got, want)
	}
}

func TestCursorHorizontalTabulationZeroDoesNotMove(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3

	vt.update(testCSI('I', []uint32{0}))

	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after CSI 0 I = %d, want %d", got, want)
	}
}

func TestCursorHorizontalTabulationDefaultMovesOnce(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3

	vt.update(testCSI('I', nil))

	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after CSI I = %d, want %d", got, want)
	}
}

func TestHorizontalTabsStartingOnTabStop(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 8
	printText(vt, "X")
	vt.update(testCSI('G', []uint32{9}))

	vt.ht()
	printText(vt, "A")

	if got, want := trimScreenString(vt.String()), "        X       A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestHorizontalTabsWithRightMargin(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.margin.left = 2
	vt.margin.right = 5
	vt.cursor.col = 0
	printText(vt, "X")

	vt.ht()
	printText(vt, "A")

	if got, want := trimScreenString(vt.String()), "X    A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestHorizontalTabRightOfRightMarginDoesNotMove(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.margin.left = 2
	vt.margin.right = 5
	vt.cursor.col = 10

	vt.ht()

	if got, want := vt.cursor.col, column(10); got != want {
		t.Fatalf("cursor after HT right of margin = %d, want %d", got, want)
	}
}

func TestHorizontalTabsBack(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 19

	vt.cbt(1)
	if got, want := vt.cursor.col, column(16); got != want {
		t.Fatalf("cursor after first CBT = %d, want %d", got, want)
	}
	vt.cbt(1)
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after second CBT = %d, want %d", got, want)
	}
	vt.cbt(1)
	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor after third CBT = %d, want %d", got, want)
	}
	vt.cbt(1)
	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor after repeated CBT at start = %d, want %d", got, want)
	}
}

func TestCursorHorizontalTabulationBackZeroDoesNotMove(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 10

	vt.update(testCSI('Z', []uint32{0}))

	if got, want := vt.cursor.col, column(10); got != want {
		t.Fatalf("cursor after CSI 0 Z = %d, want %d", got, want)
	}
}

func TestCursorHorizontalTabulationBackDefaultMovesOnce(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 10

	vt.update(testCSI('Z', nil))

	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after CSI Z = %d, want %d", got, want)
	}
}

func TestHorizontalTabBackUsesLeftMarginOnlyInOriginMode(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.margin.left = 2
	vt.margin.right = 5
	vt.mode.decom = true
	vt.cursor.col = 3
	printText(vt, "X")

	vt.cbt(1)
	printText(vt, "A")

	if got, want := trimScreenString(vt.String()), "  AX"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestHorizontalTabBackBeforeLeftMarginStaysBeforeMargin(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.mode.decom = true
	vt.decsc()
	vt.mode.declrmm = true
	vt.decslrm(testCSI('s', []uint32{5, 0}))
	vt.decrc()

	vt.cbt(1)
	printText(vt, "X")

	if got, want := trimScreenString(vt.String()), "X"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestHorizontalTabBackIgnoresLeftMarginOutsideOriginMode(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.margin.left = 5
	vt.margin.right = 10
	vt.cursor.col = 4

	vt.cbt(1)

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor after CBT outside origin mode = %d, want %d", got, want)
	}
}

func TestTabSetAndClear(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3

	vt.hts()
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after custom tab set = %d, want %d", got, want)
	}

	vt.tbc(0)
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after clearing custom tab = %d, want %d", got, want)
	}
}

func TestTabClearRequiresExplicitParameter(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3
	vt.hts()

	vt.update(testCSI('g', nil))
	vt.cursor.col = 0
	vt.ht()

	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after CSI g = %d, want %d", got, want)
	}
}

func TestTabClearCurrentWithExplicitZero(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3
	vt.hts()

	vt.update(testCSI('g', []uint32{0}))
	vt.cursor.col = 0
	vt.ht()

	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after CSI 0 g = %d, want %d", got, want)
	}
}

func TestTabClearAll(t *testing.T) {
	vt := New()
	vt.resize(30, 5)

	vt.tbc(3)
	vt.ht()

	if got, want := vt.cursor.col, column(29); got != want {
		t.Fatalf("cursor after clearing all tabs = %d, want %d", got, want)
	}
}

func TestTabClearAllWithCSI(t *testing.T) {
	vt := New()
	vt.resize(30, 5)

	vt.update(testCSI('g', []uint32{3}))
	vt.ht()

	if got, want := vt.cursor.col, column(29); got != want {
		t.Fatalf("cursor after CSI 3 g = %d, want %d", got, want)
	}
}

func TestCursorTabulationControlSetAndClear(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3

	vt.update(testCSI('W', nil))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after CSI W tab set = %d, want %d", got, want)
	}

	vt.tbc(0)
	vt.update(testCSI('W', []uint32{0}))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after CSI 0 W tab set = %d, want %d", got, want)
	}

	vt.update(testCSI('W', []uint32{2}))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after CSI 2 W tab clear = %d, want %d", got, want)
	}
}

func TestCursorTabulationControlClearAll(t *testing.T) {
	vt := New()
	vt.resize(20, 5)

	vt.update(testCSI('W', []uint32{5}))
	vt.ht()
	if got, want := vt.cursor.col, column(19); got != want {
		t.Fatalf("cursor after CSI 5 W clear all = %d, want %d", got, want)
	}
}

func TestPrivateCursorTabulationControlResetsTabs(t *testing.T) {
	vt := New()
	vt.resize(20, 5)

	vt.update(testCSI('W', []uint32{5}))
	vt.update(testCSI('W', []uint32{5}, '?'))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after private CSI ? 5 W = %d, want %d", got, want)
	}
}

func TestPrivateCursorTabulationControlNoParamsNoOps(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.update(testCSI('W', []uint32{5}))

	vt.update(testCSI('W', nil, '?'))
	vt.cursor.col = 0
	vt.ht()

	if got, want := vt.cursor.col, column(19); got != want {
		t.Fatalf("cursor after private CSI ? W = %d, want %d", got, want)
	}
}

func TestPrivateCursorTabulationControlIgnoresUnsupportedParameters(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3
	vt.hts()

	vt.update(testCSI('W', []uint32{2}, '?'))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after unsupported private CSI ? 2 W = %d, want %d", got, want)
	}

	vt.update(testCSI('W', []uint32{1, 2, 3}, '?'))
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after invalid private CSI ? 1;2;3 W = %d, want %d", got, want)
	}
}

func TestTabClearOverflowingParameterIgnored(t *testing.T) {
	vt := New()
	vt.resize(20, 5)
	vt.cursor.col = 3
	vt.hts()

	parseAndApply(t, vt, "\x1b[388888888888888888888888888888888888g\x1b[0m")
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after overflowing tab clear parameter = %d, want %d", got, want)
	}
}

func TestResizeResetsTabStopsForNewWidth(t *testing.T) {
	vt := New()
	vt.resize(4, 1)
	vt.cursor.col = 2
	vt.hts()
	vt.ht()
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("cursor after HT in narrow screen = %d, want %d", got, want)
	}

	vt.resize(12, 1)
	vt.cursor.col = 0
	vt.ht()
	if got, want := vt.cursor.col, column(8); got != want {
		t.Fatalf("cursor after HT in resized screen = %d, want %d", got, want)
	}
}

func parseAndApply(t *testing.T, vt *Model, input string) {
	t.Helper()
	parser := ansi.NewParser(strings.NewReader(input), ansi.ParserModeOutput)
	vt.parser = parser
	for {
		seq := <-parser.Next()
		if _, ok := seq.(ansi.EOF); ok {
			return
		}
		vt.update(seq)
	}
}
