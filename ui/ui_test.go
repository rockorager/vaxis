package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

type callbackIntent string

func (i callbackIntent) IntentType() ui.IntentType {
	return ui.IntentType(i)
}

func TestTextPaints(t *testing.T) {
	app := uitest.New(ui.Text{Value: "hello"})
	app.Pump(10, 1)
	if got := app.Text(); !strings.HasPrefix(got, "hello") {
		t.Fatalf("text = %q, want prefix hello", got)
	}
}

func TestCenterPaintsChildCentered(t *testing.T) {
	app := uitest.New(ui.Center(ui.Text{Value: "hi"}))
	app.Pump(6, 3)
	if got := app.Cell(2, 1).Grapheme; got != "h" {
		t.Fatalf("centered cell = %q, want h", got)
	}
}

type counter struct{ initial int }

func (c counter) CreateState() ui.State {
	return &counterState{count: c.initial}
}

type counterState struct {
	ui.StateBase
	count int
}

func (s *counterState) Build(ctx ui.BuildContext) ui.Widget {
	return (ui.Text{Value: string(rune('0' + s.count))})
}

func TestStatePreservedAcrossCompatibleUpdate(t *testing.T) {
	app := ui.NewApp(counter{initial: 1})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(counter{initial: 9})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "1" {
		t.Fatalf("state text = %q, want preserved 1", got)
	}
}

type keyedCounter struct {
	key     ui.KeyValue
	initial int
}

func (c keyedCounter) WidgetKey() ui.KeyValue {
	return c.key
}

func (c keyedCounter) CreateState() ui.State {
	return &counterState{count: c.initial}
}

func TestStateRecreatedWhenKeyChanges(t *testing.T) {
	app := ui.NewApp(keyedCounter{key: "a", initial: 1})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(keyedCounter{key: "b", initial: 9})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "9" {
		t.Fatalf("state text = %q, want recreated 9", got)
	}
}

func TestKeyedChildrenMoveWithState(t *testing.T) {
	app := ui.NewApp(ui.Row(
		keyedCounter{key: "a", initial: 1},
		keyedCounter{key: "b", initial: 2},
	))
	app.Pump(ui.Size{Width: 2, Height: 1})
	app.UpdateRoot(ui.Row(
		keyedCounter{key: "b", initial: 9},
		keyedCounter{key: "a", initial: 8},
	))
	app.Pump(ui.Size{Width: 2, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 2, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme + p.Cell(1, 0).Grapheme; got != "21" {
		t.Fatalf("state text = %q, want moved keyed state 21", got)
	}
}

type multiKind struct{}

func (multiKind) Build(ctx ui.BuildContext) ui.Widget {
	return (ui.Text{Value: "bad"})
}

func (multiKind) CreateState() ui.State {
	return &counterState{}
}

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
	return (ui.Text{Value: ui.MustDepend[string](ctx)})
}

func TestProviderNotifiesDependents(t *testing.T) {
	root := ui.Provider[string]{Value: "a", Child: providerText{}}
	app := ui.NewApp(root)
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(ui.Provider[string]{Value: "b", Child: providerText{}})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "b" {
		t.Fatalf("provider text = %q, want b", got)
	}
}

func TestProviderShouldNotifyCanSuppressDependentRebuild(t *testing.T) {
	called := false
	root := ui.Provider[string]{Value: "a", Child: providerText{}, ShouldNotify: func(old, next string) bool {
		called = true
		return false
	}}
	app := ui.NewApp(root)
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(ui.Provider[string]{Value: "b", Child: providerText{}})
	app.Pump(ui.Size{Width: 1, Height: 1})
	if !called {
		t.Fatal("expected ShouldNotify to be called")
	}
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "a" {
		t.Fatalf("provider text = %q, want stale a", got)
	}
}

func TestPaddingOffsetsChild(t *testing.T) {
	app := uitest.New(ui.Padding(ui.All(1), ui.Text{Value: "x"}))
	app.Pump(3, 3)
	if got := app.Cell(1, 1).Grapheme; got != "x" {
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
			root:   ui.Row(ui.Text{Value: "a"}, ui.Expanded(ui.Text{Value: "b"}), ui.Text{Value: "c"}),
			w:      5,
			h:      1,
			checks: map[ui.Point]string{{X: 0, Y: 0}: "a", {X: 1, Y: 0}: "b", {X: 4, Y: 0}: "c"},
		},
		{
			name:   "column",
			root:   ui.Column(ui.Text{Value: "a"}, ui.Expanded(ui.Text{Value: "b"}), ui.Text{Value: "c"}),
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
				if got := app.Cell(pt.X, pt.Y).Grapheme; got != want {
					t.Fatalf("cell %v = %q, want %q", pt, got, want)
				}
			}
		})
	}
}

