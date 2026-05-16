package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestZeroWidthPrintAttachesToPreviousCell(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("o"))
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "ò  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestZeroWidthPrintAtStartIsIgnored(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestZeroWidthPrintAttachesToPendingWrapCell(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	vt.update(testPrint("a"))
	vt.update(testPrint("b"))
	vt.update(testPrint("c"))
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "abc̀\n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("zero-width print cleared pending wrap")
	}
}

func TestZeroWidthPrintAttachesToWideCharacterHead(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.activeScreen.cell(0, 0).Character.Grapheme, "橋̀"; got != want {
		t.Fatalf("wide cell grapheme = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 0).Character.Width, 2; got != want {
		t.Fatalf("wide cell width = %d, want %d", got, want)
	}
}

func TestVariationSelectorNarrowsPendingWrapCell(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.mode.graphemeCluster = true

	vt.update(ansi.Print{Grapheme: "🍋", Width: 2})
	vt.update(ansi.Print{Grapheme: "☔", Width: 2})
	if !vt.lastCol {
		t.Fatal("setup did not enter pending wrap")
	}

	vt.update(ansi.Print{Grapheme: "\uFE0E", Width: 0})

	if vt.lastCol {
		t.Fatal("variation selector did not clear pending wrap")
	}
	if vt.cursor.row != 0 || vt.cursor.col != 3 {
		t.Fatalf("cursor = %d,%d, want 0,3", vt.cursor.row, vt.cursor.col)
	}
	cell := vt.activeScreen.cell(0, 2)
	if got, want := cell.Grapheme, "☔︎"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(0, 3)
	if tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("tail cell = %#v, want blank", tail.Character)
	}
}

func TestVariationSelectorWidensAtRightEdge(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.mode.graphemeCluster = true
	vt.cursor.col = 1

	vt.update(ansi.Print{Grapheme: "#", Width: 1})
	vt.update(ansi.Print{Grapheme: "\uFE0F", Width: 0})

	cell := vt.activeScreen.cell(0, 1)
	if got, want := cell.Grapheme, "#️"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(0, 2)
	if tail.Grapheme != " " || tail.Width != 0 {
		t.Fatalf("tail cell = %#v, want spacer", tail.Character)
	}
	if !vt.lastCol {
		t.Fatal("variation selector did not set pending wrap")
	}

	vt.update(testPrint("X"))

	if got, want := vt.activeScreen.cell(1, 0).Grapheme, "X"; got != want {
		t.Fatalf("wrapped print = %q, want X", got)
	}
}

