package ui_test

import (
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ui"
	"go.rockorager.dev/vaxis/ui/uitest"
)

func TestCheckboxPaintsCheckmark(t *testing.T) {
	app := uitest.New(ui.Checkbox{Checked: true, Label: "Enabled"})
	app.Pump(20, 1)
	if got := app.Cell(0, 0).Grapheme; got != "[" {
		t.Fatalf("checkbox left bracket = %q, want [", got)
	}
	if got := app.Cell(1, 0).Grapheme; got != "✓" {
		t.Fatalf("checkbox mark = %q, want checkmark", got)
	}
	if got := app.Cell(2, 0).Grapheme; got != "]" {
		t.Fatalf("checkbox right bracket = %q, want ]", got)
	}
	if got := app.Cell(4, 0).Grapheme; got != "E" {
		t.Fatalf("checkbox label = %q, want E", got)
	}
}

func TestCheckboxPaintsUnchecked(t *testing.T) {
	app := uitest.New(ui.Checkbox{Label: "Enabled"})
	app.Pump(20, 1)
	if got := app.Cell(1, 0).Grapheme; got != " " {
		t.Fatalf("unchecked checkbox mark = %q, want space", got)
	}
}

func TestCheckboxFocusUnderlinesMarkOnly(t *testing.T) {
	normal := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlack}
	theme := ui.DefaultTheme()
	theme.Foreground = normal.Foreground
	theme.Surface = normal.Background
	theme.SurfaceHovered = vaxis.ColorYellow
	app := ui.NewApp(ui.Checkbox{Checked: true, Label: "Enabled"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("focused checkbox mark underline = %#v, want single", got)
	}
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineOff {
		t.Fatalf("focused checkbox left bracket underline = %#v, want off", got)
	}
	if got := p.Cell(4, 0).Style; got != normal {
		t.Fatalf("focused checkbox label style = %#v, want %#v", got, normal)
	}
}

func TestCheckboxActivatesWithKeyboard(t *testing.T) {
	var values []bool
	app := ui.NewApp(ui.Checkbox{Checked: false, Label: "Enabled", OnChanged: func(ctx ui.EventContext, checked bool) {
		values = append(values, checked)
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Key{Keycode: vaxis.KeySpace})
	if len(values) != 2 || !values[0] || !values[1] {
		t.Fatalf("checkbox keyboard values = %v, want [true true]", values)
	}
}

func TestCheckboxActivateCanBeInvokedByShortcut(t *testing.T) {
	checked := false
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{"x": ui.ActivateIntent{}},
		Child: ui.Checkbox{OnChanged: func(ctx ui.EventContext, next bool) {
			checked = next
		}},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	if !checked {
		t.Fatal("expected shortcut to invoke checkbox default activate action")
	}
}

func TestCheckboxActivateCanBeOverridden(t *testing.T) {
	overridden := false
	changed := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.ActivateIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				overridden = true
				return ui.EventHandled
			},
		},
		Child: ui.Checkbox{OnChanged: func(ctx ui.EventContext, next bool) {
			changed = true
		}},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !overridden {
		t.Fatal("expected ancestor action to override checkbox activate")
	}
	if changed {
		t.Fatal("checkbox default activation ran despite ancestor override")
	}
}

func TestCheckboxIgnoresKeyRelease(t *testing.T) {
	called := false
	app := ui.NewApp(ui.Checkbox{OnChanged: func(ctx ui.EventContext, checked bool) {
		called = true
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
	if called {
		t.Fatal("checkbox should ignore key release")
	}
}

func TestCheckboxActivatesOnMouseClick(t *testing.T) {
	checked := false
	app := ui.NewApp(ui.Checkbox{OnChanged: func(ctx ui.EventContext, next bool) {
		checked = next
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !checked {
		t.Fatal("expected mouse click to activate checkbox")
	}
}

func TestCheckboxSetsMouseShapeOnHover(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Checkbox{Label: "Enabled"}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape = %q, want clickable", got)
	}
}

func TestCheckboxHoverStyleOnlyAppliesToBox(t *testing.T) {
	normal := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlack}
	hovered := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	theme := ui.DefaultTheme()
	theme.Foreground = normal.Foreground
	theme.Surface = normal.Background
	theme.Primary = normal.Background
	theme.PrimaryHovered = hovered.Background
	theme.SurfaceHovered = hovered.Background
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Checkbox{Label: "Enabled"}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got != hovered.Background {
		t.Fatalf("hovered checkbox box background = %#v, want %#v", got, hovered.Background)
	}
	if got := p.Cell(4, 0).Style; got != normal {
		t.Fatalf("hovered checkbox label style = %#v, want %#v", got, normal)
	}
}

func TestCheckedCheckboxHoverUsesPrimaryHovered(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.Foreground = ui.RGB(1, 1, 1)
	theme.PrimaryHovered = ui.RGB(2, 2, 2)
	theme.SurfaceHovered = ui.RGB(3, 3, 3)
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Checkbox{Checked: true, Label: "Enabled"}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got != theme.PrimaryHovered {
		t.Fatalf("checked hovered checkbox background = %#v, want primary hovered %#v", got, theme.PrimaryHovered)
	}
	if got := p.Cell(1, 0).Foreground; got != theme.Foreground {
		t.Fatalf("checked hovered checkbox foreground = %#v, want foreground %#v", got, theme.Foreground)
	}
}

func TestDisabledCheckboxUsesDisabledForeground(t *testing.T) {
	theme := ui.DefaultTheme()
	app := ui.NewApp(ui.Checkbox{Disabled: true, Label: "Disabled"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Foreground; got != theme.DisabledForeground {
		t.Fatalf("disabled checkbox mark foreground = %#v, want %#v", got, theme.DisabledForeground)
	}
	if got := p.Cell(4, 0).Foreground; got != theme.DisabledForeground {
		t.Fatalf("disabled checkbox label foreground = %#v, want %#v", got, theme.DisabledForeground)
	}
}

func TestDisabledCheckboxDoesNotActivate(t *testing.T) {
	called := false
	app := ui.NewApp(ui.Checkbox{Disabled: true, OnChanged: func(ctx ui.EventContext, checked bool) {
		called = true
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if called {
		t.Fatal("disabled checkbox should not activate")
	}
}

func TestDisabledCheckboxDoesNotTakeFocus(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Row(
		ui.Checkbox{Disabled: true, Label: "Disabled", OnChanged: func(ctx ui.EventContext, checked bool) {
			t.Fatal("disabled checkbox should not be focused")
		}},
		ui.Button{Label: "next", OnPressed: func(ctx ui.EventContext) { pressed = true }},
	))
	app.Pump(ui.Size{Width: 30, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !pressed {
		t.Fatal("expected focus to skip disabled checkbox")
	}
}

func TestDisabledCheckboxDoesNotSetMouseShape(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Checkbox{Disabled: true, Label: "Disabled"}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("disabled checkbox mouse shape = %q, want default", got)
	}
}
