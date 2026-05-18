package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type textAreaHarness struct{ value string }

func (h *textAreaHarness) Build(ui.BuildContext) ui.Widget {
	return ui.TextArea{
		Value:     h.value,
		SoftWrap:  true,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	}
}

func TestTextAreaEditsControlledMultilineValue(t *testing.T) {
	h := &textAreaHarness{}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 12, Height: 4})

	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})
	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 4})

	if h.value != "a\nb" {
		t.Fatalf("value = %q, want a\\nb", h.value)
	}
	p := ui.NewPainter(ui.Size{Width: 12, Height: 4})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "a" {
		t.Fatalf("first line = %q, want a", got)
	}
	if got := p.Cell(1, 1).Grapheme; got != "b" {
		t.Fatalf("second line = %q, want b", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 2,1", cursor, ok)
	}
}

func TestTextAreaPlaceholderUsesPlaceholderStyleWhenUnfocused(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.TextField.Placeholder = ui.Style{Foreground: vaxis.ColorGray, Background: vaxis.ColorBlack}
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "x"},
		ui.TextArea{Placeholder: "notes", MinWidth: 8, MinHeight: 2},
	), ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 3})
	app.Paint(p)
	if got := p.Cell(6, 0).Grapheme; got != "n" {
		t.Fatalf("placeholder = %q, want n", got)
	}
	if got := p.Cell(6, 0).Style; got != theme.TextField.Placeholder {
		t.Fatalf("placeholder style = %#v, want %#v", got, theme.TextField.Placeholder)
	}
}

func TestTextAreaMovesVisuallyThroughWrappedLines(t *testing.T) {
	h := &textAreaHarness{value: "abcdef"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 5, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight})
	app.Pump(ui.Size{Width: 5, Height: 4})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 5, Height: 4})

	p := ui.NewPainter(ui.Size{Width: 5, Height: 4})
	app.Paint(p)
	if got := p.Cell(1, 1).Grapheme; got != "d" {
		t.Fatalf("wrapped second visual line = %q, want d", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 2,1", cursor, ok)
	}
}

func TestTextAreaScrollsVerticallyToCursor(t *testing.T) {
	h := &textAreaHarness{value: "a\nb\nc"}
	app := ui.NewApp(ui.TextArea{
		Value:     h.value,
		MinHeight: 2,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDown})
	app.Pump(ui.Size{Width: 10, Height: 2})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "b" {
		t.Fatalf("top visible line = %q, want b", got)
	}
	if got := p.Cell(1, 1).Grapheme; got != "c" {
		t.Fatalf("bottom visible line = %q, want c", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 1 || cursor.Row != 1 {
		t.Fatalf("cursor = %#v ok=%v, want 1,1", cursor, ok)
	}
}

func TestTextAreaScrollsHorizontallyWithoutSoftWrap(t *testing.T) {
	h := &textAreaHarness{value: "abcdef"}
	app := ui.NewApp(ui.TextArea{
		Value:     h.value,
		MinWidth:  5,
		SoftWrap:  false,
		OnChanged: func(ctx ui.EventContext, value string) { h.value = value },
	})
	app.Pump(ui.Size{Width: 5, Height: 3})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 5, Height: 3})

	p := ui.NewPainter(ui.Size{Width: 5, Height: 3})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "e" {
		t.Fatalf("first visible scrolled cell = %q, want e", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "f" {
		t.Fatalf("second visible scrolled cell = %q, want f", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("cursor = %#v ok=%v, want 3,0", cursor, ok)
	}
}
