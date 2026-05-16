package term

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestResizeSameWidthMoreRowsPullsFromScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")

	vt.resize(5, 5)

	if got, want := vt.primaryScreen.scrollbackLen(), 0; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := viewportString(vt), "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeSameWidthLessRowsPushesIntoScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")

	vt.resize(5, 2)

	if got, want := vt.primaryScreen.scrollbackLen(), 3; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := viewportString(vt), "4ABCD\n5EFGH"; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeSameWidthLessRowsNoScrollbackDiscardsTopRows(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.primaryScreen.state.scrollbackLimit = 0
	setScreenLine(vt.primaryScreen, 0, "1ABCD")
	setScreenLine(vt.primaryScreen, 1, "2EFGH")
	setScreenLine(vt.primaryScreen, 2, "3IJKL")
	vt.cursor.row = 0
	vt.cursor.col = 0

	vt.resize(5, 1)

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
	if got, want := vt.String(), "3IJKL"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor after resize = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}

func TestResizeSameWidthLessRowsKeepsCursorOnLastLine(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.primaryScreen.state.scrollbackLimit = 0
	setScreenLine(vt.primaryScreen, 0, "1ABCD")
	setScreenLine(vt.primaryScreen, 1, "2EFGH")
	setScreenLine(vt.primaryScreen, 2, "3IJKL")
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.resize(5, 1)

	if got, want := vt.String(), "3IJKL"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 1 {
		t.Fatalf("cursor after resize = %d,%d, want 0,1", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col).Grapheme; got != "I" {
		t.Fatalf("cursor cell after resize = %q, want %q", got, "I")
	}
}

func TestResizeSameWidthLessRowsTrimsBlankTrailingRows(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	for r := 1; r < vt.height(); r += 1 {
		line := vt.primaryScreen.line(row(r))
		for col := range line {
			line[col].Background = vaxis.IndexColor(1)
		}
	}

	vt.resize(5, 2)

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
	if got, want := vt.String(), "1ABCD\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeSameWidthLessRowsWithEmptyTrailingScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1    ", "2    ", "3    ", "4    ", "5    ", "6    ", "7    ", "8    ")
	vt.primaryScreen.scrollClear(vt.cursor.Style.Background)
	vt.cursor.row = 0
	vt.cursor.col = 0
	printText(vt, "A")
	vt.cr()
	vt.lf()
	printText(vt, "B")
	cursor := vt.cursor

	vt.resize(5, 2)

	if vt.cursor.row != cursor.row || vt.cursor.col != cursor.col {
		t.Fatalf("cursor after resize = %d,%d, want %d,%d", vt.cursor.row, vt.cursor.col, cursor.row, cursor.col)
	}
	if got, want := trimScreenString(vt.String()), "A\nB"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeSameWidthPreservesRowMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abcd")

	vt.resize(3, 3)

	if !vt.primaryScreen.row(0).wrapped {
		t.Fatal("resize did not preserve wrapped metadata")
	}
	if !vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("resize did not preserve wrap continuation metadata")
	}
}

func TestResizeSameWidthMoreRowsPullsSoftWrapMetadataFromScrollback(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	vt.primaryScreen.state.scrollback.append([]cell{cellString("1"), cellString("A")}, screenRow{wrapped: true}, defaultScrollbackLines)
	vt.primaryScreen.state.scrollback.append([]cell{cellString("2"), cellString("B")}, screenRow{wrapContinuation: true}, defaultScrollbackLines)
	setScreenLine(vt.primaryScreen, 0, "3C")
	setScreenLine(vt.primaryScreen, 1, "4E")
	setScreenLine(vt.primaryScreen, 2, "5F")
	vt.primaryScreen.row(0).wrapped = true
	vt.primaryScreen.row(1).wrapContinuation = true
	vt.primaryScreen.row(2).wrapped = true

	vt.resize(2, 6)

	if got, want := viewportString(vt), "1A\n2B\n3C\n4E\n5F\n  "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if !vt.primaryScreen.row(0).wrapped {
		t.Fatal("pulled scrollback row lost wrapped metadata")
	}
	if !vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("pulled scrollback row lost wrap continuation metadata")
	}
	if !vt.primaryScreen.row(2).wrapped {
		t.Fatal("active row lost wrapped metadata")
	}
	if !vt.primaryScreen.row(3).wrapContinuation {
		t.Fatal("active continuation row lost metadata")
	}
}