type updatingCounter struct{}

func (u updatingCounter) CreateState() ui.State {
	return &updatingCounterState{}
}

type updatingCounterState struct {
	ui.StateBase
	updates int
}

func (s *updatingCounterState) DidUpdateWidget(old ui.Widget) {
	s.updates++
}

func (s *updatingCounterState) Build(ctx ui.BuildContext) ui.Widget {
	return (ui.Text{Value: fmt.Sprint(s.updates)})
}

func TestDidUpdateWidgetCalledOnCompatibleUpdate(t *testing.T) {
	app := ui.NewApp(updatingCounter{})
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(updatingCounter{})
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "1" {
		t.Fatalf("updates = %q, want 1", got)
	}
}

func TestButtonActivatesFocusedButton(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed += 1 }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed += 10 }},
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
		ui.Focus(&n1, ui.Text{Value: "one"}),
		ui.Focus(&n2, ui.Text{Value: "two"}),
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	n2.RequestFocus()
	if !n2.HasFocus() || n1.HasFocus() {
		t.Fatal("expected second focus node to have focus")
	}
}

func TestShortcutActionHandlesBeforeFocusedButton(t *testing.T) {
	intent := callbackIntent("test.capture")
	called := false
	button := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			intent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				called = true
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{"Enter": intent},
			Child:    ui.Button{Label: "button", OnPressed: func(ctx ui.EventContext) { button = true }},
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !called {
		t.Fatal("expected shortcut action")
	}
	if button {
		t.Fatal("button should not run after shortcut action handled event")
	}
}

func TestShortcutIgnoresKeyRelease(t *testing.T) {
	intent := callbackIntent("test.release")
	called := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			intent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				called = true
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{"Enter": intent},
			Child:    ui.Text{Value: "x"},
		},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
	if called {
		t.Fatal("shortcut should ignore key release")
	}
}

func TestUnhandledShortcutIntentContinuesToFocusedButton(t *testing.T) {
	button := false
	app := ui.NewApp(ui.Shortcuts{
		Bindings: map[string]ui.Intent{"Ctrl+x": callbackIntent("test.unhandled")},
		Child:    ui.Button{Label: "button", OnPressed: func(ctx ui.EventContext) { button = true }},
	})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !button {
		t.Fatal("expected button to receive event after shortcut intent was ignored")
	}
}

func TestFocusTraversalIgnoresTabRelease(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = 1 }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab, EventType: vaxis.EventRelease})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 1 {
		t.Fatalf("pressed = %d, want first button after ignored tab release", pressed)
	}
}

func TestShiftTabMovesFocusPrevious(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = 1 }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want previous focus to wrap to second button", pressed)
	}
}

func TestNilButtonCallbackStillHandlesActivation(t *testing.T) {
	app := ui.NewApp(ui.Button{Label: "noop"})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
}

func TestButtonIgnoresKeyRelease(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Button{Label: "button", OnPressed: func(ctx ui.EventContext) { pressed = true }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter, EventType: vaxis.EventRelease})
	if pressed {
		t.Fatal("button should ignore key release")
	}
}

