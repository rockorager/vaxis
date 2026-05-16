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

func TestNestedWindowSetCellClipsToParent(t *testing.T) {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
	}
	vx.screenNext.resize(4, 4)
	vx.screenLast.resize(4, 4)

	parent := vx.Window().New(1, 1, 2, 2)
	child := parent.New(1, 0, 2, 2)
	child.SetCell(1, 0, Cell{Character: Character{Grapheme: "x", Width: 1}})

	for row := range vx.screenNext.buf {
		for col := range vx.screenNext.buf[row] {
			if got := vx.screenNext.buf[row][col].Grapheme; got != "" {
				t.Fatalf("unexpected cell at %d,%d = %q", col, row, got)
			}
		}
	}
}

func TestNestedWindowSetCellWritesAbsolutePosition(t *testing.T) {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
	}
	vx.screenNext.resize(4, 4)
	vx.screenLast.resize(4, 4)

	parent := vx.Window().New(1, 1, 3, 3)
	child := parent.New(1, 1, 2, 2)
	child.SetCell(0, 0, Cell{Character: Character{Grapheme: "x", Width: 1}})

	if got, want := vx.screenNext.buf[2][2].Grapheme, "x"; got != want {
		t.Fatalf("nested cell = %q, want %q", got, want)
	}
}

func TestNestedWindowSetStyleWritesAbsolutePosition(t *testing.T) {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
	}
	vx.screenNext.resize(4, 4)
	vx.screenLast.resize(4, 4)
	vx.screenNext.buf[2][2] = Cell{Character: Character{Grapheme: "x", Width: 1}}

	parent := vx.Window().New(1, 1, 3, 3)
	child := parent.New(1, 1, 2, 2)
	child.SetStyle(0, 0, Style{Attribute: AttrBold})

	if got, want := vx.screenNext.buf[2][2].Attribute, AttrBold; got != want {
		t.Fatalf("nested style = %v, want %v", got, want)
	}
}

func TestNestedWindowSetStyleClipsToParent(t *testing.T) {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
	}
	vx.screenNext.resize(4, 4)
	vx.screenLast.resize(4, 4)
	vx.screenNext.buf[2][3] = Cell{Character: Character{Grapheme: "x", Width: 1}}

	parent := vx.Window().New(1, 1, 2, 2)
	child := parent.New(1, 0, 2, 2)
	child.SetStyle(1, 0, Style{Attribute: AttrBold})

	if got := vx.screenNext.buf[2][3].Attribute; got != 0 {
		t.Fatalf("clipped style changed to %v, want default", got)
	}
}