func TestResizeSameWidthClampsViewport(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")
	vt.scrollViewport(2)

	vt.resize(5, 5)

	if got, want := vt.scrollOffset, 0; got != want {
		t.Fatalf("scroll offset = %d, want %d", got, want)
	}
}

func TestRestoreCursorAfterResizeClampsToScreen(t *testing.T) {
	vt := New()
	vt.resize(10, 5)
	vt.cursor.row = 0
	vt.cursor.col = 9
	vt.decsc()

	vt.resize(5, 5)
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "    X\n     \n     \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestRestoreCursorClearsPendingWrapWhenNoLongerAtRightEdge(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abc")
	if !vt.lastCol {
		t.Fatal("terminal did not enter pending wrap state")
	}
	vt.decsc()

	vt.margin.right = 4
	vt.primaryScreen = newScreenBuffer(5, 2, defaultScrollbackLines)
	vt.activeScreen = vt.primaryScreen
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "   X \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeWiderReflowsSoftWrappedRows(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	printText(vt, "1A2B")

	vt.resize(5, 3)

	if got, want := vt.String(), "1A2B \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeWiderPerfectSplitMatchesGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD2EFGH3IJKL")

	vt.resize(10, 3)

	if got, want := trimScreenString(vt.String()), "1ABCD2EFGH\n3IJKL"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeWiderReflowEndsInNewlineMatchesGhostty(t *testing.T) {
	vt := New()
	vt.resize(6, 3)
	printText(vt, "1ABCD2EFGH")
	vt.cr()
	vt.lf()
	printText(vt, "3IJKL")
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.resize(10, 3)

	if got, want := trimScreenString(vt.String()), "1ABCD2EFGH\n3IJKL"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col).Grapheme; got != "3" {
		t.Fatalf("cursor cell after resize = %q, want %q", got, "3")
	}
}

