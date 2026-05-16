package term

import "testing"

func TestLinefeedModePerformsCarriageReturn(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	vt.mode.lnm = true
	printText(vt, "123456")

	vt.lf()
	printText(vt, "X")

	if got, want := trimScreenString(vt.String()), "123456\nX"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCarriageReturnOriginModeMovesToLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.mode.decom = true
	vt.margin.left = 2
	vt.cursor.col = 0

	vt.cr()

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnLeftOfMarginMovesToZero(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.margin.left = 2
	vt.cursor.col = 1

	vt.cr()

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnRightOfMarginMovesToLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.margin.left = 2
	vt.cursor.col = 3

	vt.cr()

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCarriageReturnClearsPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "hello")

	vt.cr()

	if vt.lastCol {
		t.Fatal("carriage return did not clear pending wrap")
	}
}
