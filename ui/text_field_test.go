package ui_test

import (
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type textFieldHarness struct {
	value    string
	minWidth int
}

func (h *textFieldHarness) Build(ui.BuildContext) ui.Widget {
	return ui.TextField{Value: h.value, Placeholder: "name", MinWidth: h.minWidth, OnChanged: func(ctx ui.EventContext, value string) { h.value = value }}
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

func TestTextFieldTextIntentCanBeInvokedByShortcut(t *testing.T) {
	value := ""
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{
			"Ctrl+d": ui.DeleteTextIntent{Direction: ui.TextDeleteForward, Unit: ui.TextMotionCharacter},
		},
		Child: ui.TextField{Value: "abc", OnChanged: func(ctx ui.EventContext, next string) {
			value = next
		}},
	})
	app.Pump(ui.Size{Width: 12, Height: 1})

	app.Send(vaxis.Key{Text: "d", Keycode: 'd', Modifiers: vaxis.ModCtrl})
	if value != "bc" {
		t.Fatalf("value = %q, want bc", value)
	}
}

func TestTextFieldTextIntentCanBeOverridden(t *testing.T) {
	overridden := false
	value := ""
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.InsertTextIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				overridden = true
				return ui.EventHandled
			},
		},
		Child: ui.TextField{OnChanged: func(ctx ui.EventContext, next string) {
			value = next
		}},
	})
	app.Pump(ui.Size{Width: 12, Height: 1})

	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	if !overridden {
		t.Fatal("expected ancestor action to override text insert")
	}
	if value != "" {
		t.Fatalf("value = %q, want no default insertion", value)
	}
}

func TestTextFieldWordMovementAndDelete(t *testing.T) {
	h := &textFieldHarness{value: "alpha beta, gamma"}
	app := ui.NewApp(ui.TextField{
		Value:    h.value,
		MinWidth: 30,
		OnChanged: func(ctx ui.EventContext, value string) {
			h.value = value
		},
	})
	app.Pump(ui.Size{Width: 30, Height: 1})

	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 30, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 30, Height: 1})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 6 || cursor.Row != 0 {
		t.Fatalf("cursor after ctrl-right = %#v ok=%v, want 6,0", cursor, ok)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 30, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 30, Height: 1})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 11 || cursor.Row != 0 {
		t.Fatalf("cursor after second ctrl-right = %#v ok=%v, want 11,0", cursor, ok)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyBackspace, Modifiers: vaxis.ModCtrl})
	app.UpdateRoot(ui.TextField{
		Value:    h.value,
		MinWidth: 30,
		OnChanged: func(ctx ui.EventContext, value string) {
			h.value = value
		},
	})
	app.Pump(ui.Size{Width: 30, Height: 1})
	if h.value != "alpha , gamma" {
		t.Fatalf("value after ctrl-backspace = %q, want alpha , gamma", h.value)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyHome})
	app.Pump(ui.Size{Width: 30, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyDelete, Modifiers: vaxis.ModCtrl})
	app.UpdateRoot(ui.TextField{
		Value:    h.value,
		MinWidth: 30,
		OnChanged: func(ctx ui.EventContext, value string) {
			h.value = value
		},
	})
	app.Pump(ui.Size{Width: 30, Height: 1})
	if h.value != " , gamma" {
		t.Fatalf("value after ctrl-delete = %q, want space comma space gamma", h.value)
	}
}

func TestTextFieldExtendsAndPaintsSelection(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "abcd", MinWidth: 10})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Pump(ui.Size{Width: 10, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().Selection
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("selected second cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("unselected third cell background = %#v, should not be selection background", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("cursor after extending selection = %#v ok=%v, want 3,0", cursor, ok)
	}
}

func TestTextFieldCopiesSelection(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.TextField{Value: "abcd", MinWidth: 10}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift}, now)
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab" {
		t.Fatalf("copies = %#v, want ab", backend.copies)
	}
}

func TestTextFieldMouseDragSelectsText(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "abcd", MinWidth: 10})
	app.Pump(ui.Size{Width: 10, Height: 1})

	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().Selection
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("selected second cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("unselected third cell background = %#v, should not be selection background", got)
	}
}