func TestQuitCallbackDoesNotPanic(t *testing.T) {
	app := ui.NewApp(ui.Button{Label: "quit", OnPressed: func(ctx ui.EventContext) { ctx.Quit() }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !app.ShouldQuit() {
		t.Fatal("expected app to request quit")
	}
}

func TestFocusTraversalSkipsUnmountedFocus(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = 1 }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.UpdateRoot(ui.Row(
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = 2 }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want remaining button after unmount", pressed)
	}
}

func TestButtonActivatesOnMouseClick(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Button{Label: "click", OnPressed: func(ctx ui.EventContext) { pressed = true }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !pressed {
		t.Fatal("expected mouse click to activate button")
	}
}

func TestMouseHitTestingUsesCustomRenderChildOffset(t *testing.T) {
	pressed := false
	app := ui.NewApp(offsetBox{offset: ui.Offset{X: 4, Y: 2}, child: ui.Button{Label: "hit", OnPressed: func(ctx ui.EventContext) { pressed = true }}})
	app.Pump(ui.Size{Width: 20, Height: 5})
	app.Send(vaxis.Mouse{Col: 5, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !pressed {
		t.Fatal("expected click translated through custom render child offset to activate button")
	}
}

func TestMouseHitTestingMissesCustomRenderChildOffset(t *testing.T) {
	pressed := false
	app := ui.NewApp(offsetBox{offset: ui.Offset{X: 4, Y: 2}, child: ui.Button{Label: "hit", OnPressed: func(ctx ui.EventContext) { pressed = true }}})
	app.Pump(ui.Size{Width: 20, Height: 5})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed {
		t.Fatal("button should not activate outside custom render child offset")
	}
}

func TestMouseClickOutsideWidgetIsIgnored(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Row(ui.Button{Label: "click", OnPressed: func(ctx ui.EventContext) { pressed = true }}))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 10, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed {
		t.Fatal("button should not activate for outside click")
	}
}

func TestRightMouseClickDoesNotActivateButton(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Button{Label: "click", OnPressed: func(ctx ui.EventContext) { pressed = true }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseRightButton, EventType: vaxis.EventPress})
	if pressed {
		t.Fatal("button should not activate for right click")
	}
}

func TestButtonSetsMouseShapeOnHover(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Button{Label: "click"}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape = %q, want clickable", got)
	}
	app.Send(vaxis.Mouse{Col: 10, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape after leaving button = %q, want default", got)
	}
}

func TestButtonUsesHoveredThemeStyle(t *testing.T) {
	hovered := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	normal := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlack}
	theme := ui.DefaultTheme()
	theme.Foreground = normal.Foreground
	theme.Surface = normal.Background
	theme.Primary = normal.Background
	theme.PrimaryHovered = hovered.Background
	theme.SurfaceHovered = hovered.Background
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Button{Label: "go"}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(2, 0).Style; got != hovered {
		t.Fatalf("hovered button style = %#v, want %#v", got, hovered)
	}
	app.Send(vaxis.Mouse{Col: 9, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(2, 0).Style; got != hovered {
		t.Fatalf("focused button style = %#v, want %#v", got, hovered)
	}
}

func TestButtonPaddingAndMinWidth(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Button{Label: "go", Padding: ui.Symmetric(2, 1), MinWidth: 10}})
	app.Pump(ui.Size{Width: 10, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Background; got == 0 {
		t.Fatal("button padding should paint styled surface")
	}
	if got := p.Cell(3, 1).Grapheme; got != "[" {
		t.Fatalf("focused marker at padded center = %q, want [", got)
	}
	if got := p.Cell(4, 1).Grapheme; got != "g" {
		t.Fatalf("label at padded center = %q, want g", got)
	}
	if got := p.Cell(9, 2).Background; got == 0 {
		t.Fatal("button min width and vertical padding should paint full surface")
	}
}

func TestButtonUsesWidgetPaddingAndMinWidth(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Button{Label: "x", Padding: ui.Symmetric(3, 0), MinWidth: 12}})
	app.Pump(ui.Size{Width: 12, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 12, Height: 1})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme; got != "[" {
		t.Fatalf("theme-padded focused marker = %q, want [", got)
	}
	if got := p.Cell(5, 0).Grapheme; got != "x" {
		t.Fatalf("theme-padded label = %q, want x", got)
	}
	if got := p.Cell(11, 0).Background; got == 0 {
		t.Fatal("theme min width should paint full surface")
	}
}

func TestTextUsesThemeStyle(t *testing.T) {
	style := ui.Style{Foreground: vaxis.ColorRed}
	theme := ui.DefaultTheme()
	theme.Foreground = style.Foreground
	app := ui.NewApp(ui.Text{Value: "x"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Style; got != style {
		t.Fatalf("style = %#v, want %#v", got, style)
	}
}

func TestTextFillsConstrainedBackground(t *testing.T) {
	style := ui.Style{Foreground: vaxis.ColorWhite, Background: vaxis.ColorBlue}
	app := ui.NewApp(ui.ConstrainedBox{
		Constraints: ui.Constraints{MinWidth: 5, MinHeight: 2},
		Child:       ui.Text{Value: "hi", Style: style},
	})
	app.Pump(ui.Size{Width: 5, Height: 2})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("text glyph = %q, want h", got)
	}
	for _, pt := range []ui.Point{{X: 2, Y: 0}, {X: 4, Y: 0}, {X: 0, Y: 1}, {X: 4, Y: 1}} {
		if got := p.Cell(pt.X, pt.Y).Background; got != style.Background {
			t.Fatalf("background at %v = %#v, want %#v", pt, got, style.Background)
		}
	}
}

func TestRichTextDoesNotInferConstrainedBackground(t *testing.T) {
	style := ui.Style{Background: vaxis.ColorBlue}
	app := ui.NewApp(ui.ConstrainedBox{
		Constraints: ui.Constraints{MinWidth: 6, MinHeight: 2},
		Child: ui.RichText{Spans: []ui.TextSpan{
			{Text: "a", Style: style},
			{Text: "b", Style: ui.Style{Attribute: ui.AttrBold}},
		}},
	})
	app.Pump(ui.Size{Width: 6, Height: 2})
	p := ui.NewPainter(ui.Size{Width: 6, Height: 2})
	app.Paint(p)
	for _, pt := range []ui.Point{{X: 2, Y: 0}, {X: 5, Y: 0}, {X: 0, Y: 1}, {X: 5, Y: 1}} {
		if got := p.Cell(pt.X, pt.Y).Background; got != 0 {
			t.Fatalf("rich text inferred background at %v = %#v, want default", pt, got)
		}
	}
}

func TestRichTextPaintsStyledSpans(t *testing.T) {
	bold := ui.Style{Attribute: ui.AttrBold}
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "hi "},
		{Text: "there", Style: bold},
	}})
	app.Pump(ui.Size{Width: 8, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 8, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first span = %q, want h", got)
	}
	if got := p.Cell(3, 0).Grapheme; got != "t" {
		t.Fatalf("second span = %q, want t", got)
	}
	if got := p.Cell(3, 0).Attribute; got != ui.AttrBold {
		t.Fatalf("second span attr = %#v, want bold", got)
	}
}

