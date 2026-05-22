package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestRichTextPaintsInlineHyperlink(t *testing.T) {
	app := uitest.New(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", Style: ui.Style{Hyperlink: "https://vaxis.dev", HyperlinkParams: "id=docs"}},
		{Text: " now"},
	}})
	app.Pump(20, 1)
	if got := app.Cell(3, 0).Hyperlink; got != "" {
		t.Fatalf("pre-link hyperlink = %q, want empty", got)
	}
	if got := app.Cell(4, 0).Hyperlink; got != "https://vaxis.dev" {
		t.Fatalf("link hyperlink = %q, want https://vaxis.dev", got)
	}
	if got := app.Cell(4, 0).HyperlinkParams; got != "id=docs" {
		t.Fatalf("link params = %q, want id=docs", got)
	}
	if got := app.Cell(8, 0).Hyperlink; got != "" {
		t.Fatalf("post-link hyperlink = %q, want empty", got)
	}
}

func TestRichTextHyperlinkMergesWithTheme(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.Foreground = ui.RGB(12, 34, 56)
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "docs", Style: ui.Style{Hyperlink: "https://vaxis.dev"}},
	}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	cell := p.Cell(0, 0)
	if got := cell.Hyperlink; got != "https://vaxis.dev" {
		t.Fatalf("hyperlink = %q, want https://vaxis.dev", got)
	}
	if got := cell.Foreground; got != theme.Foreground {
		t.Fatalf("foreground = %#v, want theme foreground %#v", got, theme.Foreground)
	}
}

func TestRichTextSpanOnPressedActivatesOnMouseClick(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 1 {
		t.Fatalf("pressed = %d, want 1", pressed)
	}
}

func TestRichTextSpanOnPressedIgnoresOutsideSpan(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Send(vaxis.Mouse{Col: 8, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 0 {
		t.Fatalf("pressed = %d, want 0", pressed)
	}
}

func TestRichTextSpanOnPressedSetsMouseShape(t *testing.T) {
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape over link = %q, want clickable", got)
	}
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape outside link = %q, want default", got)
	}
}

func TestRichTextSpanOnPressedPaintsSingleUnderlineWhenUnfocused(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "before"},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
		}},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(10, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused interactive span underline = %#v, want single", got)
	}
}

func TestRichTextSpanOnPressedActivatesWithKeyboardFocus(t *testing.T) {
	var pressed []string
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "one", OnPressed: func(ctx ui.EventContext) { pressed = append(pressed, "one") }},
		{Text: " "},
		{Text: "two", OnPressed: func(ctx ui.EventContext) { pressed = append(pressed, "two") }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeySpace})
	if len(pressed) != 2 || pressed[0] != "one" || pressed[1] != "two" {
		t.Fatalf("pressed = %#v, want [one two]", pressed)
	}
}

func TestRichTextFocusedSpanIsPainted(t *testing.T) {
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "one", OnPressed: func(ctx ui.EventContext) {}},
		{Text: " "},
		{Text: "two", OnPressed: func(ctx ui.EventContext) {}},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused first span underline = %#v, want double", got)
	}
	if got := p.Cell(4, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused second span underline = %#v, want single", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused first span underline = %#v, want single", got)
	}
	if got := p.Cell(4, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused second span underline = %#v, want double", got)
	}
}

func TestRichTextFocusedSpanClearsWhenFocusLeaves(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
		}},
		ui.Button{Label: "next"},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused span underline = %#v, want double", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused span underline = %#v, want single", got)
	}
}

func TestTextOnPressedActivatesWithMouseAndKeyboard(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want 2", pressed)
	}
}

func TestTextOnPressedSetsMouseShape(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape over text link = %q, want clickable", got)
	}
	app.Send(vaxis.Mouse{Col: 8, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape outside text link = %q, want default", got)
	}
}

func TestTextOnPressedIsPaintedWhenFocused(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused text underline = %#v, want double", got)
	}
}

func TestTextOnPressedClearsFocusUnderlineWhenFocusLeaves(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}},
		ui.Button{Label: "next"},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused text underline = %#v, want double", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused text underline = %#v, want single", got)
	}
}

func TestTextOnPressedPaintsSingleUnderlineWhenUnfocused(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "before"},
		ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(10, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused interactive text underline = %#v, want single", got)
	}
}

func TestTextWithoutOnPressedDoesNotTakeFocus(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Row(
		ui.Text{Value: "plain"},
		ui.Button{Label: "next", OnPressed: func(ctx ui.EventContext) { pressed = true }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !pressed {
		t.Fatal("expected focus to skip plain text")
	}
}
