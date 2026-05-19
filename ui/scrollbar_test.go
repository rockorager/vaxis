package ui

import "testing"

func TestScrollbarPaintsVerticalThumbForOverflow(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	)}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 0).Grapheme; got != "█" {
		t.Fatalf("top thumb cell = %q, want block", got)
	}
	if got := p.Cell(9, 1).Grapheme; got != "│" {
		t.Fatalf("track cell = %q, want track", got)
	}
}

func TestScrollbarMovesThumbWithScrollOffset(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 3).Grapheme; got != "█" {
		t.Fatalf("bottom thumb cell = %q, want block", got)
	}
}

func TestScrollbarHidesWhenChildDoesNotOverflow(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines("one", "two")}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 0).Grapheme; got != "" {
		t.Fatalf("scrollbar cell without overflow = %q, want empty", got)
	}
}

func TestScrollbarThumbUsesViewportRatio(t *testing.T) {
	top, height := scrollbarThumb(ScrollMetrics{
		ScrollOffset:    2,
		MaxScrollOffset: 6,
		ViewportHeight:  4,
		ContentHeight:   10,
	})
	if top != 1 || height != 1 {
		t.Fatalf("thumb = top %d height %d, want top 1 height 1", top, height)
	}
}