func TestVariationSelectorWidensPendingWrapCell(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.mode.graphemeCluster = true
	vt.cursor.col = 2
	vt.osc("8;id=vs16;http://example.com/vs16")

	vt.update(testPrint("#"))
	if !vt.lastCol {
		t.Fatal("setup did not enter pending wrap")
	}

	vt.update(ansi.Print{Grapheme: "\uFE0F", Width: 0})

	if got, want := vt.activeScreen.cell(1, 0).Grapheme, "#️"; got != want {
		t.Fatalf("wrapped cell grapheme = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(1, 0).Width, 2; got != want {
		t.Fatalf("wrapped cell width = %d, want %d", got, want)
	}
	if got := vt.activeScreen.cell(0, 2); got.Grapheme != " " || got.Width != 0 {
		t.Fatalf("previous row spacer = %#v, want spacer", got.Character)
	}
	if got := vt.activeScreen.cell(1, 1); got.Grapheme != " " || got.Width != 0 {
		t.Fatalf("wrapped tail = %#v, want spacer", got.Character)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("previous row was not marked wrapped")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("wrapped row was not marked as continuation")
	}
	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if vt.lastCol {
		t.Fatal("pending wrap stayed set after widened grapheme moved to next line")
	}
	for _, cell := range []*cell{vt.activeScreen.cell(0, 2), vt.activeScreen.cell(1, 0), vt.activeScreen.cell(1, 1)} {
		if got, want := cell.Hyperlink, "http://example.com/vs16"; got != want {
			t.Fatalf("hyperlink = %q, want %q", got, want)
		}
		if got, want := cell.HyperlinkParams, "id=vs16"; got != want {
			t.Fatalf("hyperlink params = %q, want %q", got, want)
		}
	}
}

func TestVariationSelectorWidensPreservesHyperlink(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.mode.graphemeCluster = true
	vt.cursor.col = 1
	vt.osc("8;id=emoji;https://example.com/emoji")

	vt.update(ansi.Print{Grapheme: "#", Width: 1})
	vt.update(ansi.Print{Grapheme: "\uFE0F", Width: 0})

	for _, col := range []column{1, 2} {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "https://example.com/emoji"; got != want {
			t.Fatalf("cell %d hyperlink = %q, want %q", col, got, want)
		}
		if got, want := cell.HyperlinkParams, "id=emoji"; got != want {
			t.Fatalf("cell %d hyperlink params = %q, want %q", col, got, want)
		}
	}
}

func TestVariationSelectorDoesNotWidenWhenGraphemeClusterModeDisabled(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(ansi.Print{Grapheme: "❤", Width: 1})
	vt.update(ansi.Print{Grapheme: "\uFE0F", Width: 0})

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "❤️"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if got := vt.activeScreen.cell(0, 1).Grapheme; got != "" {
		t.Fatalf("tail grapheme = %q, want empty", got)
	}
}

func TestMulticodepointGraphemeSplitsWhenGraphemeClusterModeDisabled(t *testing.T) {
	vt := New()
	vt.resize(10, 1)

	parseAndApply(t, vt, "👨\u200d👩\u200d👧")

	if got, want := vt.cursor.col, column(6); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 0).Grapheme, "👨\u200d"; got != want {
		t.Fatalf("first emoji grapheme = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 2).Grapheme, "👩\u200d"; got != want {
		t.Fatalf("second emoji grapheme = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 4).Grapheme, "👧"; got != want {
		t.Fatalf("third emoji grapheme = %q, want %q", got, want)
	}
}

func TestInvalidVariationSelectorIgnoredWhenGraphemeClusterModeDisabled(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "x\uFE0F")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "x"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if got := vt.activeScreen.cell(0, 1).Grapheme; got != "" {
		t.Fatalf("second cell grapheme = %q, want empty", got)
	}
}

func TestInvalidVariationSelectorIgnoredWhenGraphemeClusterModeEnabled(t *testing.T) {
	tests := []struct {
		name     string
		grapheme string
		width    int
		vs       string
		wantCol  column
	}{
		{name: "narrow base VS16", grapheme: "x", width: 1, vs: "\uFE0F", wantCol: 1},
		{name: "wide base VS15", grapheme: "🧠", width: 2, vs: "\uFE0E", wantCol: 2},
		{name: "wide base VS16", grapheme: "🧠", width: 2, vs: "\uFE0F", wantCol: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 1)
			vt.mode.graphemeCluster = true

			vt.update(ansi.Print{Grapheme: tt.grapheme, Width: tt.width})
			vt.update(ansi.Print{Grapheme: tt.vs, Width: 0})

			cell := vt.activeScreen.cell(0, 0)
			if got := cell.Grapheme; got != tt.grapheme {
				t.Fatalf("cell grapheme = %q, want %q", got, tt.grapheme)
			}
			if got := cell.Width; got != tt.width {
				t.Fatalf("cell width = %d, want %d", got, tt.width)
			}
			if got := vt.cursor.col; got != tt.wantCol {
				t.Fatalf("cursor col = %d, want %d", got, tt.wantCol)
			}
		})
	}
}

func TestParsedVariationSelectorDoesNotWidenWhenGraphemeClusterModeDisabled(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "❤️")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "❤️"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedVariationSelectorDoesNotNarrowWhenGraphemeClusterModeDisabled(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "☔︎")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "☔︎"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedVariationSelectorWidensWhenGraphemeClusterModeEnabled(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027h❤️")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "❤️"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedVariationSelectorRepeatedWhenGraphemeClusterModeEnabled(t *testing.T) {
	vt := New()
	vt.resize(6, 1)

	parseAndApply(t, vt, "\x1b[?2027h❤️❤️")

	if got, want := vt.cursor.col, column(4); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	for _, col := range []column{0, 2} {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Grapheme, "❤️"; got != want {
			t.Fatalf("cell %d grapheme = %q, want %q", col, got, want)
		}
		if got, want := cell.Width, 2; got != want {
			t.Fatalf("cell %d width = %d, want %d", col, got, want)
		}
	}
}