func TestRichTextMergesSpanStyleWithTheme(t *testing.T) {
	themeStyle := ui.Style{Foreground: vaxis.ColorGreen}
	theme := ui.DefaultTheme()
	theme.Foreground = themeStyle.Foreground
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{{Text: "x", Style: ui.Style{Attribute: ui.AttrBold}}}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 1, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 1, Height: 1})
	app.Paint(p)
	style := p.Cell(0, 0).Style
	if style.Foreground != themeStyle.Foreground || style.Attribute != ui.AttrBold {
		t.Fatalf("style = %#v, want theme foreground with bold", style)
	}
}

func TestTextSoftWrapsWords(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "hello world", SoftWrap: true})
	app.Pump(ui.Size{Width: 6, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 6, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first line = %q, want h", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "w" {
		t.Fatalf("second line = %q, want w", got)
	}
}

func TestTextSoftWrapBreaksLongWords(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "abcdef", SoftWrap: true})
	app.Pump(ui.Size{Width: 3, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 3})
	app.Paint(p)
	if got := p.Cell(2, 0).Grapheme; got != "c" {
		t.Fatalf("first line end = %q, want c", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "d" {
		t.Fatalf("second line start = %q, want d", got)
	}
}

func TestTextHardNewlineBreaksWhenSoftWrapFalse(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "a\nb"})
	app.Pump(ui.Size{Width: 3, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "a" {
		t.Fatalf("first line = %q, want a", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "b" {
		t.Fatalf("second line = %q, want b", got)
	}
}

func TestTextMaxLinesEllipsis(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "abcdef", Overflow: ui.TextOverflowEllipsis, MaxLines: 1})
	app.Pump(ui.Size{Width: 4, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 4, Height: 1})
	app.Paint(p)
	if got := p.Cell(3, 0).Grapheme; got != "…" {
		t.Fatalf("ellipsis = %q, want …", got)
	}
}

func TestTextAlignCenter(t *testing.T) {
	app := ui.NewApp(ui.SizedBox{Width: 5, Height: 1, Child: ui.Text{Value: "x", Align: ui.TextAlignCenter}})
	app.Pump(ui.Size{Width: 5, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 1})
	app.Paint(p)
	if got := p.Cell(2, 0).Grapheme; got != "x" {
		t.Fatalf("centered text = %q, want x", got)
	}
}

func TestRichTextWrapPreservesSpanStyle(t *testing.T) {
	bold := ui.Style{Attribute: ui.AttrBold}
	app := ui.NewApp(ui.RichText{SoftWrap: true, Spans: []ui.TextSpan{{Text: "aa "}, {Text: "bb", Style: bold}}})
	app.Pump(ui.Size{Width: 3, Height: 2})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 1).Grapheme; got != "b" {
		t.Fatalf("wrapped rich text = %q, want b", got)
	}
	if got := p.Cell(0, 1).Attribute; got != ui.AttrBold {
		t.Fatalf("wrapped rich text attr = %#v, want bold", got)
	}
}

