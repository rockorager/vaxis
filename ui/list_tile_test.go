package ui

import "testing"

func TestListTilePaintsSlotsAndSubtitle(t *testing.T) {
	app := NewApp(SizedBox{Width: 20, Height: 2, Child: ListTile{
		Leading:  Text{Value: ">"},
		Title:    Text{Value: "Title"},
		Subtitle: Text{Value: "Subtitle"},
		Trailing: Text{Value: "!"},
	}})
	app.Pump(Size{Width: 20, Height: 2})

	p := NewPainter(Size{Width: 20, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != " > Title          !\n   Subtitle" {
		t.Fatalf("rendered tile = %q, want slots and subtitle", got)
	}
}

func TestListTileActivatesWithKeyboardAndMouse(t *testing.T) {
	activated := 0
	app := NewApp(SizedBox{Width: 12, Height: 1, Child: ListTile{
		Title: Text{Value: "Open"},
		OnPressed: func(EventContext) {
			activated++
		},
	}})
	app.Pump(Size{Width: 12, Height: 1})

	app.focusNext()
	app.Send(Key{Keycode: '\r'})
	app.Send(Mouse{Col: 11, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if activated != 2 {
		t.Fatalf("activated = %d, want keyboard and full-row mouse activation", activated)
	}
}

func TestListTileActivateCanBeInvokedByShortcut(t *testing.T) {
	activated := false
	app := NewApp(Shortcuts{
		Bindings: map[string]Intent{"x": ActivateIntent{}},
		Child: SizedBox{Width: 12, Height: 1, Child: ListTile{
			Title: Text{Value: "Open"},
			OnPressed: func(EventContext) {
				activated = true
			},
		}},
	})
	app.Pump(Size{Width: 12, Height: 1})
	app.focusNext()

	app.Send(Key{Text: "x", Keycode: 'x'})
	if !activated {
		t.Fatal("expected shortcut to invoke list tile default activate action")
	}
}

func TestListTileActivateCanBeOverridden(t *testing.T) {
	overridden := false
	activated := false
	app := NewApp(Actions{
		Bindings: map[IntentType]ActionFunc{
			ActivateIntentType: func(ctx EventContext, intent Intent) EventResult {
				overridden = true
				return EventHandled
			},
		},
		Child: SizedBox{Width: 12, Height: 1, Child: ListTile{
			Title: Text{Value: "Open"},
			OnPressed: func(EventContext) {
				activated = true
			},
		}},
	})
	app.Pump(Size{Width: 12, Height: 1})
	app.focusNext()

	app.Send(Key{Keycode: '\r'})
	if !overridden {
		t.Fatal("expected ancestor action to override list tile activate")
	}
	if activated {
		t.Fatal("list tile default activation ran despite ancestor override")
	}
}

func TestListTileUsesStateStyles(t *testing.T) {
	theme := DefaultTheme()
	theme.Foreground = RGB(1, 1, 1)
	theme.Surface = RGB(20, 20, 20)
	theme.SurfaceHovered = RGB(30, 30, 30)
	theme.Primary = RGB(40, 40, 40)
	theme.PrimaryHovered = RGB(60, 60, 60)

	app := NewApp(Column(
		SizedBox{Width: 10, Height: 1, Child: ListTile{
			Title:    Text{Value: "Tile"},
			Selected: true,
		}},
		SizedBox{Width: 10, Height: 1, Child: ListTile{
			Title: Text{Value: "Focus"},
			OnPressed: func(EventContext) {
			},
		}},
	), WithTheme(theme))
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Background; got != theme.Primary {
		t.Fatalf("selected background = %#v, want %#v", got, theme.Primary)
	}
	if got := p.Cell(1, 0).Foreground; got != theme.Foreground {
		t.Fatalf("selected text foreground = %#v, want %#v", got, theme.Foreground)
	}

	app.focusNext()
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 1).Background; got != theme.SurfaceHovered {
		t.Fatalf("focused background = %#v, want %#v", got, theme.SurfaceHovered)
	}
}

func TestListTileSelectedHoverUsesPrimaryHovered(t *testing.T) {
	theme := DefaultTheme()
	theme.Primary = RGB(40, 40, 40)
	theme.PrimaryHovered = RGB(60, 60, 60)

	app := NewApp(SizedBox{Width: 10, Height: 1, Child: ListTile{
		Title:    Text{Value: "Tile"},
		Selected: true,
		OnPressed: func(EventContext) {
		},
	}}, WithTheme(theme))
	app.Pump(Size{Width: 10, Height: 1})
	app.Send(Mouse{Col: 9, Row: 0, EventType: EventMotion})
	app.Pump(Size{Width: 10, Height: 1})

	p := NewPainter(Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Background; got != theme.PrimaryHovered {
		t.Fatalf("selected hovered background = %#v, want %#v", got, theme.PrimaryHovered)
	}
}

func TestListTileHoverStyle(t *testing.T) {
	theme := DefaultTheme()
	theme.SurfaceHovered = RGB(30, 30, 30)

	app := NewApp(SizedBox{Width: 10, Height: 1, Child: ListTile{
		Title: Text{Value: "Tile"},
		OnPressed: func(EventContext) {
		},
	}}, WithTheme(theme))
	app.Pump(Size{Width: 10, Height: 1})

	app.Send(Mouse{Col: 9, Row: 0, EventType: EventMotion})
	app.Pump(Size{Width: 10, Height: 1})

	p := NewPainter(Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Background; got != theme.SurfaceHovered {
		t.Fatalf("hovered background = %#v, want %#v", got, theme.SurfaceHovered)
	}
}

func TestListTileDisabledDoesNotFocusOrActivate(t *testing.T) {
	activated := false
	buttonActivated := false
	app := NewApp(Column(
		ListTile{
			Title:    Text{Value: "Disabled"},
			Disabled: true,
			OnPressed: func(EventContext) {
				activated = true
			},
		},
		Button{Label: "Next", OnPressed: func(EventContext) {
			buttonActivated = true
		}},
	))
	app.Pump(Size{Width: 20, Height: 2})

	app.focusNext()
	app.Send(Mouse{Col: 0, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	app.Send(Key{Keycode: '\r'})
	if activated {
		t.Fatal("disabled tile activated")
	}
	if !buttonActivated {
		t.Fatal("focus did not skip disabled tile to activate next button")
	}
}