func TestResizeNarrowerPreservesExplicitRows(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	setScreenLine(vt.primaryScreen, 0, "1AB  ")
	setScreenLine(vt.primaryScreen, 1, "2EF  ")
	setScreenLine(vt.primaryScreen, 2, "3IJ  ")

	vt.resize(3, 3)

	if got, want := vt.String(), "1AB\n2EF\n3IJ"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerReflowsOneSoftWrappedLine(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")

	vt.resize(3, 3)

	if got, want := vt.String(), "1AB\nCD \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerReflowsTrimmedRowsLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "3IJKL")
	vt.cr()
	vt.lf()
	printText(vt, "4ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "5EFGH")

	vt.resize(3, 3)

	if got, want := viewportString(vt), "CD \n5EF\nGH "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeNarrowerReflowsTrimmedRowsAndScrollbackLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "3IJKL")
	vt.cr()
	vt.lf()
	printText(vt, "4ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "5EFGH")

	vt.resize(3, 3)

	if got, want := viewportString(vt), "CD \n5EF\nGH "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(0), "3IJ"; got != want {
		t.Fatalf("scrollback row after resize = %q, want %q", got, want)
	}
}

func TestResizeNarrowerReflowsScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD2EFGH3IJKL4ABCD5EFGH")

	vt.resize(3, 3)

	if got, want := viewportString(vt), "CD5\nEFG\nH  "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackLen(), 6; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(0), "1AB"; got != want {
		t.Fatalf("first scrollback row = %q, want %q", got, want)
	}
}

func TestResizeNarrowerReflowsPreviouslyWrappedScrollbackAndPreservesCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD2EFGH3IJKL4ABCD5EFGH")
	vt.cursor.row = 2
	vt.cursor.col = 4

	vt.resize(3, 3)

	if got, want := viewportString(vt), "CD5\nEFG\nH  "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 0 {
		t.Fatalf("cursor after resize = %d,%d, want 2,0", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col).Grapheme; got != "H" {
		t.Fatalf("cursor cell after resize = %q, want %q", got, "H")
	}
}

func TestResizeNarrowerReflowsPreviouslyWrappedRowsLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "3IJKL4ABCD5EFGH")

	vt.resize(3, 3)

	if got, want := vt.String(), "ABC\nD5E\nFGH"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeMoreRowsLessColsWithReflowAndScrollbackLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "2EFGH3IJKL")
	vt.cr()
	vt.lf()
	printText(vt, "4MNOP")

	vt.resize(2, 10)

	if got, want := viewportString(vt), "BC\nD \n2E\nFG\nH3\nIJ\nKL\n4M\nNO\nP "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeWiderWithScrollbackScrolledUpPreservesViewport(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1    ", "2    ", "3    ", "4    ", "5    ", "6    ", "7    ", "8    ")
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.scrollViewport(4)
	if got, want := viewportString(vt), "2    \n3    \n4    "; got != want {
		t.Fatalf("scrolled viewport = %q, want %q", got, want)
	}

	vt.resize(8, 3)

	if got, want := viewportString(vt), "2       \n3       \n4       "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got, want := vt.String(), "6       \n7       \n8       "; got != want {
		t.Fatalf("active screen after resize = %q, want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 1 {
		t.Fatalf("cursor after resize = %d,%d, want 2,1", vt.cursor.row, vt.cursor.col)
	}
}

func TestResizeWiderReflowsPopulatedScrollbackAndPreservesCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.primaryScreen.state.scrollback.append([]cell{cellString("1"), cellString("A"), cellString("B"), cellString("C"), cellString("D")}, screenRow{}, defaultScrollbackLines)
	vt.primaryScreen.state.scrollback.append([]cell{cellString("2"), cellString("E"), cellString("F"), cellString("G"), cellString("H")}, screenRow{}, defaultScrollbackLines)
	setScreenLine(vt.primaryScreen, 0, "3IJKL")
	setScreenLine(vt.primaryScreen, 1, "4ABCD")
	setScreenLine(vt.primaryScreen, 2, "5EFGH")
	vt.primaryScreen.row(1).wrapped = true
	vt.primaryScreen.row(2).wrapContinuation = true
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.resize(10, 3)

	if got, want := viewportString(vt), "2EFGH     \n3IJKL     \n4ABCD5EFGH"; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col).Grapheme; got != "5" {
		t.Fatalf("cursor cell after resize = %q, want %q", got, "5")
	}
}

func TestResizeWiderReflowsWrappedScrollbackLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "2EFGH")
	vt.cr()
	vt.lf()
	printText(vt, "3IJKL")
	vt.cr()
	vt.lf()
	printText(vt, "4ABCD5EFGH")
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.resize(10, 3)

	if got, want := viewportString(vt), "2EFGH     \n3IJKL     \n4ABCD5EFGH"; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got := vt.activeScreen.cell(vt.cursor.row, vt.cursor.col).Grapheme; got != "5" {
		t.Fatalf("cursor cell after resize = %q, want %q", got, "5")
	}
}

