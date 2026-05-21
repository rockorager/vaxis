package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type testIntent struct {
	intentType ui.IntentType
	Value      string
}

func (i testIntent) IntentType() ui.IntentType {
	return i.intentType
}

func TestShortcutsInvokeNearestActionFromFocusedTarget(t *testing.T) {
	intent := testIntent{intentType: "test.invoke"}
	called := ""
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			intent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				called = "global"
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{"x": intent},
			Child: ui.Row(
				ui.Actions{
					Bindings: map[ui.IntentType]ui.ActionFunc{
						intent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
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

func TestActionsReceiveIntentPayload(t *testing.T) {
	intent := testIntent{intentType: "test.payload", Value: "open"}
	payload := ""
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{"x": intent},
		Child: ui.Actions{
			Bindings: map[ui.IntentType]ui.ActionFunc{
				intent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
					payload = intent.(testIntent).Value
					return ui.EventHandled
				},
			},
			Child: ui.Button{Label: "go"},
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	if payload != "open" {
		t.Fatalf("payload = %q, want open", payload)
	}
}

func TestShortcutsIgnoreUnhandledIntent(t *testing.T) {
	intent := testIntent{intentType: "test.unhandled"}
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

func TestShortcutCanInvokeButtonDefaultAction(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{"x": ui.ActivateIntent{}},
		Child: ui.Button{
			Label:     "go",
			OnPressed: func(ctx ui.EventContext) { pressed = true },
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	if !pressed {
		t.Fatal("expected shortcut to invoke button default activate action")
	}
}

func TestActionsOverrideButtonDefaultAction(t *testing.T) {
	override := false
	pressed := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.ActivateIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				override = true
				return ui.EventHandled
			},
		},
		Child: ui.Button{
			Label:     "go",
			OnPressed: func(ctx ui.EventContext) { pressed = true },
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !override {
		t.Fatal("expected ancestor action to override button default activate action")
	}
	if pressed {
		t.Fatal("button default action ran despite ancestor override")
	}
}

func TestDefaultFocusShortcutCanBeOverriddenByLocalAction(t *testing.T) {
	called := false
	pressed := 0
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.NextFocusIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				called = true
				return ui.EventHandled
			},
		},
		Child: ui.Row(
			ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = 1 }},
			ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
		),
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	if !called {
		t.Fatal("expected local next-focus action")
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 1 {
		t.Fatalf("pressed = %d, want first button to keep focus", pressed)
	}
}

func TestDefaultDismissShortcutCanBeOverriddenByLocalAction(t *testing.T) {
	called := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.DismissIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				called = true
				return ui.EventHandled
			},
		},
		Child: ui.Button{Label: "dismissible"},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyEsc})
	if !called {
		t.Fatal("expected local dismiss action")
	}
}

func TestWithShortcutsRemapsDefaultShortcut(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = 1 }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
	), ui.WithShortcuts(ui.ShortcutMap{
		"Ctrl+n": ui.NextFocusIntent{},
	}))
	app.Pump(ui.Size{Width: 20, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 1 {
		t.Fatalf("pressed after Tab = %d, want first button", pressed)
	}

	app.Send(vaxis.Key{Text: "n", Keycode: 'n', Modifiers: vaxis.ModCtrl})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 2 {
		t.Fatalf("pressed after Ctrl+n = %d, want second button", pressed)
	}
}

func TestWithShortcutsClonesBindings(t *testing.T) {
	shortcuts := ui.DefaultShortcuts()
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one"},
		ui.Button{Label: "two"},
	), ui.WithShortcuts(shortcuts))
	app.Pump(ui.Size{Width: 20, Height: 1})
	delete(shortcuts, "Tab")

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	snapshot := app.DebugSnapshot()
	if !debugSnapshotHasFocusedLabel(snapshot, "two") {
		t.Fatalf("focus did not move after mutating caller shortcuts: %#v", snapshot.Focusables)
	}
}

func debugSnapshotHasFocusedLabel(snapshot ui.DebugSnapshot, label string) bool {
	for _, target := range snapshot.Focusables {
		if target.Focused && target.Label == label {
			return true
		}
	}
	return false
}
