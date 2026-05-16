package vaxis

import (
	"strings"
	"testing"
)

func newPrintTruncateTestWindow(cols int) Window {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
		charCache:  make(map[string]int),
	}
	vx.screenNext.resize(cols, 1)
	vx.screenLast.resize(cols, 1)
	return vx.Window()
}

func printTruncateLine(win Window) string {
	cols, _ := win.Size()
	var b strings.Builder
	for col := 0; col < cols; col++ {
		b.WriteString(win.Vx.screenNext.buf[0][col].Character.Grapheme)
	}
	return b.String()
}

func TestPrintTruncateDoesNotEllipsizeExactWidth(t *testing.T) {
	win := newPrintTruncateTestWindow(2)

	win.PrintTruncate(0, Segment{Text: "ab"})

	if got, want := printTruncateLine(win), "ab"; got != want {
		t.Fatalf("line = %q, want %q", got, want)
	}
}

func TestPrintTruncateEllipsizesOverflow(t *testing.T) {
	win := newPrintTruncateTestWindow(2)

	win.PrintTruncate(0, Segment{Text: "abc"})

	if got, want := printTruncateLine(win), "a…"; got != want {
		t.Fatalf("line = %q, want %q", got, want)
	}
}