func TestResizeNarrowerWithScrollbackScrolledUpPreservesViewport(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1    ", "2    ", "3    ", "4    ", "5    ", "6    ", "7    ", "8    ")
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.scrollViewport(4)
	if got, want := viewportString(vt), "2    \n3    \n4    "; got != want {
		t.Fatalf("scrolled viewport = %q, want %q", got, want)
	}

	vt.resize(4, 3)

	if got, want := viewportString(vt), "2   \n3   \n4   "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
	if got, want := vt.String(), "6   \n7   \n8   "; got != want {
		t.Fatalf("active screen after resize = %q, want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 1 {
		t.Fatalf("cursor after resize = %d,%d, want 2,1", vt.cursor.row, vt.cursor.col)
	}
}

func TestResizeReflowNormalizesViewportWhenPinnedRowBecomesActive(t *testing.T) {
	vt := New()
	vt.resize(2, 10)

	for i := 0; i < 40; i += 1 {
		if i >= vt.height() {
			vt.scrollUp(1)
		}
		r := i
		if i >= vt.height() {
			r = vt.height() - 1
		}
		setScreenLine(vt.primaryScreen, r, "AA")
		if i%2 == 0 {
			vt.primaryScreen.row(row(r)).wrapped = true
			vt.primaryScreen.row(row(r)).wrapContinuation = false
		} else {
			vt.primaryScreen.row(row(r)).wrapped = false
			vt.primaryScreen.row(row(r)).wrapContinuation = true
		}
	}
	if got, want := vt.primaryScreen.scrollbackLen(), 30; got != want {
		t.Fatalf("scrollback len before resize = %d, want %d", got, want)
	}

	vt.scrollOffset = 2
	vt.resize(4, 10)

	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset after reflow = %d, want active viewport", got)
	}
	if got, want := vt.primaryScreen.scrollbackLen(), 10; got != want {
		t.Fatalf("scrollback len after resize = %d, want %d", got, want)
	}
}

func TestResizeReflowPreservesBlankActiveRowsAfterClear(t *testing.T) {
	vt := New()
	vt.resize(10, 5)
	writeViewportLines(vt,
		"old0000000",
		"old1111111",
		"old2222222",
		"old3333333",
		"old4444444",
		"old5555555",
	)
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.ed(2, false)
	vt.update(testCSI('H', []uint32{}))
	printText(vt, "abcdefghijklmnopqrst")

	vt.resize(5, 5)

	if got, want := viewportString(vt), "abcde\nfghij\nklmno\npqrst\n     "; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeReflowKeepsWrappedOutputBeforeRedrawnPrompt(t *testing.T) {
	vt := New()
	vt.resize(10, 4)
	setScreenLine(vt.primaryScreen, 0, "old0000000")
	setScreenLine(vt.primaryScreen, 1, "old1111111")
	setScreenLine(vt.primaryScreen, 2, "abcdefghi")
	setScreenLine(vt.primaryScreen, 3, ">")
	vt.primaryScreen.row(3).semanticPrompt = semanticPromptPrimary
	vt.cursor.row = 3
	vt.cursor.col = 1
	vt.cursor.semanticContent = semanticPromptContent

	vt.resize(5, 4)
	vt.cr()
	printText(vt, ">")

	if got, want := viewportString(vt), "11111\nabcde\nfghi \n>    "; got != want {
		t.Fatalf("viewport after prompt redraw = %q, want %q", got, want)
	}
}

func TestResizeNarrowerReflowsScrollbackWithoutDroppingLogicalLines(t *testing.T) {
	vt := New()
	vt.resize(40, 8)

	for i := 1; i <= 18; i += 1 {
		printText(vt, fmt.Sprintf("L%02d:%s", i, strings.Repeat("x", 36)))
		vt.cr()
		vt.lf()
	}

	vt.resize(20, 8)

	screen := fullScreenString(vt.primaryScreen)
	for i := 1; i <= 18; i += 1 {
		marker := fmt.Sprintf("L%02d:", i)
		if !strings.Contains(screen, marker) {
			t.Fatalf("missing logical line marker %q after resize; screen:\n%s", marker, screen)
		}
	}
}

func TestResizeNarrowerPreservesScrollbackWithPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 4)

	for i := 1; i <= 8; i += 1 {
		printText(vt, fmt.Sprintf("L%02dxxxxxxx", i))
		if i < 8 {
			vt.cr()
			vt.lf()
		}
	}
	if !vt.lastCol {
		t.Fatal("setup did not leave cursor in pending wrap")
	}

	vt.resize(5, 4)

	screen := fullScreenString(vt.primaryScreen)
	for i := 1; i <= 8; i += 1 {
		marker := fmt.Sprintf("L%02d", i)
		if !strings.Contains(screen, marker) {
			t.Fatalf("missing logical line marker %q after resize; screen:\n%s", marker, screen)
		}
	}
}

