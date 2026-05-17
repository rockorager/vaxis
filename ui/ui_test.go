package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestTextPaints(t *testing.T) {
	app := uitest.New(ui.Text("hello"))
	app.Pump(10, 1)
	if got := app.Text(); !strings.HasPrefix(got, "hello") {
		t.Fatalf("text = %q, want prefix hello", got)
	}
}

func TestCenterPaintsChildCentered(t *testing.T) {
	app := uitest.New(ui.Center(ui.Text("hi")))
	app.Pump(6, 3)
	if got := app.Cell(2, 1).Character.Grapheme; got != "h" {
		t.Fatalf("centered cell = %q, want h", got)
	}
}

type counter struct{ initial int }

func (c counter) CreateState() ui.State { return &counterState{count: c.initial} }

type counterState struct {
	ui.StateBase
	count int
}

func (s *counterState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Text(string(rune('0' + s.count)))
}

func TestStatePreservedAcrossCompatibleUpdate(t *testing.T) {
	app := ui.NewApp(counter{initial: 1})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(counter{initial: 9})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Character.Grapheme; got != "1" {
		t.Fatalf("state text = %q, want preserved 1", got)
	}
}

type keyedCounter struct {
	key     ui.KeyValue
	initial int
}

func (c keyedCounter) WidgetKey() ui.KeyValue { return c.key }
func (c keyedCounter) CreateState() ui.State  { return &counterState{count: c.initial} }

func TestStateRecreatedWhenKeyChanges(t *testing.T) {
	app := ui.NewApp(keyedCounter{key: "a", initial: 1})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(keyedCounter{key: "b", initial: 9})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Character.Grapheme; got != "9" {
		t.Fatalf("state text = %q, want recreated 9", got)
	}
}

type multiKind struct{}

func (multiKind) Build(ctx ui.BuildContext) ui.Widget { return ui.Text("bad") }
func (multiKind) CreateState() ui.State               { return &counterState{} }

func TestInvalidWidgetPanics(t *testing.T) {
	tests := []struct {
		name string
		root ui.Widget
	}{
		{name: "not widget", root: 42},
		{name: "multiple kinds", root: multiKind{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			_ = ui.NewApp(tt.root)
		})
	}
}

type providerText struct{}

func (providerText) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Text(ui.MustDepend[string](ctx))
}

func TestProviderNotifiesDependents(t *testing.T) {
	root := ui.Provider[string]{Value: "a", ChildWidget: providerText{}}
	app := ui.NewApp(root)
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(ui.Provider[string]{Value: "b", ChildWidget: providerText{}})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Character.Grapheme; got != "b" {
		t.Fatalf("provider text = %q, want b", got)
	}
}

func TestProviderShouldNotifyCanSuppressDependentRebuild(t *testing.T) {
	called := false
	root := ui.Provider[string]{Value: "a", ChildWidget: providerText{}, ShouldNotify: func(old, next string) bool {
		called = true
		return false
	}}
	app := ui.NewApp(root)
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(ui.Provider[string]{Value: "b", ChildWidget: providerText{}})
	app.Pump(ui.Size{Width: 1, Height: 1})
	if !called {
		t.Fatal("expected ShouldNotify to be called")
	}
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Character.Grapheme; got != "a" {
		t.Fatalf("provider text = %q, want stale a", got)
	}
}

func TestPaddingOffsetsChild(t *testing.T) {
	app := uitest.New(ui.Padding(ui.All(1), ui.Text("x")))
	app.Pump(3, 3)
	if got := app.Cell(1, 1).Character.Grapheme; got != "x" {
		t.Fatalf("padded cell = %q, want x", got)
	}
}

func TestRowColumnAndExpandedPaintInExpectedPositions(t *testing.T) {
	tests := []struct {
		name   string
		root   ui.Widget
		w, h   int
		checks map[ui.Point]string
	}{
		{
			name:   "row",
			root:   ui.Row(ui.Text("a"), ui.Expanded(ui.Text("b")), ui.Text("c")),
			w:      5,
			h:      1,
			checks: map[ui.Point]string{{X: 0, Y: 0}: "a", {X: 1, Y: 0}: "b", {X: 4, Y: 0}: "c"},
		},
		{
			name:   "column",
			root:   ui.Column(ui.Text("a"), ui.Expanded(ui.Text("b")), ui.Text("c")),
			w:      1,
			h:      5,
			checks: map[ui.Point]string{{X: 0, Y: 0}: "a", {X: 0, Y: 1}: "b", {X: 0, Y: 4}: "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := uitest.New(tt.root)
			app.Pump(tt.w, tt.h)
			for pt, want := range tt.checks {
				if got := app.Cell(pt.X, pt.Y).Character.Grapheme; got != want {
					t.Fatalf("cell %v = %q, want %q", pt, got, want)
				}
			}
		})
	}
}

type updatingCounter struct{}

func (u updatingCounter) CreateState() ui.State { return &updatingCounterState{} }

type updatingCounterState struct {
	ui.StateBase
	updates int
}

func (s *updatingCounterState) DidUpdateWidget(old ui.Widget) { s.updates++ }
func (s *updatingCounterState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Text(fmt.Sprint(s.updates))
}

