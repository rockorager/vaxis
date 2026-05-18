package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type textFieldHarness struct{ value string }

func (h *textFieldHarness) Build(ui.BuildContext) ui.Widget {
	return ui.TextField{Value: h.value, Placeholder: "name", OnChanged: func(ctx ui.EventContext, value string) { h.value = value }}
}

func TestTextFieldEditsControlledValue(t *testing.T) {
	h := &textFieldHarness{}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	if h.value != "ab" {
		t.Fatalf("value = %q, want ab", h.value)
	}
}

func TestTextFieldCursorMovementAndDelete(t *testing.T) {
	h := &textFieldHarness{value: "abc"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyBackspace})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	if h.value != "ac" {
		t.Fatalf("value = %q, want ac", h.value)
	}
}

func TestTextFieldPlaceholderAndCursorStyles(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.TextField.Placeholder = ui.Style{Foreground: vaxis.ColorGray, Background: vaxis.ColorBlack}
	theme.TextField.Cursor = ui.Style{Foreground: vaxis.ColorBlack, Background: vaxis.ColorWhite}
	app := ui.NewApp(ui.Row(ui.Button{Label: "x"}, ui.TextField{Placeholder: "name"}), ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(6, 0).Character.Grapheme; got != "n" {
		t.Fatalf("placeholder = %q, want n", got)
	}
	if got := p.Cell(6, 0).Style; got != theme.TextField.Placeholder {
		t.Fatalf("placeholder style = %#v, want %#v", got, theme.TextField.Placeholder)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(6, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("cursor style = %#v, want %#v", got, theme.TextField.Cursor)
	}
}

func TestTextFieldCursorDoesNotFillFocusedBackground(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.TextField.Focused = ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	theme.TextField.Cursor = ui.Style{Foreground: vaxis.ColorBlack, Background: vaxis.ColorWhite}
	app := ui.NewApp(ui.TextField{Value: "abc"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("cursor cell style = %#v, want cursor %#v", got, theme.TextField.Cursor)
	}
	if got := p.Cell(4, 0).Style.Background; got != theme.TextField.Focused.Background {
		t.Fatalf("field fill background = %#v, want focused background %#v", got, theme.TextField.Focused.Background)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style.Background; got != theme.TextField.Focused.Background {
		t.Fatalf("typed text background = %#v, want focused background %#v", got, theme.TextField.Focused.Background)
	}
	if got := p.Cell(4, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("end cursor cell style = %#v, want cursor %#v", got, theme.TextField.Cursor)
	}
}

func TestTextFieldKeepsEndCursorVisiblePastMinimumWidth(t *testing.T) {
	h := &textFieldHarness{value: "123456789"}
	theme := ui.DefaultTheme()
	theme.TextField.MinWidth = 3
	app := ui.NewApp(h, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("end cursor style = %#v, want cursor %#v", got, theme.TextField.Cursor)
	}
}

func TestTextFieldScrollsHorizontallyToKeepCursorVisible(t *testing.T) {
	h := &textFieldHarness{value: "abcdef"}
	theme := ui.DefaultTheme()
	theme.TextField.MinWidth = 5
	app := ui.NewApp(h, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Character.Grapheme; got != "…" {
		t.Fatalf("scrolled leading overflow = %q, want ellipsis", got)
	}
	if got := p.Cell(2, 0).Character.Grapheme; got != "f" {
		t.Fatalf("scrolled visible text = %q, want f", got)
	}
	if got := p.Cell(3, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("scrolled cursor style = %#v, want cursor %#v", got, theme.TextField.Cursor)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Character.Grapheme; got != "…" {
		t.Fatalf("left-scrolled leading overflow = %q, want ellipsis", got)
	}
	if got := p.Cell(2, 0).Character.Grapheme; got != "c" {
		t.Fatalf("left-scrolled cursor text = %q, want c", got)
	}
	if got := p.Cell(2, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("left-scrolled cursor style = %#v, want cursor %#v", got, theme.TextField.Cursor)
	}
	if got := p.Cell(3, 0).Character.Grapheme; got != "…" {
		t.Fatalf("left-scrolled trailing overflow = %q, want ellipsis", got)
	}
}

func TestTextFieldTrailingEllipsisStaysPinnedWhileCursorMoves(t *testing.T) {
	h := &textFieldHarness{value: "abcdef"}
	theme := ui.DefaultTheme()
	theme.TextField.MinWidth = 7
	app := ui.NewApp(h, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	for i := 0; i < 5; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	}
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(5, 0).Character.Grapheme; got != "…" {
		t.Fatalf("trailing ellipsis before cursor move = %q, want ellipsis", got)
	}

	for i := 0; i < 3; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyRight})
		app.Pump(ui.Size{Width: 10, Height: 1})
		p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
		app.Paint(p)
		if got := p.Cell(5, 0).Character.Grapheme; got != "…" {
			t.Fatalf("trailing ellipsis after cursor move %d = %q, want ellipsis pinned", i+1, got)
		}
	}
}

func TestTextFieldMouseShape(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.TextField{Value: "x"}})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeTextInput {
		t.Fatalf("mouse shape = %q, want text input", got)
	}
}

type controlledTextFieldApp struct{}

func (controlledTextFieldApp) CreateState() ui.State { return &controlledTextFieldState{} }

type controlledTextFieldState struct {
	ui.StateBase
	value string
}

func (s *controlledTextFieldState) Build(ui.BuildContext) ui.Widget {
	return ui.TextField{Value: s.value, OnChanged: func(ctx ui.EventContext, value string) {
		s.SetState(func() { s.value = value })
	}}
}

func TestTextFieldCursorAdvancesWithControlledSetState(t *testing.T) {
	theme := ui.DefaultTheme()
	app := ui.NewApp(controlledTextFieldApp{}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Character.Grapheme; got != "a" {
		t.Fatalf("after first key text cell = %q, want a", got)
	}
	if got := p.Cell(2, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("after first key cursor style = %#v at x=2, want %#v", got, theme.TextField.Cursor)
	}

	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Character.Grapheme; got != "a" {
		t.Fatalf("after second key first text cell = %q, want a", got)
	}
	if got := p.Cell(2, 0).Character.Grapheme; got != "b" {
		t.Fatalf("after second key second text cell = %q, want b", got)
	}
	if got := p.Cell(3, 0).Style; got != theme.TextField.Cursor {
		t.Fatalf("after second key cursor style = %#v at x=3, want %#v", got, theme.TextField.Cursor)
	}
}

func TestTextFieldRepeatedBackspaceBeforePumpUsesLatestEditState(t *testing.T) {
	h := &textFieldHarness{value: "abcdef"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	for i := 0; i < 6; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyBackspace})
	}
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 20, Height: 1})
	if h.value != "" {
		t.Fatalf("value after repeated backspace = %q, want empty", h.value)
	}
	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 20, Height: 1})
	if h.value != "x" {
		t.Fatalf("value after typing at cleared field = %q, want x", h.value)
	}
}

func TestTextFieldSubmitsCurrentValue(t *testing.T) {
	h := &textFieldHarness{value: "hello"}
	submitted := ""
	app := ui.NewApp(ui.TextField{Value: h.value, OnSubmitted: func(ctx ui.EventContext, value string) { submitted = value }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if submitted != "hello" {
		t.Fatalf("submitted = %q, want hello", submitted)
	}
}

func TestTextFieldObscuresDisplayedValue(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "secret", ObscureText: true})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	for i := 1; i <= 6; i++ {
		if got := p.Cell(i, 0).Character.Grapheme; got != "•" {
			t.Fatalf("obscured cell %d = %q, want bullet", i, got)
		}
	}
	if got := p.Cell(7, 0).Style; got != ui.DefaultTheme().TextField.Cursor {
		t.Fatalf("cursor after obscured text = %#v, want cursor", got)
	}
}
