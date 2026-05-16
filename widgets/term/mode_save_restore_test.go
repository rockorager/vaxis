package term

import "testing"

func TestSaveRestorePrivateMode(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('s', []uint32{7}, '?'))
	vt.update(testCSI('l', []uint32{7}, '?'))
	if vt.mode.decawm {
		t.Fatal("wraparound mode was not reset")
	}

	vt.update(testCSI('r', []uint32{7}, '?'))
	if !vt.mode.decawm {
		t.Fatal("wraparound mode was not restored")
	}
}

func TestRestoreUnsavedPrivateModeUsesResetValue(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('r', []uint32{7}, '?'))

	if vt.mode.decawm {
		t.Fatal("unsaved wraparound mode restored as set")
	}
}

func TestRestorePrivateOriginModeAppliesSideEffects(t *testing.T) {
	vt := New()
	vt.resize(8, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 4
	vt.cursor.col = 6

	vt.update(testCSI('h', []uint32{6}, '?'))
	vt.update(testCSI('s', []uint32{6}, '?'))
	vt.update(testCSI('l', []uint32{6}, '?'))

	vt.cursor.row = 4
	vt.cursor.col = 6
	vt.update(testCSI('r', []uint32{6}, '?'))

	if !vt.mode.decom {
		t.Fatal("origin mode was not restored")
	}
	if vt.cursor.row != vt.margin.top || vt.cursor.col != vt.margin.left {
		t.Fatalf("cursor after origin restore = %d,%d, want %d,%d", vt.cursor.row, vt.cursor.col, vt.margin.top, vt.margin.left)
	}
}

func TestFullResetClearsSavedPrivateModes(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('s', []uint32{7}, '?'))
	vt.update(testESC('c'))
	vt.update(testCSI('r', []uint32{7}, '?'))

	if vt.mode.decawm {
		t.Fatal("full reset preserved saved wraparound mode")
	}
}
