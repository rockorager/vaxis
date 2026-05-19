package ui

import "testing"

func TestDividerPaintsHorizontalLine(t *testing.T) {
	app := NewApp(Divider{})
	app.Pump(Size{Width: 4, Height: 1})

	p := NewPainter(Size{Width: 4, Height: 1})
	app.Paint(p)
	for x := 0; x < 4; x++ {
		if got := p.Cell(x, 0).Grapheme; got != "─" {
			t.Fatalf("cell %d = %q, want horizontal divider", x, got)
		}
	}
}

func TestDividerPaintsVerticalLine(t *testing.T) {
	app := NewApp(Divider{Axis: Vertical})
	app.Pump(Size{Width: 1, Height: 3})

	p := NewPainter(Size{Width: 1, Height: 3})
	app.Paint(p)
	for y := 0; y < 3; y++ {
		if got := p.Cell(0, y).Grapheme; got != "│" {
			t.Fatalf("cell %d = %q, want vertical divider", y, got)
		}
	}
}

func TestDividerStyleAndCharacter(t *testing.T) {
	style := Style{Foreground: RGB(12, 34, 56)}
	app := NewApp(Divider{Character: Character{Grapheme: "━", Width: 1}, Style: style})
	app.Pump(Size{Width: 2, Height: 1})

	p := NewPainter(Size{Width: 2, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0); got.Grapheme != "━" || got.Style != style {
		t.Fatalf("divider cell = %#v, want custom character and style", got)
	}
}

func TestDividerDryLayout(t *testing.T) {
	h := (&Divider{Style: Style{Foreground: RGB(1, 2, 3)}}).CreateRenderObject(BuildContext{})
	if got := DryLayout(LayoutContext{}, h, Constraints{MaxWidth: 8, MaxHeight: 4}); got != (Size{Width: 8, Height: 1}) {
		t.Fatalf("horizontal dry layout = %#v, want 8x1", got)
	}
	v := (&Divider{Axis: Vertical, Style: Style{Foreground: RGB(1, 2, 3)}}).CreateRenderObject(BuildContext{})
	if got := DryLayout(LayoutContext{}, v, Constraints{MaxWidth: 8, MaxHeight: 4}); got != (Size{Width: 1, Height: 4}) {
		t.Fatalf("vertical dry layout = %#v, want 1x4", got)
	}
}
