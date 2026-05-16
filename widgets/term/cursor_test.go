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