func TestButtonUsesFocusedThemeStyle(t *testing.T) {
	focused := ui.Style{Foreground: vaxis.ColorGreen, Background: vaxis.ColorYellow}
	theme := ui.DefaultTheme()
	theme.Foreground = focused.Foreground
	theme.SurfaceHovered = focused.Background
	app := ui.NewApp(ui.Button{Label: "go"}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 6, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 6, Height: 1})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "[" {
		t.Fatalf("focused left marker = %q, want [", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "g" {
		t.Fatalf("centered label starts at cell 2 = %q, want g", got)
	}
	if got := p.Cell(2, 0).Style; got != focused {
		t.Fatalf("label style = %#v, want focused %#v", got, focused)
	}
	if got := p.Cell(4, 0).Grapheme; got != "]" {
		t.Fatalf("focused right marker = %q, want ]", got)
	}
}

func TestButtonStyleUpdatesOnFocusChange(t *testing.T) {
	focused := ui.Style{Foreground: vaxis.ColorGreen, Background: vaxis.ColorYellow}
	theme := ui.DefaultTheme()
	theme.Foreground = focused.Foreground
	theme.SurfaceHovered = focused.Background
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "a"},
		ui.Button{Label: "b"},
	), ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got == "[" {
		t.Fatal("first button should no longer show focus marker")
	}
	if got := p.Cell(6, 0).Grapheme; got != "[" {
		t.Fatalf("second button left marker = %q, want [", got)
	}
	if got := p.Cell(7, 0).Style; got != focused {
		t.Fatalf("second button label style = %#v, want focused %#v", got, focused)
	}
	if got := p.Cell(2, 0).Style; got == focused {
		t.Fatal("first button should no longer have focused style")
	}
}