func TestTextFieldDoubleClickSelectsWord(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.TextField{Value: "alpha beta", MinWidth: 20}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	mouse := vaxis.Mouse{Col: 7, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventRelease
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventPress
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventRelease
	runner.HandleEvent(mouse, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "beta" {
		t.Fatalf("copies = %#v, want beta", backend.copies)
	}
}

func TestTextFieldTripleClickSelectsLine(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.TextField{Value: "alpha beta", MinWidth: 20}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	mouse := vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}
	for i := 0; i < 3; i++ {
		mouse.EventType = vaxis.EventPress
		runner.HandleEvent(mouse, now)
		mouse.EventType = vaxis.EventRelease
		runner.HandleEvent(mouse, now)
	}
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "alpha beta" {
		t.Fatalf("copies = %#v, want alpha beta", backend.copies)
	}
}

func TestTextFieldSelectionReplacementAndSingleLineInsert(t *testing.T) {
	h := &textFieldHarness{value: "abcd"}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Send(vaxis.Key{Keycode: vaxis.KeyRight, Modifiers: vaxis.ModShift})
	app.Send(vaxis.Key{Text: "x\ny", Keycode: 'x'})
	app.UpdateRoot(h)
	app.Pump(ui.Size{Width: 12, Height: 1})
	if h.value != "xycd" {
		t.Fatalf("value after replacing selection = %q, want xycd", h.value)
	}
}

func TestTextFieldPlaceholderAndHardwareCursor(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.MutedForeground = vaxis.ColorGray
	theme.Surface = vaxis.ColorBlack
	placeholder := ui.Style{Foreground: theme.MutedForeground, Background: theme.Surface}
	focusedPlaceholder := ui.Style{Foreground: theme.MutedForeground, Background: theme.SurfaceHovered}
	app := ui.NewApp(ui.Row(ui.Button{Label: "x"}, ui.TextField{Placeholder: "name"}), ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(6, 0).Grapheme; got != "n" {
		t.Fatalf("placeholder = %q, want n", got)
	}
	if got := p.Cell(6, 0).Style; got != placeholder {
		t.Fatalf("placeholder style = %#v, want %#v", got, placeholder)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(6, 0).Grapheme; got != "n" {
		t.Fatalf("focused placeholder = %q, want n", got)
	}
	if got := p.Cell(6, 0).Style; got != focusedPlaceholder {
		t.Fatalf("focused placeholder style = %#v, want %#v", got, focusedPlaceholder)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 6 || cursor.Row != 0 || cursor.Shape != ui.CursorBeam {
		t.Fatalf("cursor = %#v, ok = %v; want beam at 6,0", cursor, ok)
	}
}

func TestTextFieldHardwareCursorDoesNotChangeCellStyle(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.Foreground = vaxis.ColorWhite
	theme.SurfaceHovered = vaxis.ColorBlue
	focused := ui.Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered}
	app := ui.NewApp(ui.TextField{Value: "abc"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style; got != focused {
		t.Fatalf("cursor cell style = %#v, want focused %#v", got, focused)
	}
	if got := p.Cell(4, 0).Background; got != focused.Background {
		t.Fatalf("field fill background = %#v, want focused background %#v", got, focused.Background)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Background; got != focused.Background {
		t.Fatalf("typed text background = %#v, want focused background %#v", got, focused.Background)
	}
	if got := p.Cell(4, 0).Style; got != focused {
		t.Fatalf("end cursor cell style = %#v, want focused %#v", got, focused)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 4 || cursor.Row != 0 {
		t.Fatalf("end cursor = %#v, ok = %v; want at 4,0", cursor, ok)
	}
}

func TestTextFieldKeepsEndCursorVisiblePastMinimumWidth(t *testing.T) {
	h := &textFieldHarness{value: "123456789", minWidth: 3}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 1 || cursor.Row != 0 {
		t.Fatalf("end cursor = %#v, ok = %v; want at 1,0", cursor, ok)
	}
}

func TestTextFieldScrollsHorizontallyToKeepCursorVisible(t *testing.T) {
	h := &textFieldHarness{value: "abcdef", minWidth: 5}
	theme := ui.DefaultTheme()
	focused := ui.Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered}
	app := ui.NewApp(h, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "…" {
		t.Fatalf("scrolled leading overflow = %q, want ellipsis", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "f" {
		t.Fatalf("scrolled visible text = %q, want f", got)
	}
	if got := p.Cell(3, 0).Style; got != focused {
		t.Fatalf("scrolled cursor cell style = %#v, want focused %#v", got, focused)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("scrolled cursor = %#v, ok = %v; want at 3,0", cursor, ok)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "…" {
		t.Fatalf("left-scrolled leading overflow = %q, want ellipsis", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "c" {
		t.Fatalf("left-scrolled cursor text = %q, want c", got)
	}
	if got := p.Cell(2, 0).Style; got != focused {
		t.Fatalf("left-scrolled cursor cell style = %#v, want focused %#v", got, focused)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 0 {
		t.Fatalf("left-scrolled cursor = %#v, ok = %v; want at 2,0", cursor, ok)
	}
	if got := p.Cell(3, 0).Grapheme; got != "…" {
		t.Fatalf("left-scrolled trailing overflow = %q, want ellipsis", got)
	}
}

func TestTextFieldTrailingEllipsisStaysPinnedWhileCursorMoves(t *testing.T) {
	h := &textFieldHarness{value: "abcdef", minWidth: 7}
	app := ui.NewApp(h)
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	for i := 0; i < 5; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	}
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(5, 0).Grapheme; got != "…" {
		t.Fatalf("trailing ellipsis before cursor move = %q, want ellipsis", got)
	}

	for i := 0; i < 3; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyRight})
		app.Pump(ui.Size{Width: 10, Height: 1})
		p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
		app.Paint(p)
		if got := p.Cell(5, 0).Grapheme; got != "…" {
			t.Fatalf("trailing ellipsis after cursor move %d = %q, want ellipsis pinned", i+1, got)
		}
	}
}

func TestTextFieldControlledValueShrinkClampsCursor(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "abcdef", MinWidth: 12})
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.UpdateRoot(ui.TextField{Value: "x", MinWidth: 12})
	app.Pump(ui.Size{Width: 12, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 12, Height: 1})
	app.Paint(p)
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 0 {
		t.Fatalf("cursor after shrink = %#v ok=%v, want 2,0", cursor, ok)
	}
}

func TestTextFieldControlledValueShrinkClampsSelection(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "abcdef", MinWidth: 12})
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 12, Height: 1})
	app.UpdateRoot(ui.TextField{Value: "x", MinWidth: 12})
	app.Pump(ui.Size{Width: 12, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 12, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().Selection
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("clamped selected cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got == want {
		t.Fatalf("cell after clamped selection background = %#v, should not be selected", got)
	}
}

func TestTextFieldExternalReplacementResetsStaleScroll(t *testing.T) {
	app := ui.NewApp(ui.TextField{Value: "abcdef", MinWidth: 5})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnd})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.UpdateRoot(ui.TextField{Value: "x", MinWidth: 5})
	app.Pump(ui.Size{Width: 10, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "x" {
		t.Fatalf("visible replacement text = %q, want x", got)
	}
	if got := p.Cell(1, 0).Grapheme; got == "…" {
		t.Fatal("replacement should not keep stale leading ellipsis")
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 0 {
		t.Fatalf("replacement cursor = %#v ok=%v, want 2,0", cursor, ok)
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

func (controlledTextFieldApp) CreateState() ui.State {
	return &controlledTextFieldState{}
}

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
	if got := p.Cell(1, 0).Grapheme; got != "a" {
		t.Fatalf("after first key text cell = %q, want a", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 2 || cursor.Row != 0 {
		t.Fatalf("after first key cursor = %#v, ok = %v; want at 2,0", cursor, ok)
	}

	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "a" {
		t.Fatalf("after second key first text cell = %q, want a", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "b" {
		t.Fatalf("after second key second text cell = %q, want b", got)
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 3 || cursor.Row != 0 {
		t.Fatalf("after second key cursor = %#v, ok = %v; want at 3,0", cursor, ok)
	}
}

type controlledTextFieldAfterButtonApp struct {
	state *controlledTextFieldAfterButtonState
}

func (a *controlledTextFieldAfterButtonApp) CreateState() ui.State {
	a.state = &controlledTextFieldAfterButtonState{}
	return a.state
}

type controlledTextFieldAfterButtonState struct {
	ui.StateBase
	value string
}

func (s *controlledTextFieldAfterButtonState) Build(ui.BuildContext) ui.Widget {
	return ui.Column(
		ui.Button{Label: "before"},
		ui.TextField{Value: s.value, OnChanged: func(ctx ui.EventContext, value string) {
			s.SetState(func() { s.value = value })
		}},
	)
}

func TestTextFieldCursorAdvancesWithControlledSetStateAfterFocusChange(t *testing.T) {
	root := &controlledTextFieldAfterButtonApp{}
	app := ui.NewApp(root)
	app.Pump(ui.Size{Width: 20, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})

	app.Send(vaxis.Key{Text: "a", Keycode: 'a'})
	app.Pump(ui.Size{Width: 20, Height: 2})
	app.Send(vaxis.Key{Text: "b", Keycode: 'b'})
	app.Pump(ui.Size{Width: 20, Height: 2})
	app.Send(vaxis.Key{Keycode: vaxis.KeyLeft})
	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Pump(ui.Size{Width: 20, Height: 2})
	if root.state.value != "axb" {
		t.Fatalf("value = %q, want axb", root.state.value)
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

func TestTextFieldIgnoresKeyRelease(t *testing.T) {
	h := &textFieldHarness{}
	submitted := false
	app := ui.NewApp(ui.TextField{
		Value:       h.value,
		OnChanged:   func(ctx ui.EventContext, value string) { h.value = value },
		OnSubmitted: func(ctx ui.EventContext, value string) { submitted = true },
	})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Text: "x", Keycode: 'x', EventType: vaxis.EventRelease})
	app.UpdateRoot(ui.TextField{
		Value:       h.value,
		OnChanged:   func(ctx ui.EventContext, value string) { h.value = value },
		OnSubmitted: func(ctx ui.EventContext, value string) { submitted = true },
	})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
	if h.value != "" {
		t.Fatalf("value after key release = %q, want empty", h.value)
	}
	if submitted {
		t.Fatal("submit should ignore key release")
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
		if got := p.Cell(i, 0).Grapheme; got != "•" {
			t.Fatalf("obscured cell %d = %q, want bullet", i, got)
		}
	}
	if cursor, ok := p.Cursor(); !ok || cursor.Col != 7 || cursor.Row != 0 {
		t.Fatalf("cursor after obscured text = %#v, ok = %v; want at 7,0", cursor, ok)
	}
}