func fullScreenString(screen screenBuffer) string {
	var out strings.Builder
	for i := 0; i < screen.scrollbackLen(); i += 1 {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(screen.scrollbackString(i))
	}
	for r := 0; r < screen.height(); r += 1 {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(lineString(screen.line(row(r))))
	}
	return out.String()
}

func TestResizeWiderRemapsCursorToSameLogicalCell(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	printText(vt, "1A2B")
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.resize(5, 3)
	vt.update(testPrint("X"))

	if got, want := vt.String(), "1A2X \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerRemapsCursorToSameLogicalCell(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	vt.cursor.row = 0
	vt.cursor.col = 4

	vt.resize(3, 3)
	vt.update(testPrint("X"))

	if got, want := vt.String(), "1AB\nCX \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerCursorInWrappedRowMovesToReflowedActiveRow(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	setScreenLine(vt.primaryScreen, 0, "abcd")
	setScreenLine(vt.primaryScreen, 1, "abcd")
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.resize(2, 2)
	vt.update(testPrint("X"))

	if got, want := vt.String(), "ab\nXd"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerBlankLinesBetweenMatchGhostty(t *testing.T) {
	vt := New()
	vt.resize(4, 3)
	setScreenLine(vt.primaryScreen, 0, "abcd")
	setScreenLine(vt.primaryScreen, 2, "abcd")

	vt.resize(2, 3)

	if got, want := vt.primaryScreen.scrollbackLen(), 2; got != want {
		t.Fatalf("scrollback len after resize = %d, want %d", got, want)
	}
	if got, want := viewportString(vt), "  \nab\ncd"; got != want {
		t.Fatalf("viewport after resize = %q, want %q", got, want)
	}
}

func TestResizeSameWidthUsesAltScreenCursorDeltaWhenAltActive(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")
	vt.decset(testCSI('h', []uint32{1049}, '?'))
	vt.cursor.row = 1
	vt.cursor.col = 0

	vt.resize(5, 5)
	vt.update(testPrint("X"))

	if got, want := vt.String(), "     \nX    \n     \n     \n     "; got != want {
		t.Fatalf("alt screen after resize = %q, want %q", got, want)
	}
}

func TestResizeSameWidthRemapsInactiveAltSavedCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.decset(testCSI('h', []uint32{47}, '?'))
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.decsc()
	vt.decrst(testCSI('l', []uint32{47}, '?'))

	vt.resize(5, 2)
	vt.decset(testCSI('h', []uint32{47}, '?'))
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "X    \n     "; got != want {
		t.Fatalf("alt screen after resize = %q, want %q", got, want)
	}
}

func TestResizeResetsMargins(t *testing.T) {
	vt := New()
	vt.resize(8, 5)
	vt.margin.top = 2
	vt.margin.bottom = 3
	vt.margin.left = 2
	vt.margin.right = 4

	vt.resize(8, 5)

	if vt.margin.top != 0 {
		t.Fatalf("top margin = %d, want 0", vt.margin.top)
	}
	if got, want := vt.margin.bottom, row(4); got != want {
		t.Fatalf("bottom margin = %d, want %d", got, want)
	}
	if vt.margin.left != 0 {
		t.Fatalf("left margin = %d, want 0", vt.margin.left)
	}
	if got, want := vt.margin.right, column(7); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
}

