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

func TestDECSLRMWithParamsDoesNotSaveCursorWhenModeUnset(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.update(testCSI('s', []uint32{2, 4}))
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.update(testCSI('u', nil))

	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("restored cursor = %d,%d, want unchanged 0,0", vt.cursor.row, vt.cursor.col)
	}
}

func TestDisablingLeftRightMarginModeResetsMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{2, 4}))

	vt.update(testCSI('l', []uint32{69}, '?'))

	if vt.mode.declrmm {
		t.Fatal("left/right margin mode remained enabled")
	}
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

func TestDECSTBMClampsZeroAndOversizedMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.top = 1
	vt.margin.bottom = 1

	vt.update(testCSI('r', []uint32{0, 999}))

	if got, want := vt.margin.top, row(0); got != want {
		t.Fatalf("top margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.bottom, row(2); got != want {
		t.Fatalf("bottom margin = %d, want %d", got, want)
	}
}

func TestDECSTBMIgnoresTooManyParams(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.top = 1
	vt.margin.bottom = 2

	vt.update(testCSI('r', []uint32{1, 2, 3}))

	if got, want := vt.margin.top, row(1); got != want {
		t.Fatalf("top margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.bottom, row(2); got != want {
		t.Fatalf("bottom margin = %d, want %d", got, want)
	}
}

func TestDECSTBMHomesCursorRelativeToOriginMode(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.left = 2
	vt.margin.right = 4
	vt.mode.decom = true

	vt.update(testCSI('r', []uint32{3, 4}))

	if vt.cursor.row != 2 || vt.cursor.col != 2 {
		t.Fatalf("cursor = %d,%d, want 2,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestDECSLRMClampsZeroAndOversizedMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.margin.left = 1
	vt.margin.right = 1

	vt.update(testCSI('s', []uint32{0, 999}))

	if got, want := vt.margin.left, column(0); got != want {
		t.Fatalf("left margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(4); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
}

func TestDECSLRMIgnoresTooManyParams(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.margin.left = 1
	vt.margin.right = 3

	vt.update(testCSI('s', []uint32{1, 2, 3}))

	if got, want := vt.margin.left, column(1); got != want {
		t.Fatalf("left margin = %d, want %d", got, want)
	}
	if got, want := vt.margin.right, column(3); got != want {
		t.Fatalf("right margin = %d, want %d", got, want)
	}
}

func TestDECSLRMHomesCursorRelativeToOriginMode(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.mode.decom = true

	vt.update(testCSI('s', []uint32{3, 4}))

	if vt.cursor.row != 2 || vt.cursor.col != 2 {
		t.Fatalf("cursor = %d,%d, want 2,2", vt.cursor.row, vt.cursor.col)
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
