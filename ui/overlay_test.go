package ui

import "testing"

func TestOverlayPaintsModalBarrierAndEntryAboveChild(t *testing.T) {
	theme := DefaultTheme()
	theme.Background = RGB(10, 10, 10)
	theme.Surface = RGB(50, 50, 50)
	app := NewApp(Overlay{
		Child: DecoratedBox(
			Decoration{Style: Style{Background: theme.Background}},
			SizedBox{Width: 10, Height: 5, Child: Text{Value: "base"}},
		),
		Entries: []OverlayEntry{{
			Modal: true,
			Child: DecoratedBox(
				Decoration{Style: Style{Background: theme.Surface}},
				SizedBox{Width: 3, Height: 1, Child: Text{Value: "top"}},
			),
		}},
	}, WithTheme(theme))
	app.Pump(Size{Width: 10, Height: 5})
	p := NewPainter(Size{Width: 10, Height: 5})
	app.Paint(p)

	if got := p.Cell(0, 0).Background; got == theme.Background {
		t.Fatalf("overlay did not paint a modal barrier over base background")
	}
	if got := p.Cell(3, 2).Grapheme + p.Cell(4, 2).Grapheme + p.Cell(5, 2).Grapheme; got != "top" {
		t.Fatalf("centered overlay text = %q, want top", got)
	}
	if got := p.Cell(3, 2).Background; got != theme.Surface {
		t.Fatalf("overlay entry background = %#v, want %#v", got, theme.Surface)
	}
}