func TestParsedInvalidVariationSelectorInZWJGraphemeIsIgnored(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027h👩\uFE0E\u200d👦")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "👩\u200d👦"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedInvalidVariationSelectorThenCombiningMark(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027hn\uFE0F\u0303")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "n\u0303"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedDevanagariGraphemeWide(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027hक्\u200dष")

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Grapheme, "क्\u200dष"; got != want {
		t.Fatalf("cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("cell width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(0, 1)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("tail grapheme = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("tail width = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestParsedWideGraphemeWrapsToNextLine(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.cursor.col = 2

	parseAndApply(t, vt, "\x1b[?2027hक्\u200dष")

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if vt.lastCol {
		t.Fatal("pending wrap stayed set after wide grapheme wrapped")
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("previous row was not marked wrapped")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("wrapped row was not marked as a continuation")
	}
	if got := vt.activeScreen.cell(0, 2); got.Grapheme != "" || got.Width != 0 {
		t.Fatalf("previous row edge cell = %#v, want blank spacer", got.Character)
	}
	cell := vt.activeScreen.cell(1, 0)
	if got, want := cell.Grapheme, "क्\u200dष"; got != want {
		t.Fatalf("wrapped grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("wrapped width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(1, 1)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("tail grapheme = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("tail width = %d, want %d", got, want)
	}
}

func TestParsedWideGraphemeWrapsAtBottomOfScreen(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.cursor.row = 1
	vt.cursor.col = 2

	parseAndApply(t, vt, "\x1b[?2027hक्\u200dष")

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if vt.lastCol {
		t.Fatal("pending wrap stayed set after wide grapheme wrapped at bottom")
	}
	if got, want := vt.primaryScreen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback len = %d, want %d", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("scrolled row was not marked wrapped")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("bottom row was not marked as a continuation")
	}
	if got := vt.activeScreen.cell(0, 2); got.Grapheme != "" || got.Width != 0 {
		t.Fatalf("scrolled row edge cell = %#v, want blank spacer", got.Character)
	}
	cell := vt.activeScreen.cell(1, 0)
	if got, want := cell.Grapheme, "क्\u200dष"; got != want {
		t.Fatalf("wrapped grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 2; got != want {
		t.Fatalf("wrapped width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(1, 1)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("tail grapheme = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("tail width = %d, want %d", got, want)
	}
}

func TestParsedWideGraphemeOverwritesClearTail(t *testing.T) {
	vt := New()
	vt.resize(10, 1)

	parseAndApply(t, vt, "\x1b[?2027h👨\u200d👩\u200d👧")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.update(testPrint("X"))

	if got, want := vt.String(), " X        "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if tail := vt.activeScreen.cell(0, 0); tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("wide head after tail overwrite = %#v, want blank", tail.Character)
	}
}

func TestParsedWideGraphemeOverwriteHeadClearsTail(t *testing.T) {
	vt := New()
	vt.resize(10, 1)

	parseAndApply(t, vt, "\x1b[?2027h👨\u200d👩\u200d👧")
	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testPrint("X"))

	if got, want := vt.String(), "X         "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if tail := vt.activeScreen.cell(0, 1); tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("wide tail after head overwrite = %#v, want blank", tail.Character)
	}
}

func TestWideCharacterAtRightEdgePreservesHyperlinkAfterWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 5)
	vt.cursor.col = 9
	vt.osc("8;id=wide;http://example.com")

	vt.update(ansi.Print{Grapheme: "中", Width: 2})

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("previous row was not marked wrapped")
	}
	head := vt.activeScreen.cell(1, 0)
	if got, want := head.Grapheme, "中"; got != want {
		t.Fatalf("wide head grapheme = %q, want %q", got, want)
	}
	if got, want := head.Width, 2; got != want {
		t.Fatalf("wide head width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(1, 1)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("wide tail grapheme = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("wide tail width = %d, want %d", got, want)
	}
	for _, cell := range []*cell{head, tail} {
		if got, want := cell.Hyperlink, "http://example.com"; got != want {
			t.Fatalf("hyperlink = %q, want %q", got, want)
		}
		if got, want := cell.HyperlinkParams, "id=wide"; got != want {
			t.Fatalf("hyperlink params = %q, want %q", got, want)
		}
	}
}
