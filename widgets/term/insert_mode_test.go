package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
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
