package term

import "testing"

func TestCursorForwardOutsideRightMarginStaysOutside(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.right = 2
	vt.cursor.col = 4

	vt.cuf(100)

	if got, want := vt.cursor.col, column(4); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorForwardInsideRightMarginClampsToMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.right = 2

	vt.cuf(100)

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorBackwardIgnoresLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.left = 2
	vt.cursor.col = 3

	vt.cub(100)

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorDownOutsideBottomMarginStaysOutside(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.bottom = 2
	vt.cursor.row = 4

	vt.cud(100)

	if got, want := vt.cursor.row, row(4); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestCursorDownInsideBottomMarginClampsToMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.bottom = 2

	vt.cud(100)

	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestBackspaceDoesNotReverseWrapByDefault(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.cursor.row = 1

	vt.bs()

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorBackwardPendingWrapWithoutReverseWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)

	printText(vt, "ABCDE")
	vt.cub(1)
	printText(vt, "X")

	if got, want := vt.String(), "ABCXE\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardPendingWrapWithReverseWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE")
	vt.cub(1)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDX\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardReverseWrapUsesSoftWrappedRows(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE1")
	vt.cub(2)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDX\n1    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardReverseWrapStopsAtUnwrappedRow(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE")
	vt.cr()
	vt.lf()
	printText(vt, "1")
	vt.cub(2)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDE\nX    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