func TestDecoratedBoxPaintsFillBehindChild(t *testing.T) {
	style := ui.Style{Background: vaxis.ColorBlue}
	app := ui.NewApp(ui.DecoratedBox(ui.Decoration{Style: style}, ui.Padding(ui.All(1), ui.Text{Value: "x"})))
	app.Pump(ui.Size{Width: 3, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Style; got != style {
		t.Fatalf("fill style = %#v, want %#v", got, style)
	}
	if got := p.Cell(1, 1).Grapheme; got != "x" {
		t.Fatalf("child cell = %q, want x", got)
	}
	if got := p.Cell(1, 1).Background; got != style.Background {
		t.Fatalf("child background = %#v, want inherited decoration background %#v", got, style.Background)
	}
}

func TestDecoratedBoxPaintsBorder(t *testing.T) {
	style := ui.Style{Foreground: vaxis.ColorRed}
	app := ui.NewApp(ui.DecoratedBox(ui.Decoration{Border: ui.BorderAll(style)}, ui.Padding(ui.All(1), ui.Text{Value: "x"})))
	app.Pump(ui.Size{Width: 3, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "┌" {
		t.Fatalf("top-left border = %q, want ┌", got)
	}
	if got := p.Cell(2, 1).Grapheme; got != "│" {
		t.Fatalf("right border = %q, want │", got)
	}
	if got := p.Cell(0, 0).Style; got != style {
		t.Fatalf("border style = %#v, want %#v", got, style)
	}
}

func TestDecoratedBoxUpdateRequestsPaintFrame(t *testing.T) {
	app := ui.NewApp(ui.DecoratedBox(ui.Decoration{Style: ui.Style{Background: vaxis.ColorBlue}}, ui.Text{Value: "x"}))
	app.Pump(ui.Size{Width: 1, Height: 1})
	app.UpdateRoot(ui.DecoratedBox(ui.Decoration{Style: ui.Style{Background: vaxis.ColorRed}}, ui.Text{Value: "x"}))
	if !app.FrameRequested() {
		t.Fatal("decoration update should request a paint frame")
	}
}

func TestSizedBoxTightensChild(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.SizedBox{Width: 2, Height: 1, Child: ui.Text{Value: "abcd"}}})
	app.Pump(ui.Size{Width: 10, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "a" {
		t.Fatalf("first cell = %q, want a", got)
	}
	if got := p.Cell(1, 0).Grapheme; got != "b" {
		t.Fatalf("second cell = %q, want b", got)
	}
	if got := p.Cell(2, 0).Grapheme; got != "" {
		t.Fatalf("third cell = %q, want clipped empty cell", got)
	}
}

func TestSizedBoxUnspecifiedAxisFollowsChild(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.SizedBox{Width: 4, Child: ui.Text{Value: "x"}}})
	app.Pump(ui.Size{Width: 10, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "x" {
		t.Fatalf("width-only sized box painted child = %q, want x", got)
	}

	app = ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.SizedBox{Height: 1, Child: ui.Text{Value: "xy"}}})
	app.Pump(ui.Size{Width: 10, Height: 3})
	p = ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(1, 0).Grapheme; got != "y" {
		t.Fatalf("height-only sized box painted child = %q, want y", got)
	}
}

func TestAlignPositionsChild(t *testing.T) {
	app := ui.NewApp(ui.Align{Alignment: ui.BottomRight, Child: ui.Text{Value: "x"}})
	app.Pump(ui.Size{Width: 4, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 4, Height: 3})
	app.Paint(p)
	if got := p.Cell(3, 2).Grapheme; got != "x" {
		t.Fatalf("bottom-right aligned cell = %q, want x", got)
	}
}

func TestAlignShrinkWrapsUnboundedAxis(t *testing.T) {
	app := ui.NewApp(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.Align{Alignment: ui.TopCenter, Child: ui.DecoratedBox(
			ui.Decoration{Style: ui.Style{Background: ui.RGB(1, 2, 3)}},
			ui.Padding(ui.All(1), ui.Text{Value: "x"}),
		)},
	}})
	app.Pump(ui.Size{Width: 5, Height: 5})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 5})
	app.Paint(p)
	if got := p.Cell(2, 0).Background; got != ui.RGB(1, 2, 3) {
		t.Fatalf("aligned panel top padding background = %#v, want decorated background", got)
	}
	if got := p.Cell(2, 1).Grapheme; got != "x" {
		t.Fatalf("aligned panel text = %q, want x", got)
	}
	if got := p.Cell(2, 2).Background; got != ui.RGB(1, 2, 3) {
		t.Fatalf("aligned panel bottom padding background = %#v, want decorated background", got)
	}
}

