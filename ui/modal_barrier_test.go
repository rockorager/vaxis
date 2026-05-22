package ui

import "testing"

func TestPainterScrimBlendsRGBColors(t *testing.T) {
	p := NewPainter(Size{Width: 1, Height: 1})
	p.DrawCell(Point{}, Cell{
		Character: Character{Grapheme: "x", Width: 1},
		Style:     Style{Foreground: RGB(200, 100, 50), Background: RGB(100, 50, 0)},
	})
	p.Scrim(Rect{Width: 1, Height: 1}, RGB(0, 0, 0), 128)
	cell := p.Cell(0, 0)
	if cell.Grapheme != "x" {
		t.Fatalf("scrim grapheme = %q, want x", cell.Grapheme)
	}
	if cell.Foreground != RGB(99, 49, 24) {
		t.Fatalf("scrim foreground = %#v, want blended RGB", cell.Foreground)
	}
	if cell.Background != RGB(49, 24, 0) {
		t.Fatalf("scrim background = %#v, want blended RGB", cell.Background)
	}
}

func TestModalBarrierScrimsBehindDialog(t *testing.T) {
	app := NewApp(Stack{Children: []Widget{
		DecoratedBox(Decoration{Style: Style{Foreground: RGB(200, 200, 200), Background: RGB(100, 100, 100)}}, Text{Value: "bg"}),
		ModalBarrier{Opacity: 128},
		Positioned{Left: 0, Top: 0, Child: Text{Value: "d", Style: Style{Foreground: RGB(255, 255, 255), Background: RGB(20, 20, 20)}}},
	}})
	app.Pump(Size{Width: 2, Height: 1})
	p := NewPainter(Size{Width: 2, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0); got.Grapheme != "g" || got.Background != RGB(49, 49, 49) {
		t.Fatalf("scrimmed background cell = %#v, want preserved/darkened g", got)
	}
	if got := p.Cell(0, 0); got.Grapheme != "d" || got.Background != RGB(20, 20, 20) {
		t.Fatalf("dialog cell = %#v, want unsrimmed dialog", got)
	}
}