func TestDidUpdateWidgetCalledOnCompatibleUpdate(t *testing.T) {
	app := ui.NewApp(updatingCounter{})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(updatingCounter{})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Character.Grapheme; got != "1" {
		t.Fatalf("updates = %q, want 1", got)
	}
}

func TestButtonActivatesFocusedButton(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button("one", func(ctx ui.EventContext) { pressed += 1 }),
		ui.Button("two", func(ctx ui.EventContext) { pressed += 10 }),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 1 {
		t.Fatalf("pressed = %d, want first button", pressed)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 11 {
		t.Fatalf("pressed = %d, want second button after tab", pressed)
	}
}

func TestFocusNodeRequestFocus(t *testing.T) {
	var n1, n2 ui.FocusNode
	app := ui.NewApp(ui.Row(
		ui.Focus(&n1, ui.Text("one")),
		ui.Focus(&n2, ui.Text("two")),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	n2.RequestFocus()
	if !n2.HasFocus() || n1.HasFocus() {
		t.Fatal("expected second focus node to have focus")
	}
}

func TestKeymapCaptureHandlesBeforeFocusedButton(t *testing.T) {
	called := false
	button := false
	app := ui.NewApp(ui.Keymap(map[string]ui.VoidCallback{
		"Enter": func(ctx ui.EventContext) { called = true },
	}, ui.Button("button", func(ctx ui.EventContext) { button = true })))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !called {
		t.Fatal("expected keymap callback")
	}
	if button {
		t.Fatal("button should not run after capture keymap handled event")
	}
}

func TestKeymapIgnoredEventContinuesToFocusedButton(t *testing.T) {
	button := false
	app := ui.NewApp(ui.Keymap(map[string]ui.VoidCallback{
		"Ctrl+x": func(ctx ui.EventContext) {},
	}, ui.Button("button", func(ctx ui.EventContext) { button = true })))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !button {
		t.Fatal("expected button to receive event after keymap ignored it")
	}
}

func TestShiftTabMovesFocusPrevious(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button("one", func(ctx ui.EventContext) { pressed = 1 }),
		ui.Button("two", func(ctx ui.EventContext) { pressed = 2 }),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want previous focus to wrap to second button", pressed)
	}
}

func TestNilButtonCallbackStillHandlesActivation(t *testing.T) {
	app := ui.NewApp(ui.Button("noop", nil))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
}

func TestQuitCallbackDoesNotPanic(t *testing.T) {
	app := ui.NewApp(ui.Button("quit", func(ctx ui.EventContext) { ctx.Quit() }))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !app.ShouldQuit() {
		t.Fatal("expected app to request quit")
	}
}

func TestFocusTraversalSkipsUnmountedFocus(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button("one", func(ctx ui.EventContext) { pressed = 1 }),
		ui.Button("two", func(ctx ui.EventContext) { pressed = 2 }),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.UpdateRoot(ui.Row(
		ui.Button("two", func(ctx ui.EventContext) { pressed = 2 }),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want remaining button after unmount", pressed)
	}
}

func TestButtonActivatesOnMouseClick(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Button("click", func(ctx ui.EventContext) { pressed = true }))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !pressed {
		t.Fatal("expected mouse click to activate button")
	}
}

func TestMouseClickOutsideWidgetIsIgnored(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Row(ui.Button("click", func(ctx ui.EventContext) { pressed = true })))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 10, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed {
		t.Fatal("button should not activate for outside click")
	}
}

func TestRightMouseClickDoesNotActivateButton(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Button("click", func(ctx ui.EventContext) { pressed = true }))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseRightButton, EventType: vaxis.EventPress})
	if pressed {
		t.Fatal("button should not activate for right click")
	}
}

func TestTextUsesThemeStyle(t *testing.T) {
	style := ui.Style{Foreground: vaxis.ColorRed}
	app := ui.NewApp(ui.Text("x"), ui.WithTheme(ui.Theme{Text: style}))
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Style; got != style {
		t.Fatalf("style = %#v, want %#v", got, style)
	}
}

func TestButtonUsesFocusedThemeStyle(t *testing.T) {
	focused := ui.Style{Foreground: vaxis.ColorGreen}
	app := ui.NewApp(ui.Button("go", nil), ui.WithTheme(ui.Theme{Button: ui.ButtonTheme{Focused: focused}}))
	app.Pump(ui.Size{Width: 4, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 4, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style; got != focused {
		t.Fatalf("style = %#v, want focused %#v", got, focused)
	}
}

func TestButtonStyleUpdatesOnFocusChange(t *testing.T) {
	focused := ui.Style{Foreground: vaxis.ColorGreen}
	app := ui.NewApp(ui.Row(
		ui.Button("a", nil),
		ui.Button("b", nil),
	), ui.WithTheme(ui.Theme{Button: ui.ButtonTheme{Focused: focused}}))
	app.Pump(ui.Size{Width: 8, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 8, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 8, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Style; got == focused {
		t.Fatal("first button should no longer have focused style")
	}
	if got := p.Cell(4, 0).Style; got != focused {
		t.Fatalf("second button style = %#v, want focused %#v", got, focused)
	}
}
