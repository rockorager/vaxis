package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestREPRepeatsPreviousGraphicAfterCursorMovement(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("A"))
	vt.update(testCSI('G', []uint32{5}))
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "A   A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestREPWorksAtColumnZero(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	vt.update(testPrint("A"))
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.update(testCSI('b', []uint32{2}))

	if got, want := vt.String(), "A  \nAA "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestREPUsesCurrentCursorStyle(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("A"))
	vt.cursor.Foreground = vaxis.IndexColor(1)
	vt.update(testCSI('b', []uint32{1}))

	cell := vt.activeScreen.cell(0, 1)
	if got, want := cell.Grapheme, "A"; got != want {
		t.Fatalf("repeated grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Foreground, vaxis.IndexColor(1); got != want {
		t.Fatalf("repeated foreground = %v, want %v", got, want)
	}
}

func TestREPDefaultsToOne(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("A"))
	vt.update(testCSI('b', nil))
	vt.update(testCSI('b', []uint32{0}))

	if got, want := vt.String(), "AAA"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestREPWrapsFromPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	printText(vt, "ABC")
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "ABC\nC  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("REP did not preserve source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("REP did not mark destination row continuation")
	}
}

func TestREPIgnoresMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("A"))
	vt.update(testCSI('b', []uint32{2, 1}))

	if got, want := vt.String(), "A  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestRISClearsREPPreviousGraphic(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("A"))
	vt.update(testESC('c'))
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.hasPreviousChar {
		t.Fatal("RIS did not clear REP previous graphic")
	}
}

func TestREPDoesNotTrackCombiningMarks(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testPrint("A"))
	vt.update(ansi.Print{Grapheme: "\u0301", Width: 0})
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "ÁA  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestREPRepeatsWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(6, 1)

	vt.update(ansi.Print{Grapheme: "界", Width: 2})
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.activeScreen.cell(0, 0).Character, (vaxis.Character{Grapheme: "界", Width: 2}); got != want {
		t.Fatalf("first cell = %#v, want %#v", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 2).Character, (vaxis.Character{Grapheme: "界", Width: 2}); got != want {
		t.Fatalf("repeated cell = %#v, want %#v", got, want)
	}
	if got := vt.activeScreen.cell(0, 1).Width; got != 0 {
		t.Fatalf("first wide tail width = %d, want 0", got)
	}
	if got := vt.activeScreen.cell(0, 3).Width; got != 0 {
		t.Fatalf("repeated wide tail width = %d, want 0", got)
	}
}

func TestREPRepeatsParsedGraphemeCluster(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.graphemeCluster = true

	vt.update(ansi.Print{Grapheme: "A\u0301", Width: 1})
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "ÁÁ   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestREPRepeatsCharsetMappedCharacter(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testPrint("`"))
	vt.update(testCSI('b', []uint32{1}))

	if got, want := vt.String(), "◆◆  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