func TestCenterShrinkWrapsUnboundedAxis(t *testing.T) {
	app := ui.NewApp(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.Center(ui.DecoratedBox(
			ui.Decoration{Style: ui.Style{Background: ui.RGB(4, 5, 6)}},
			ui.Padding(ui.All(1), ui.Text{Value: "x"}),
		)),
	}})
	app.Pump(ui.Size{Width: 5, Height: 5})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 5})
	app.Paint(p)
	if got := p.Cell(2, 0).Background; got != ui.RGB(4, 5, 6) {
		t.Fatalf("centered panel top padding background = %#v, want decorated background", got)
	}
	if got := p.Cell(2, 1).Grapheme; got != "x" {
		t.Fatalf("centered panel text = %q, want x", got)
	}
	if got := p.Cell(2, 2).Background; got != ui.RGB(4, 5, 6) {
		t.Fatalf("centered panel bottom padding background = %#v, want decorated background", got)
	}
}

func TestAlignHitTestingUsesRelayoutOffset(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Align{Alignment: ui.BottomRight, Child: ui.Button{Label: "x", OnPressed: func(ctx ui.EventContext) { pressed = true }}})
	app.Pump(ui.Size{Width: 4, Height: 3})
	app.Pump(ui.Size{Width: 8, Height: 3})
	app.Send(vaxis.Mouse{Col: 7, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !pressed {
		t.Fatal("expected click at relaid-out alignment offset to activate button")
	}
}

func TestAlignDoesNotHitTestOutsideChild(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Stack{Alignment: ui.TopLeft, Children: []ui.Widget{
		ui.Button{Label: "bottom", OnPressed: func(ctx ui.EventContext) { pressed = true }},
		ui.Align{Alignment: ui.BottomRight, Child: ui.Button{Label: "top", OnPressed: func(ctx ui.EventContext) {}}},
	}})
	app.Pump(ui.Size{Width: 20, Height: 4})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if !pressed {
		t.Fatal("expected click outside aligned child to pass through to lower stack child")
	}
}

type offsetBox struct {
	offset ui.Offset
	child  ui.Widget
}

func (b offsetBox) WidgetChild() ui.Widget {
	return b.child
}

func (b offsetBox) CreateRenderObject(ctx ui.BuildContext) ui.RenderObject {
	return &offsetRender{offset: b.offset}
}

func (b offsetBox) UpdateRenderObject(ctx ui.BuildContext, ro ui.RenderObject) {
	ro.(*offsetRender).offset = b.offset
}

type offsetRender struct {
	ui.SingleChildRenderObject
	offset ui.Offset
}

func (r *offsetRender) Layout(ctx ui.LayoutContext, c ui.Constraints) {
	if child := r.Child(); child != nil {
		child.Layout(ctx, c)
	}
	r.SetSize(c.Constrain(ui.Size{Width: 20, Height: 5}))
}

func (r *offsetRender) Paint(p *ui.Painter, off ui.Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *offsetRender) HitTest(*ui.HitTestResult, ui.Point) bool {
	return false
}

func (r *offsetRender) ChildOffset(ui.RenderObject) ui.Offset {
	return r.offset
}
