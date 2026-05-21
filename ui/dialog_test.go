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

func focusedDebugLabel(app *App) string {
	snapshot := app.DebugSnapshot()
	for _, target := range snapshot.Focusables {
		if target.Focused {
			return target.Label
		}
	}
	return ""
}