func TestResizeNarrowerWithWraparoundOffDoesNotReflow(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.mode.decawm = false
	printText(vt, "0123")

	vt.resize(2, 2)

	if got, want := vt.String(), "01\n  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerWithWraparoundOnReflows(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.mode.decawm = true
	printText(vt, "0123")

	vt.resize(2, 2)

	if got, want := vt.String(), "01\n23"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerAltScreenDoesNotReflow(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.decset(testCSI('h', []uint32{1049}, '?'))
	vt.mode.decawm = true
	printText(vt, "0123")

	vt.resize(2, 2)

	if got, want := vt.String(), "01\n  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNarrowerWithWideCharacterThenPrintWide(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	vt.update(testPrint("x"))
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	withoutPanic(t, func() {
		vt.resize(2, 3)
		vt.update(testCSI('H', []uint32{1, 2}))
		vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	})

	if got, want := vt.activeScreen.cell(0, 0).Grapheme, "x"; got != want {
		t.Fatalf("first cell = %q, want %q", got, want)
	}
}

func TestResizeNarrowerWrapsWideCharacterAtBoundary(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	vt.update(testPrint("x"))
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	vt.resize(2, 3)

	if got, want := vt.String(), "x \n😀 \n  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 1).Grapheme, ""; got != want {
		t.Fatalf("row 0 boundary cell = %q, want blank", got)
	}
	if got, want := vt.activeScreen.cell(1, 0).Grapheme, "😀"; got != want {
		t.Fatalf("row 1 wide head = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(1, 1).Width, 0; got != want {
		t.Fatalf("row 1 wide tail width = %d, want %d", got, want)
	}
}

func TestResizeNarrowerRemapsCursorAfterWrappedWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	vt.update(testPrint("x"))
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.resize(2, 3)
	vt.update(testPrint("X"))

	if got, want := vt.String(), "x \nX \n  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeWiderUnwrapsWideCharacterFromBoundary(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "  ")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	vt.resize(4, 2)

	if got, want := vt.String(), "  😀 \n    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 2).Grapheme, "😀"; got != want {
		t.Fatalf("row 0 wide head = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 3).Width, 0; got != want {
		t.Fatalf("row 0 wide tail width = %d, want %d", got, want)
	}
}

func TestResizeWiderKeepsWideCharacterWrappedWhenStillNoRoom(t *testing.T) {
	vt := New()
	vt.resize(2, 2)
	printText(vt, "xx")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	vt.resize(3, 2)

	if got, want := vt.String(), "xx \n😀  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 2).Grapheme, ""; got != want {
		t.Fatalf("row 0 boundary cell = %q, want blank", got)
	}
	if got, want := vt.activeScreen.cell(1, 0).Grapheme, "😀"; got != want {
		t.Fatalf("row 1 wide head = %q, want %q", got, want)
	}
}

func TestResizeWiderRequiresWideCharacterSpacerHead(t *testing.T) {
	vt := New()
	vt.resize(2, 2)
	printText(vt, "xx")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	vt.resize(3, 2)

	if got, want := vt.String(), "xx \n😀  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.activeScreen.cell(0, 2)
	if got, want := head.Grapheme, ""; got != want {
		t.Fatalf("spacer head grapheme = %q, want blank", got)
	}
	if got, want := head.Width, 0; got != want {
		t.Fatalf("spacer head width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(1, 1)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("wide tail grapheme = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("wide tail width = %d, want %d", got, want)
	}
}

func TestResizeWithLeftRightMarginSetDoesNotPanic(t *testing.T) {
	vt := New()
	vt.resize(70, 23)

	withoutPanic(t, func() {
		vt.update(testCSI('h', []uint32{69}, '?'))
		vt.update(testPrint("0"))
		vt.update(testCSI('s', []uint32{2, 0}))
		vt.rep(1850)
		vt.resize(70, 23)
	})

	if got, want := vt.margin.left, column(0); got != want {
		t.Fatalf("left margin after resize = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(69); got != want {
		t.Fatalf("right margin after resize = %d, want %d", got, want)
	}
}

func TestResizeNoReflowPreservesSoftWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	printText(vt, "1A2B")
	if !vt.primaryScreen.row(0).wrapped {
		t.Fatal("source row was not marked wrapped before resize")
	}
	if !vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("destination row was not marked wrap continuation before resize")
	}
	vt.mode.decawm = false

	vt.resize(3, 5)

	if !vt.primaryScreen.row(0).wrapped {
		t.Fatal("source row wrap metadata was not preserved")
	}
	if !vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("destination row wrap continuation was not preserved")
	}
	if got, want := vt.String(), "1A \n2B \n   \n   \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestResizeNoReflowPreservesSemanticPromptMetadata(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	vt.cr()
	vt.lf()
	vt.osc("133;P")
	printText(vt, "2EFGH")
	vt.osc("133;C")
	vt.cr()
	vt.lf()
	printText(vt, "3IJKL")
	vt.mode.decawm = false

	vt.resize(10, 3)

	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row 0 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(1).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("row 1 semantic prompt = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.row(2).semanticPrompt, semanticPromptNone; got != want {
		t.Fatalf("row 2 semantic prompt = %d, want %d", got, want)
	}
}

