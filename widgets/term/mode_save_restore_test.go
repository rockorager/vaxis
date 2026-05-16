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

func TestFullResetRestoresDefaultModes(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('l', []uint32{7, 1007, 1035, 1036}, '?'))
	vt.update(testCSI('l', []uint32{12}))
	vt.update(testCSI('h', []uint32{4}))
	vt.update(testESC('c'))

	if !vt.mode.decawm {
		t.Fatal("full reset did not restore wraparound mode")
	}
	if !vt.mode.srm {
		t.Fatal("full reset did not restore send/receive mode")
	}
	if !vt.mode.altScroll {
		t.Fatal("full reset did not restore alternate scroll mode")
	}
	if !vt.mode.ignoreKeypadWithNumLock {
		t.Fatal("full reset did not restore ignore-keypad mode")
	}
	if !vt.mode.altEscPrefix {
		t.Fatal("full reset did not restore alt-esc-prefix mode")
	}
	if vt.mode.irm {
		t.Fatal("full reset preserved insert mode")
	}
}

func TestFullResetRestoresMarginsAndViewport(t *testing.T) {
	vt := New()
	vt.resize(5, 4)
	vt.margin.top = 1
	vt.margin.bottom = 2
	vt.margin.left = 2
	vt.margin.right = 3
	vt.scrollOffset = 1

	vt.update(testESC('c'))

	if vt.margin.top != 0 || vt.margin.bottom != 3 || vt.margin.left != 0 || vt.margin.right != 4 {
		t.Fatalf("margins after reset = top:%d bottom:%d left:%d right:%d, want 0,3,0,4", vt.margin.top, vt.margin.bottom, vt.margin.left, vt.margin.right)
	}
	if vt.scrollOffset != 0 {
		t.Fatalf("scroll offset after reset = %d, want 0", vt.scrollOffset)
	}
}
