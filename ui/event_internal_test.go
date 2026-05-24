package ui

import (
	"math"
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestEventContextFractionalMousePointUsesPixelReports(t *testing.T) {
	app := NewApp(Text{Value: "x"})
	app.Send(Resize{Cols: 100, Rows: 40, XPixel: 1000, YPixel: 800})

	got := (EventContext{app: app}).FractionalMousePoint(Mouse{
		Col:    39,
		Row:    1,
		XPixel: 399,
		YPixel: 21,
	})
	want := FractionalMousePoint{Col: 39.9, Row: 1.05}
	if math.Abs(got.Col-want.Col) > 0.001 || math.Abs(got.Row-want.Row) > 0.001 {
		t.Fatalf("fractional point = %#v, want %#v", got, want)
	}
}

func TestEventContextFractionalMousePointFallsBackToCells(t *testing.T) {
	app := NewApp(Text{Value: "x"})
	got := (EventContext{app: app}).FractionalMousePoint(Mouse{Col: 3, Row: 4})
	want := FractionalMousePoint{Col: 3, Row: 4}
	if got != want {
		t.Fatalf("fractional point without pixels = %#v, want %#v", got, want)
	}
}

func TestAppPumpDropsStalePixelResize(t *testing.T) {
	app := NewApp(Text{Value: "x"})
	app.Send(Resize{Cols: 100, Rows: 40, XPixel: 1000, YPixel: 800})
	app.Pump(Size{Width: 80, Height: 20})

	got := (EventContext{app: app}).FractionalMousePoint(Mouse{
		Col:    3,
		Row:    4,
		XPixel: 399,
		YPixel: 21,
	})
	want := FractionalMousePoint{Col: 3, Row: 4}
	if got != want {
		t.Fatalf("fractional point with stale pixels = %#v, want %#v", got, want)
	}
}

func TestEventContextTogglesProfileOverlay(t *testing.T) {
	app := NewApp(Text{Value: "x"})
	ctx := EventContext{app: app}
	if ctx.ProfileOverlay() {
		t.Fatal("profile overlay should start hidden")
	}
	if !ctx.ToggleProfileOverlay() {
		t.Fatal("toggle should return visible state")
	}
	if !app.ProfileOverlay() {
		t.Fatal("expected context toggle to enable profile overlay")
	}
	if !app.FrameRequested() {
		t.Fatal("toggling profile overlay should request a frame")
	}

	ctx.SetProfileOverlay(false)
	if app.ProfileOverlay() {
		t.Fatal("expected SetProfileOverlay(false) to hide overlay")
	}
}

type staleFocusQuitIntent struct{}

func (staleFocusQuitIntent) IntentType() IntentType {
	return "test.quit"
}

func TestAppDispatchFallsBackToRootForDetachedFocus(t *testing.T) {
	quit := false
	app := NewApp(Actions{
		Bindings: map[IntentType]ActionFunc{
			staleFocusQuitIntent{}.IntentType(): func(EventContext, Intent) EventResult {
				quit = true
				return EventHandled
			},
		},
		Child: Shortcuts{
			Bindings: ShortcutMap{"q": staleFocusQuitIntent{}},
			Child:    Button{Label: "old"},
		},
	})
	app.Pump(Size{Width: 10, Height: 1})
	detached := app.focused.element
	app.UpdateRoot(Actions{
		Bindings: map[IntentType]ActionFunc{
			staleFocusQuitIntent{}.IntentType(): func(EventContext, Intent) EventResult {
				quit = true
				return EventHandled
			},
		},
		Child: Shortcuts{
			Bindings: ShortcutMap{"q": staleFocusQuitIntent{}},
			Child:    Text{Value: "new"},
		},
	})
	app.Pump(Size{Width: 10, Height: 1})
	app.focused = focusTarget{element: detached, index: elementFocusIndex}

	app.Send(vaxis.Key{Text: "q", Keycode: 'q'})
	if !quit {
		t.Fatal("expected root shortcut to run when focused element is detached")
	}
}
