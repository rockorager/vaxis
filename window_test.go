package vaxis

import (
	"strings"
	"testing"
)

func newPrintTruncateTestWindow(cols int) Window {
	return newWindowTestWindow(cols, 1)
}

func newWindowTestWindow(cols int, rows int) Window {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
		charCache:  make(map[string]int),
	}
	vx.screenNext.resize(cols, rows)
	vx.screenLast.resize(cols, rows)
	return vx.Window()
}

func printTruncateLine(win Window) string {
	cols, _ := win.Size()
	var b strings.Builder
	for col := 0; col < cols; col++ {
		b.WriteString(win.Vx.screenNext.cell(col, 0).Character.Grapheme)
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

func TestWrapBreaksLongSegmentAtGrapheme(t *testing.T) {
	win := newWindowTestWindow(5, 2)

	col, row := win.Wrap(Segment{Text: "abcdefg"})

	if col != 2 || row != 1 {
		t.Fatalf("cursor = %d,%d, want 2,1", col, row)
	}
	if got, want := win.Vx.screenNext.cell(4, 0).Grapheme, "e"; got != want {
		t.Fatalf("last first-row cell = %q, want %q", got, want)
	}
	if got, want := win.Vx.screenNext.cell(1, 1).Grapheme, "g"; got != want {
		t.Fatalf("second-row cell = %q, want %q", got, want)
	}
}

func TestWrapDoesNotDropBufferedOverflow(t *testing.T) {
	const width = 300
	win := newWindowTestWindow(width, 1)

	col, row := win.Wrap(Segment{Text: strings.Repeat("a", 260)})

	if col != 260 || row != 0 {
		t.Fatalf("cursor = %d,%d, want 260,0", col, row)
	}
	if got, want := win.Vx.screenNext.cell(259, 0).Grapheme, "a"; got != want {
		t.Fatalf("last written cell = %q, want %q", got, want)
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

	for row := 0; row < vx.screenNext.rows; row += 1 {
		for col := 0; col < vx.screenNext.cols; col += 1 {
			if got := vx.screenNext.cell(col, row).Grapheme; got != "" {
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

	if got, want := vx.screenNext.cell(2, 2).Grapheme, "x"; got != want {
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
	vx.screenNext.setCellDirect(2, 2, Cell{Character: Character{Grapheme: "x", Width: 1}})

	parent := vx.Window().New(1, 1, 3, 3)
	child := parent.New(1, 1, 2, 2)
	child.SetStyle(0, 0, Style{Attribute: AttrBold})

	if got, want := vx.screenNext.cell(2, 2).Attribute, AttrBold; got != want {
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
	vx.screenNext.setCellDirect(3, 2, Cell{Character: Character{Grapheme: "x", Width: 1}})

	parent := vx.Window().New(1, 1, 2, 2)
	child := parent.New(1, 0, 2, 2)
	child.SetStyle(1, 0, Style{Attribute: AttrBold})

	if got := vx.screenNext.cell(3, 2).Attribute; got != 0 {
		t.Fatalf("clipped style changed to %v, want default", got)
	}
}