func TestResizeReflowPreservesSemanticPromptMetadata(t *testing.T) {
	vt := New()
	vt.resize(5, 4)
	vt.osc("133;P")
	printText(vt, "1234567890")
	vt.shellRedrawsPrompt = semanticPromptRedrawFalse

	vt.resize(3, 4)

	if got, want := vt.String(), "123\n456\n789\n0  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.primaryScreen.row(0).semanticPrompt, semanticPromptPrimary; got != want {
		t.Fatalf("row 0 semantic prompt = %d, want %d", got, want)
	}
	for r := row(1); r < 4; r += 1 {
		if got, want := vt.primaryScreen.row(r).semanticPrompt, semanticPromptContinuation; got != want {
			t.Fatalf("row %d semantic prompt = %d, want %d", r, got, want)
		}
	}
	if got, want := vt.primaryScreen.cell(2, 0).semanticContent, semanticPromptContent; got != want {
		t.Fatalf("reflowed cell semantic content = %d, want %d", got, want)
	}
}

func TestResizeClearsPromptForRedraw(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	printText(vt, "ABCDE")
	vt.cr()
	vt.lf()
	vt.osc("133;P")
	printText(vt, "> ")
	vt.osc("133;I")
	printText(vt, "echo")

	vt.resize(20, 3)

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(6); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if got, want := trimScreenString(vt.String()), "ABCDE"; got != want {
		t.Fatalf("screen after prompt redraw clear = %q, want %q", got, want)
	}
}

func TestResizePromptRedrawFalsePreservesPrompt(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	vt.osc("133;A;redraw=0")
	printText(vt, "> ")
	vt.osc("133;I")
	printText(vt, "echo")

	vt.resize(20, 3)

	if got, want := trimScreenString(vt.String()), "> echo"; got != want {
		t.Fatalf("screen after redraw=0 resize = %q, want %q", got, want)
	}
}

func TestResizeDisablesSynchronizedOutput(t *testing.T) {
	vt := New()
	vt.resize(80, 24)
	vt.update(testCSI('h', []uint32{2026}, '?'))

	if !vt.mode.synchronizedOutput {
		t.Fatal("synchronized output mode was not enabled")
	}
	vt.resize(100, 40)
	if vt.mode.synchronizedOutput {
		t.Fatal("resize did not disable synchronized output mode")
	}
}

func TestResizeEventDisablesSynchronizedOutput(t *testing.T) {
	vt := New()
	vt.resize(80, 24)
	vt.update(testCSI('h', []uint32{2026}, '?'))

	vt.Update(vaxis.Resize{Cols: 100, Rows: 40, XPixel: 900, YPixel: 720})
	if vt.mode.synchronizedOutput {
		t.Fatal("resize event did not disable synchronized output mode")
	}
}

