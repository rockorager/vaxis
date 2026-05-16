package term

import "testing"

func TestDECSCADECSELProtectsCells(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))
	vt.update(testPrint("d"))
	vt.update(testPrint("e"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}, '?'))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.cell(0, 1).protected {
		t.Fatal("protected cell lost protection")
	}
}

func TestDECProtectedCellsIgnoredByNormalEraseLine(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}))

	if got, want := vt.String(), "     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestISOProtectedCellsRespectedByNormalEraseLine(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testESC('V'))
	vt.update(testPrint("b"))
	vt.update(testESC('W'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestISOProtectedCellsRespectedByEraseCharacters(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testESC('V'))
	vt.update(testPrint("b"))
	vt.update(testESC('W'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('X', []uint32{3}))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDECSEDProtectsCells(t *testing.T) {
	vt := New()
	vt.resize(4, 2)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))
	vt.update(testPrint("d"))
	vt.update(testCSI('H', []uint32{2, 1}))
	vt.update(testPrint("e"))
	vt.update(testPrint("f"))
	vt.update(testPrint("g"))
	vt.update(testPrint("h"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('J', []uint32{0}, '?'))

	if got, want := vt.String(), " b  \n    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestSaveCursorRestoresProtectedPen(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('V'))
	vt.update(testESC('7'))
	vt.update(testESC('W'))
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	if !vt.activeScreen.cell(0, 0).protected {
		t.Fatal("restored cursor did not restore protected pen")
	}
}

func TestDECSCAIgnoresInvalidParameterLists(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testCSI('q', []uint32{0, 1}, '"'))
	vt.update(testPrint("x"))

	if !vt.activeScreen.cell(0, 0).protected {
		t.Fatal("multi-parameter DECSCA changed protected pen")
	}

	vt.update(testCSI('q', []uint32{3}, '"'))
	vt.update(testPrint("y"))

	if !vt.activeScreen.cell(0, 1).protected {
		t.Fatal("invalid DECSCA value changed protected pen")
	}
}
