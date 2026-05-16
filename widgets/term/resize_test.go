package term

import "testing"

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
