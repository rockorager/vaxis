package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestRadioPaintsSelectedMark(t *testing.T) {
	app := uitest.New(ui.Radio[string]{Value: "a", GroupValue: "a", Label: "Alpha"})
	app.Pump(20, 1)
	if got := app.Cell(0, 0).Grapheme; got != "(" {
		t.Fatalf("radio left paren = %q, want (", got)
	}
	if got := app.Cell(1, 0).Grapheme; got != "•" {
		t.Fatalf("radio mark = %q, want bullet", got)
	}
	if got := app.Cell(2, 0).Grapheme; got != ")" {
		t.Fatalf("radio right paren = %q, want )", got)
	}
	if got := app.Cell(4, 0).Grapheme; got != "A" {
		t.Fatalf("radio label = %q, want A", got)
	}
}

func TestRadioPaintsUnselected(t *testing.T) {
	app := uitest.New(ui.Radio[string]{Value: "a", GroupValue: "b", Label: "Alpha"})
	app.Pump(20, 1)
	if got := app.Cell(1, 0).Grapheme; got != " " {
		t.Fatalf("unselected radio mark = %q, want space", got)
	}
}

func TestRadioFocusUnderlinesMarkOnly(t *testing.T) {
	normal := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlack}
	focused := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	theme := ui.DefaultTheme()
	theme.Button.Normal = normal
	theme.Button.Focused = focused
	app := ui.NewApp(ui.Radio[string]{Value: "a", GroupValue: "a", Label: "Alpha"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("focused radio mark underline = %#v, want single", got)
	}
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineOff {
		t.Fatalf("focused radio left paren underline = %#v, want off", got)
	}
	if got := p.Cell(4, 0).Style; got != normal {
		t.Fatalf("focused radio label style = %#v, want %#v", got, normal)
	}
}

func TestRadioActivatesWithKeyboard(t *testing.T) {
	var values []string
	app := ui.NewApp(ui.Radio[string]{Value: "a", GroupValue: "b", Label: "Alpha", OnChanged: func(ctx ui.EventContext, value string) {
		values = append(values, value)
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Key{Keycode: vaxis.KeySpace})
	if len(values) != 2 || values[0] != "a" || values[1] != "a" {
		t.Fatalf("radio keyboard values = %v, want [a a]", values)
	}
}

func TestRadioIgnoresKeyRelease(t *testing.T) {
	called := false
	app := ui.NewApp(ui.Radio[string]{Value: "a", OnChanged: func(ctx ui.EventContext, value string) {
		called = true
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
	if called {
		t.Fatal("radio should ignore key release")
	}
}

func TestRadioActivatesOnMouseClick(t *testing.T) {
	value := ""
	app := ui.NewApp(ui.Radio[string]{Value: "a", OnChanged: func(ctx ui.EventContext, next string) {
		value = next
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if value != "a" {
		t.Fatalf("radio value = %q, want a", value)
	}
}

func TestRadioSetsMouseShapeOnHover(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Radio[string]{Value: "a", Label: "Alpha"}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape = %q, want clickable", got)
	}
}

func TestRadioHoverStyleOnlyAppliesToBox(t *testing.T) {
	normal := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlack}
	hovered := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	theme := ui.DefaultTheme()
	theme.Button.Normal = normal
	theme.Button.Focused = normal
	theme.Button.Hovered = hovered
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Radio[string]{Value: "a", Label: "Alpha"}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got != hovered.Background {
		t.Fatalf("hovered radio box background = %#v, want %#v", got, hovered.Background)
	}
	if got := p.Cell(4, 0).Style; got != normal {
		t.Fatalf("hovered radio label style = %#v, want %#v", got, normal)
	}
}
