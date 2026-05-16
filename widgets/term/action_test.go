package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func testESC(final rune, intermediates ...rune) ansi.ESC {
	var seq ansi.ESC
	seq.Final = final
	seq.NumIntermediate = len(intermediates)
	copy(seq.Intermediate[:], intermediates)
	return seq
}

func testCSI(final rune, params []uint32, intermediates ...rune) ansi.CSI {
	var seq ansi.CSI
	seq.Final = final
	seq.NumParameters = len(params)
	copy(seq.Parameters[:], params)
	seq.NumIntermediate = len(intermediates)
	copy(seq.Intermediate[:], intermediates)
	return seq
}

func testPrint(s string) ansi.Print {
	return ansi.Print{Grapheme: s, Width: 1}
}

func TestLockingShiftInSelectsG0(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(ansi.C0(0x0E))
	vt.update(testPrint("q"))
	vt.update(ansi.C0(0x0F))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "─q  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("SI selected %v, want G0", vt.charsets.selected)
	}
}

func TestSingleShiftAppliesToOneGraphicCharacter(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testPrint("q"))
	vt.update(testESC('0', '*'))
	vt.update(testESC('N'))
	vt.update(testPrint("q"))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "q─q "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("single shift restored %v, want G0", vt.charsets.selected)
	}
	if vt.charsets.singleShift {
		t.Fatal("single shift remained active after printing one graphic character")
	}
}

func TestLockingShiftTwoAndThreeSelectGCharsets(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '*'))
	vt.update(testESC('0', '+'))
	vt.update(testESC('n'))
	vt.update(testPrint("q"))
	vt.update(testESC('o'))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "──  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g3 {
		t.Fatalf("LS3 selected %v, want G3", vt.charsets.selected)
	}
}

func TestCharsetBritishDesignation(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('A', '('))
	vt.update(testPrint("#"))

	if got, want := vt.String(), "£   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestTableCharsetPrintsNonASCIIAsSpace(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testPrint("😀"))

	if got, want := vt.String(), "    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorStyleIgnoresInvalidValues(t *testing.T) {
	vt := New()
	vt.cursor.style = vaxis.CursorBlock

	vt.update(testCSI('q', []uint32{5}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style = %d, want %d", got, want)
	}

	vt.update(testCSI('q', []uint32{9}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style after invalid value = %d, want %d", got, want)
	}

	vt.update(testCSI('q', []uint32{1, 2}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style after invalid parameter count = %d, want %d", got, want)
	}
}

func TestOriginModeMovesCursorToHome(t *testing.T) {
	vt := New()
	vt.resize(8, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 4
	vt.cursor.col = 6

	vt.update(testCSI('h', []uint32{6}, '?'))
	if !vt.mode.decom {
		t.Fatal("DEC origin mode was not enabled")
	}
	if vt.cursor.row != vt.margin.top || vt.cursor.col != vt.margin.left {
		t.Fatalf("cursor after DECSET 6 = %d,%d, want %d,%d", vt.cursor.row, vt.cursor.col, vt.margin.top, vt.margin.left)
	}

	vt.cursor.row = 4
	vt.cursor.col = 6
	vt.update(testCSI('l', []uint32{6}, '?'))
	if vt.mode.decom {
		t.Fatal("DEC origin mode was not disabled")
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor after DECRST 6 = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}
