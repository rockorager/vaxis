package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestZeroWidthPrintAttachesToPreviousCell(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(testPrint("o"))
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "ò  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestZeroWidthPrintAtStartIsIgnored(t *testing.T) {
	vt := New()
	vt.resize(3, 1)

	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestZeroWidthPrintAttachesToPendingWrapCell(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	vt.update(testPrint("a"))
	vt.update(testPrint("b"))
	vt.update(testPrint("c"))
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.String(), "abc̀\n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("zero-width print cleared pending wrap")
	}
}

func TestZeroWidthPrintAttachesToWideCharacterHead(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.update(ansi.Print{Grapheme: "\u0300", Width: 0})

	if got, want := vt.activeScreen.cell(0, 0).Character.Grapheme, "橋̀"; got != want {
		t.Fatalf("wide cell grapheme = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 0).Character.Width, 2; got != want {
		t.Fatalf("wide cell width = %d, want %d", got, want)
	}
}