func TestResizeEventResizesModelWithoutPty(t *testing.T) {
	vt := New()
	vt.resize(80, 24)

	vt.Update(vaxis.Resize{Cols: 20, Rows: 8})

	if got, want := vt.width(), 20; got != want {
		t.Fatalf("width after resize event = %d, want %d", got, want)
	}
	if got, want := vt.height(), 8; got != want {
		t.Fatalf("height after resize event = %d, want %d", got, want)
	}
}

func TestSynchronizedOutputResetsAfterTimeout(t *testing.T) {
	oldDelay := synchronizedOutputResetDelay
	synchronizedOutputResetDelay = 10 * time.Millisecond
	defer func() {
		synchronizedOutputResetDelay = oldDelay
	}()

	vt := New()
	vt.resize(80, 24)
	vt.update(testCSI('h', []uint32{2026}, '?'))

	if !vt.mode.synchronizedOutput {
		t.Fatal("synchronized output mode was not enabled")
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		vt.mu.Lock()
		enabled := vt.mode.synchronizedOutput
		vt.mu.Unlock()
		if !enabled {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("synchronized output mode did not reset after timeout")
}

func TestResizePromptRedrawLastClearsOnlyCursorRow(t *testing.T) {
	vt := New()
	vt.resize(10, 4)
	printText(vt, "ABCDE")
	vt.cr()
	vt.lf()
	vt.osc("133;A;redraw=last")
	printText(vt, "> hello")
	vt.cr()
	vt.lf()
	vt.osc("133;I")
	printText(vt, "world")

	vt.resize(20, 4)

	if got, want := trimScreenString(vt.String()), "ABCDE\n> hello"; got != want {
		t.Fatalf("screen after redraw=last resize = %q, want %q", got, want)
	}
}

func TestResizePreservesActiveCursorHyperlink(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.osc("8;id=resize;http://example.com")

	vt.resize(10, 2)
	vt.update(testPrint("x"))

	cell := vt.activeScreen.cell(0, 0)
	if got, want := cell.Hyperlink, "http://example.com"; got != want {
		t.Fatalf("cell hyperlink = %q, want %q", got, want)
	}
	if got, want := cell.HyperlinkParams, "id=resize"; got != want {
		t.Fatalf("cell hyperlink params = %q, want %q", got, want)
	}
}

func TestResizeReflowPreservesHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(6, 3)
	vt.osc("8;id=reflow;http://example.com")
	printText(vt, "abcdef")

	vt.resize(3, 3)

	if got, want := vt.String(), "abc\ndef\n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for r := row(0); r < 2; r += 1 {
		for col := column(0); col < 3; col += 1 {
			cell := vt.activeScreen.cell(r, col)
			if got, want := cell.Hyperlink, "http://example.com"; got != want {
				t.Fatalf("cell %d,%d hyperlink = %q, want %q", r, col, got, want)
			}
			if got, want := cell.HyperlinkParams, "id=reflow"; got != want {
				t.Fatalf("cell %d,%d hyperlink params = %q, want %q", r, col, got, want)
			}
		}
	}
}

func TestSavedCursorRemapsThroughWiderReflow(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	printText(vt, "1A2B")
	vt.cursor.row = 1
	vt.cursor.col = 1
	vt.decsc()

	vt.resize(5, 3)
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "1A2X \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestSavedCursorPendingWrapClearsWhenReflowMovesFromRightEdge(t *testing.T) {
	vt := New()
	vt.resize(2, 3)
	printText(vt, "1A2B")
	if !vt.lastCol {
		t.Fatal("terminal did not enter pending wrap state")
	}
	vt.decsc()

	vt.resize(5, 3)
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "1A2BX\n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestRestoreCursorWithoutSaveAfterResizeUsesDefault(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")

	vt.resize(5, 5)
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "XABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestSavedCursorRemapsThroughNarrowerReflow(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "1ABCD")
	vt.cursor.row = 0
	vt.cursor.col = 4
	vt.decsc()

	vt.resize(3, 3)
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.decrc()
	vt.update(testPrint("X"))

	if got, want := vt.String(), "1AB\nCX \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
