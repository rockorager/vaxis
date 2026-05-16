package term

import "testing"

func TestLeftRightMarginModeEnablesDECSLRM(t *testing.T) {
	vt := New()
	vt.resize(5, 3)

	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{2, 4}))

	if !vt.mode.declrmm {
		t.Fatal("left/right margin mode was not enabled")
	}
	if got, want := vt.margin.left, column(1); got != want {
		t.Fatalf("left margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(3); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}

func TestDECSLRMIgnoredWhenModeUnset(t *testing.T) {
	vt := New()
	vt.resize(5, 3)

	vt.update(testCSI('s', []uint32{2, 4}))

	if got, want := vt.margin.left, column(0); got != want {
		t.Fatalf("left margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(4); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
}

func TestAmbiguousCSISavesCursorWhenLeftRightMarginModeUnset(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.update(testCSI('s', nil))
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.update(testCSI('u', nil))

	if vt.cursor.row != 1 || vt.cursor.col != 2 {
		t.Fatalf("restored cursor = %d,%d, want 1,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestAmbiguousCSIResetsLeftRightMarginsWhenModeSet(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{2, 4}))

	vt.update(testCSI('s', nil))

	if got, want := vt.margin.left, column(0); got != want {
		t.Fatalf("left margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(4); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
}

func TestCursorPositionRelativeToOriginWithLeftRightMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 2
	vt.margin.bottom = 3
	vt.margin.left = 2
	vt.margin.right = 4
	vt.mode.decom = true

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testPrint("X"))

	if got, want := vt.String(), "     \n     \n  X  \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorPositionOriginModeClampsToMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 2
	vt.margin.bottom = 3
	vt.margin.left = 2
	vt.margin.right = 4
	vt.mode.decom = true

	vt.update(testCSI('H', []uint32{500, 500}))
	vt.update(testPrint("X"))

	if got, want := vt.String(), "     \n     \n     \n    X\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
