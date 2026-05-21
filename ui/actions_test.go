package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

func TestShortcutsInvokeNearestActionFromFocusedTarget(t *testing.T) {
	intent := ui.Intent("test.invoke")
	called := ""
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.Intent]ui.ActionFunc{
			intent: func(ctx ui.EventContext) ui.EventResult {
				called = "global"
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{"x": intent},
			Child: ui.Row(
				ui.Actions{
					Bindings: map[ui.Intent]ui.ActionFunc{
						intent: func(ctx ui.EventContext) ui.EventResult {
							called = "local"
							return ui.EventHandled
						},
					},
					Child: ui.Button{Label: "local"},
				},
				ui.Button{Label: "global"},
			),
		},
	})
	app.Pump(ui.Size{Width: 40, Height: 1})

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	if called != "local" {
		t.Fatalf("action = %q, want local", called)
	}
}

func TestShortcutsIgnoreUnhandledIntent(t *testing.T) {
	intent := ui.Intent("test.unhandled")
	button := false
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{"Enter": intent},
		Child: ui.Button{
			Label:     "go",
			OnPressed: func(ctx ui.EventContext) { button = true },
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !button {
		t.Fatal("shortcut without action should leave event for focused button")
	}
}
