package ui

import "testing"

func TestProgressBarPaintsDeterminateValue(t *testing.T) {
	filled := Style{Foreground: RGB(1, 2, 3)}
	empty := Style{Foreground: RGB(4, 5, 6)}
	app := NewApp(SizedBox{Width: 8, Height: 1, Child: ProgressBar{
		Value:       0.5,
		FilledStyle: filled,
		EmptyStyle:  empty,
	}})
	app.Pump(Size{Width: 8, Height: 1})

	p := NewPainter(Size{Width: 8, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "████" {
		t.Fatalf("bar = %q, want four filled cells", got)
	}
	if got := p.Cell(0, 0).Foreground; got != filled.Foreground {
		t.Fatalf("filled foreground = %#v, want %#v", got, filled.Foreground)
	}
	if got := p.Cell(4, 0).Foreground; got != empty.Foreground {
		t.Fatalf("empty foreground = %#v, want %#v", got, empty.Foreground)
	}
}

func TestProgressBarPaintsPartialCell(t *testing.T) {
	app := NewApp(SizedBox{Width: 8, Height: 1, Child: ProgressBar{Value: 0.5625}})
	app.Pump(Size{Width: 8, Height: 1})

	p := NewPainter(Size{Width: 8, Height: 1})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme; got != "▌" {
		t.Fatalf("partial cell = %q, want half block", got)
	}
}

func TestProgressBarClampsValue(t *testing.T) {
	app := NewApp(SizedBox{Width: 4, Height: 1, Child: ProgressBar{Value: 2}})
	app.Pump(Size{Width: 4, Height: 1})

	p := NewPainter(Size{Width: 4, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "████" {
		t.Fatalf("clamped high bar = %q, want full", got)
	}

	app.UpdateRoot(SizedBox{Width: 4, Height: 1, Child: ProgressBar{Value: -1}})
	app.Pump(Size{Width: 4, Height: 1})
	p = NewPainter(Size{Width: 4, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "" {
		t.Fatalf("clamped low bar = %q, want empty", got)
	}
}

func TestProgressBarUsesWidthWhenUnbounded(t *testing.T) {
	r := &renderProgressBar{Width: 7}
	r.Layout(LayoutContext{}, Constraints{MaxWidth: Unbounded, MaxHeight: Unbounded})
	if got := r.Size(); got != (Size{Width: 7, Height: 1}) {
		t.Fatalf("size = %#v, want width option", got)
	}
}

func TestProgressBarPaintsGradient(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 1, Child: ProgressBar{
		Value:         1,
		GradientStart: RGB(0, 0, 0),
		GradientEnd:   RGB(100, 50, 200),
	}})
	app.Pump(Size{Width: 3, Height: 1})

	p := NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Foreground; got != RGB(0, 0, 0) {
		t.Fatalf("start color = %#v, want gradient start", got)
	}
	if got := p.Cell(1, 0).Foreground; got != RGB(50, 25, 100) {
		t.Fatalf("mid color = %#v, want interpolated color", got)
	}
	if got := p.Cell(2, 0).Foreground; got != RGB(100, 50, 200) {
		t.Fatalf("end color = %#v, want gradient end", got)
	}
}

func TestProgressBarIgnoresNonRGBGradient(t *testing.T) {
	filled := Style{Foreground: RGB(1, 2, 3)}
	app := NewApp(SizedBox{Width: 2, Height: 1, Child: ProgressBar{
		Value:         1,
		FilledStyle:   filled,
		GradientStart: Color(1),
		GradientEnd:   RGB(100, 50, 200),
	}})
	app.Pump(Size{Width: 2, Height: 1})

	p := NewPainter(Size{Width: 2, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Foreground; got != filled.Foreground {
		t.Fatalf("fallback color = %#v, want filled style", got)
	}
}
