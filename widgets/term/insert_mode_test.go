package term

import (
	"testing"

	"go.rockorager.dev/vaxis/ansi"
)

func TestInsertModeInsertsBeforeCursor(t *testing.T) {
	vt := New()
	vt.resize(10, 2)
	printText(vt, "hello")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := trimScreenString(vt.String()), "hXello"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeDoesNotWrapPushedCharacters(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "hello")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := trimScreenString(vt.String()), "hXell"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeAtEndWrapsPendingCursorBeforePrinting(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "hello")
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := trimScreenString(vt.String()), "hello\nX"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeWithWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "hello")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.mode.irm = true

	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	if got, want := trimScreenString(vt.String()), "h😀 el"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeWithWideCharacterAtEndWrapsBeforePrinting(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "well")
	vt.mode.irm = true

	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	if got, want := trimScreenString(vt.String()), "well\n😀"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 1 || vt.cursor.col != 2 {
		t.Fatalf("cursor = %d,%d, want 1,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestInsertModePushingOffWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "123")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	vt.update(testCSI('H', []uint32{1, 1}))
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := trimScreenString(vt.String()), "X123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeAtWideCharacterHead(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 1}))
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := vt.String(), "X  12"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertModeAtWideCharacterTail(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.mode.irm = true

	vt.update(testPrint("X"))

	if got, want := vt.String(), " X 12"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
