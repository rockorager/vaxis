package ui

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDialogTrapsFocusTraversal(t *testing.T) {
	app := NewApp(Row(
		Button{Label: "Outside"},
		Dialog{
			Title: "Confirm",
			Child: Text{Value: "Continue?"},
			Actions: []Widget{
				Button{Label: "Cancel"},
				Button{Label: "OK"},
			},
		},
	))
	app.Pump(Size{Width: 40, Height: 8})
	if got := focusedDebugLabel(app); !strings.Contains(got, "Cancel") {
		t.Fatalf("focused label = %q, want Cancel", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(Size{Width: 40, Height: 8})
	if got := focusedDebugLabel(app); !strings.Contains(got, "OK") {
		t.Fatalf("focused label after Tab = %q, want OK", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(Size{Width: 40, Height: 8})
	if got := focusedDebugLabel(app); !strings.Contains(got, "Cancel") {
		t.Fatalf("focused label after wrapped Tab = %q, want Cancel", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift})
	app.Pump(Size{Width: 40, Height: 8})
	if got := focusedDebugLabel(app); !strings.Contains(got, "OK") {
		t.Fatalf("focused label after Shift+Tab = %q, want OK", got)
	}
}

func TestDialogEscapeDismisses(t *testing.T) {
	dismissed := false
	app := NewApp(Dialog{
		Title:     "Confirm",
		Actions:   []Widget{Button{Label: "OK"}},
		OnDismiss: func(EventContext) { dismissed = true },
	})
	app.Pump(Size{Width: 30, Height: 5})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	if !dismissed {
		t.Fatal("expected Escape to dismiss dialog")
	}
}

func TestDialogDismissCanBeOverridden(t *testing.T) {
	overridden := false
	dismissed := false
	app := NewApp(Actions{
		Bindings: map[IntentType]ActionFunc{
			DismissIntentType: func(ctx EventContext, intent Intent) EventResult {
				overridden = true
				return EventHandled
			},
		},
		Child: Dialog{
			Title:     "Confirm",
			Actions:   []Widget{Button{Label: "OK"}},
			OnDismiss: func(EventContext) { dismissed = true },
		},
	})
	app.Pump(Size{Width: 30, Height: 5})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	if !overridden {
		t.Fatal("expected ancestor action to override dialog dismiss")
	}
	if dismissed {
		t.Fatal("dialog default dismiss ran despite ancestor override")
	}
}

func TestDialogUsesSurfaceThemeForContent(t *testing.T) {
	theme := DefaultTheme()
	theme.Surface = RGB(10, 20, 30)
	theme.SurfaceRaised = RGB(30, 40, 50)
	theme.Foreground = RGB(220, 230, 240)
	theme.SurfaceHovered = RGB(50, 60, 70)
	theme.SurfacePressed = RGB(40, 50, 60)
	theme.Primary = RGB(90, 100, 110)
	theme.PrimaryHovered = RGB(100, 110, 120)

	app := NewApp(Dialog{
		Title: "Confirm",
		Child: Text{Value: "Continue?"},
		Actions: []Widget{
			Button{Label: "OK"},
		},
	}, WithTheme(theme))
	app.Pump(Size{Width: 30, Height: 7})
	p := NewPainter(Size{Width: 30, Height: 7})
	app.Paint(p)

	if got := p.Cell(0, 0).Style; got.Foreground != theme.Foreground || got.Background != theme.SurfaceRaised {
		t.Fatalf("dialog corner style = %#v, want raised surface foreground/background", got)
	}
	if got := p.Cell(1, 1).Style; got.Foreground != theme.Foreground || got.Background != theme.SurfaceRaised {
		t.Fatalf("dialog title style = %#v, want surface foreground/background", got)
	}
	if got := p.Cell(1, 3).Style; got.Foreground != theme.Foreground || got.Background != theme.SurfaceRaised {
		t.Fatalf("dialog body style = %#v, want surface foreground/background", got)
	}
	foundFocusedButton := false
	for _, cell := range p.Cells() {
		if cell.Background == theme.SurfacePressed {
			foundFocusedButton = true
			break
		}
	}
	if !foundFocusedButton {
		t.Fatalf("focused dialog button background not found, want nested surface hovered %#v", theme.SurfacePressed)
	}
}

func TestDialogFocusedButtonUsesSurfaceHovered(t *testing.T) {
	theme := DefaultTheme()
	theme.SurfacePressed = RGB(100, 110, 120)

	app := NewApp(Dialog{
		Title:   "Confirm",
		Actions: []Widget{Button{Label: "OK"}},
	}, WithTheme(theme))
	app.Pump(Size{Width: 30, Height: 5})
	foundHoveredButton := false
	for y := 0; y < 5 && !foundHoveredButton; y++ {
		for x := 0; x < 30 && !foundHoveredButton; x++ {
			app.Send(Mouse{Col: x, Row: y, EventType: EventMotion})
			app.Pump(Size{Width: 30, Height: 5})
			p := NewPainter(Size{Width: 30, Height: 5})
			app.Paint(p)
			for _, cell := range p.Cells() {
				if cell.Background == theme.SurfacePressed {
					foundHoveredButton = true
					break
				}
			}
		}
	}
	if !foundHoveredButton {
		t.Fatalf("hovered focused dialog button background not found, want nested surface hovered %#v", theme.SurfacePressed)
	}
}

func focusedDebugLabel(app *App) string {
	snapshot := app.DebugSnapshot()
	for _, target := range snapshot.Focusables {
		if target.Focused {
			return target.Label
		}
	}
	return ""
}
