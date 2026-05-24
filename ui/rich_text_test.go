package ui_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestRichTextPaintsInlineHyperlink(t *testing.T) {
	app := uitest.New(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", Style: ui.Style{Hyperlink: "https://vaxis.dev", HyperlinkParams: "id=docs"}},
		{Text: " now"},
	}})
	app.Pump(20, 1)
	if got := app.Cell(3, 0).Hyperlink; got != "" {
		t.Fatalf("pre-link hyperlink = %q, want empty", got)
	}
	if got := app.Cell(4, 0).Hyperlink; got != "https://vaxis.dev" {
		t.Fatalf("link hyperlink = %q, want https://vaxis.dev", got)
	}
	if got := app.Cell(4, 0).HyperlinkParams; got != "id=docs" {
		t.Fatalf("link params = %q, want id=docs", got)
	}
	if got := app.Cell(8, 0).Hyperlink; got != "" {
		t.Fatalf("post-link hyperlink = %q, want empty", got)
	}
}

func TestRichTextHyperlinkMergesWithTheme(t *testing.T) {
	theme := ui.DefaultTheme()
	theme.Foreground = ui.RGB(12, 34, 56)
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "docs", Style: ui.Style{Hyperlink: "https://vaxis.dev"}},
	}}, ui.WithTheme(theme))
	app.Pump(ui.Size{Width: 10, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	cell := p.Cell(0, 0)
	if got := cell.Hyperlink; got != "https://vaxis.dev" {
		t.Fatalf("hyperlink = %q, want https://vaxis.dev", got)
	}
	if got := cell.Foreground; got != theme.Foreground {
		t.Fatalf("foreground = %#v, want theme foreground %#v", got, theme.Foreground)
	}
}

func TestRichTextSpanOnPressedActivatesOnMouseClick(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 1 {
		t.Fatalf("pressed = %d, want 1", pressed)
	}
}

func TestRichTextSpanOnPressedIgnoresOutsideSpan(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Send(vaxis.Mouse{Col: 8, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 0 {
		t.Fatalf("pressed = %d, want 0", pressed)
	}
}

func TestRichTextSpanOnPressedSetsMouseShape(t *testing.T) {
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape over link = %q, want clickable", got)
	}
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape outside link = %q, want default", got)
	}
}

func TestRichTextSpanOnHoverActivatesOnMouseMotion(t *testing.T) {
	hovered := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnHover: func(ctx ui.EventContext) { hovered++ }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if hovered != 1 {
		t.Fatalf("hovered = %d, want 1", hovered)
	}
}

func TestRichTextSpanOnHoverExitActivatesWhenMouseLeavesSpan(t *testing.T) {
	exited := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "see "},
		{Text: "docs", OnHover: func(ctx ui.EventContext) {}, OnHoverExit: func(ctx ui.EventContext) { exited++ }},
		{Text: " please"},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Send(vaxis.Mouse{Col: 9, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if exited != 1 {
		t.Fatalf("exited = %d, want 1", exited)
	}
}

func TestRichTextSpanOnHoverExitActivatesWhenMouseLeavesWidget(t *testing.T) {
	exited := 0
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{{Text: "docs", OnHover: func(ctx ui.EventContext) {}, OnHoverExit: func(ctx ui.EventContext) { exited++ }}}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	app.Send(vaxis.Mouse{Col: 30, Row: 3, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if exited != 1 {
		t.Fatalf("exited = %d, want 1", exited)
	}
}

func TestRichTextSpanOnPressedPaintsSingleUnderlineWhenUnfocused(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "before"},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
		}},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(10, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused interactive span underline = %#v, want single", got)
	}
}

func TestRichTextSpanOnPressedActivatesWithKeyboardFocus(t *testing.T) {
	var pressed []string
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "one", OnPressed: func(ctx ui.EventContext) { pressed = append(pressed, "one") }},
		{Text: " "},
		{Text: "two", OnPressed: func(ctx ui.EventContext) { pressed = append(pressed, "two") }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeySpace})
	if len(pressed) != 2 || pressed[0] != "one" || pressed[1] != "two" {
		t.Fatalf("pressed = %#v, want [one two]", pressed)
	}
}

func TestRichTextSpanOnFocusActivatesWithKeyboardFocus(t *testing.T) {
	var focused []string
	var exited []string
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "one", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() { focused = append(focused, "one") }, OnFocusExit: func() { exited = append(exited, "one") }},
		{Text: " "},
		{Text: "two", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() { focused = append(focused, "two") }, OnFocusExit: func() { exited = append(exited, "two") }},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	if len(focused) != 2 || focused[0] != "one" || focused[1] != "two" {
		t.Fatalf("focused = %#v, want [one two]", focused)
	}
	if len(exited) != 1 || exited[0] != "one" {
		t.Fatalf("exited = %#v, want [one]", exited)
	}
}

func TestRichTextSpanFocusSurvivesOverlayAddedFromFocusCallback(t *testing.T) {
	var events []string
	app := ui.NewApp(richTextFocusOverlay{Events: &events})
	app.Pump(ui.Size{Width: 30, Height: 3})
	app.Pump(ui.Size{Width: 30, Height: 3})
	app.Pump(ui.Size{Width: 30, Height: 3})
	if len(events) != 1 || events[0] != "focus" {
		t.Fatalf("focus events = %#v, want only initial focus", events)
	}
	p := ui.NewPainter(ui.Size{Width: 30, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused link underline = %#v, want double after overlay appears", got)
	}
	if got := p.Cell(0, 2).Grapheme; got != "h" {
		t.Fatalf("overlay text = %q, want URL at bottom-left", got)
	}
}

func TestRichTextSpanFocusSurvivesRootOverlayAddedFromDialogFocusCallback(t *testing.T) {
	var events []string
	app := ui.NewApp(richTextDialogFocusOverlay{Events: &events})
	size := ui.Size{Width: 80, Height: 10}
	app.Pump(size)
	for i := 0; i < 10 && focusedDebugLabel(app) != "one"; i++ {
		app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
		app.Pump(size)
	}
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label before overlay = %q, want one", got)
	}
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label after overlay appears = %q, want one", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "two" {
		t.Fatalf("focused label after Tab = %q, want two", got)
	}
	if len(events) < 2 || events[0] != "one" || events[len(events)-1] != "two" {
		t.Fatalf("focus events = %#v, want one then two", events)
	}
}

func TestRichTextSpanFocusSurvivesFocusedTargetReplacementDuringBuild(t *testing.T) {
	var events []string
	app := ui.NewApp(replacingFocusedRichText{Events: &events})
	size := ui.Size{Width: 40, Height: 3}
	app.Pump(size)
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label before replacement = %q, want one", got)
	}
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label after replacement = %q, want one", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "two" {
		t.Fatalf("focused label after Tab = %q, want two", got)
	}
	if len(events) < 2 || events[0] != "one" || events[len(events)-1] != "two" {
		t.Fatalf("focus events = %#v, want one then two", events)
	}
}

func TestRichTextSpanFocusSurvivesReplacementAfterAncestorFocusableRemoved(t *testing.T) {
	var events []string
	app := ui.NewApp(replacingFocusedRichTextWithFocusableAncestor{Events: &events})
	size := ui.Size{Width: 40, Height: 3}
	app.Pump(size)
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label before replacement = %q, want one", got)
	}
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "one" {
		t.Fatalf("focused label after replacement = %q, want one", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(size)
	if got := focusedDebugLabel(app); got != "two" {
		t.Fatalf("focused label after Tab = %q, want two", got)
	}
	if len(events) < 2 || events[0] != "one" || events[len(events)-1] != "two" {
		t.Fatalf("focus events = %#v, want one then two", events)
	}
}

type replacingFocusedRichTextWithFocusableAncestor struct {
	Events *[]string
}

func (w replacingFocusedRichTextWithFocusableAncestor) CreateState() ui.State {
	return &replacingFocusedRichTextWithFocusableAncestorState{}
}

type replacingFocusedRichTextWithFocusableAncestorState struct {
	ui.StateBase
	replace bool
}

func (s *replacingFocusedRichTextWithFocusableAncestorState) Build(ctx ui.BuildContext) ui.Widget {
	w := s.Widget().(replacingFocusedRichTextWithFocusableAncestor)
	key := ui.KeyValue("before")
	if s.replace {
		key = "after"
	}
	return ui.Row(
		ui.Button{Label: "before", OnPressed: func(ctx ui.EventContext) {}},
		keyedChild{Key: key, Child: ui.Focus(nil, ui.RichText{Spans: []ui.TextSpan{
			{Text: "one", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
				*w.Events = append(*w.Events, "one")
				s.SetState(func() { s.replace = true })
			}},
			{Text: " "},
			{Text: "two", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
				*w.Events = append(*w.Events, "two")
			}},
		}})},
	)
}

type keyedChild struct {
	Key   ui.KeyValue
	Child ui.Widget
}

func (w keyedChild) WidgetKey() ui.KeyValue {
	return w.Key
}

func (w keyedChild) Build(ui.BuildContext) ui.Widget {
	return w.Child
}

type replacingFocusedRichText struct {
	Events *[]string
}

func (w replacingFocusedRichText) CreateState() ui.State {
	return &replacingFocusedRichTextState{}
}

type replacingFocusedRichTextState struct {
	ui.StateBase
	replace bool
}

func (s *replacingFocusedRichTextState) Build(ctx ui.BuildContext) ui.Widget {
	w := s.Widget().(replacingFocusedRichText)
	key := ui.KeyValue("before")
	if s.replace {
		key = "after"
	}
	return ui.Row(
		ui.Button{Label: "before", OnPressed: func(ctx ui.EventContext) {}},
		keyedRichText{
			RichText: ui.RichText{Spans: []ui.TextSpan{
				{Text: "one", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
					*w.Events = append(*w.Events, "one")
					s.SetState(func() { s.replace = true })
				}},
				{Text: " "},
				{Text: "two", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
					*w.Events = append(*w.Events, "two")
				}},
			}},
			Key: key,
		},
	)
}

type keyedRichText struct {
	ui.RichText
	Key ui.KeyValue
}

func (w keyedRichText) WidgetKey() ui.KeyValue {
	return w.Key
}

type richTextDialogFocusOverlay struct {
	Events *[]string
}

func (w richTextDialogFocusOverlay) CreateState() ui.State {
	return &richTextDialogFocusOverlayState{}
}

type richTextDialogFocusOverlayState struct {
	ui.StateBase
	show bool
}

func (s *richTextDialogFocusOverlayState) Build(ctx ui.BuildContext) ui.Widget {
	w := s.Widget().(richTextDialogFocusOverlay)
	entries := []ui.OverlayEntry{{
		Modal: true,
		Child: ui.Dialog{
			Child: ui.SelectionArea{Child: ui.RichText{SoftWrap: true, Spans: []ui.TextSpan{
				{Text: "Intro text before links. "},
				{Text: "one", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
					*w.Events = append(*w.Events, "one")
					s.SetState(func() { s.show = true })
				}},
				{Text: " "},
				{Text: "two", OnPressed: func(ctx ui.EventContext) {}, OnFocus: func() {
					*w.Events = append(*w.Events, "two")
					s.SetState(func() { s.show = true })
				}},
			}}},
			Actions: []ui.Widget{ui.Button{Label: "Close", OnPressed: func(ctx ui.EventContext) {}}},
		},
	}}
	if s.show {
		entries = append(entries, ui.OverlayEntry{Alignment: ui.BottomLeft, Child: ui.Text{Value: "https://example.com"}})
	}
	return ui.Overlay{Child: ui.Button{Label: "underlying", OnPressed: func(ctx ui.EventContext) {}}, Entries: entries}
}

func focusedDebugLabel(app *ui.App) string {
	for _, target := range app.DebugSnapshot().Focusables {
		if target.Focused {
			return target.Label
		}
	}
	return ""
}

type richTextFocusOverlay struct {
	Events *[]string
}

func (w richTextFocusOverlay) CreateState() ui.State {
	return &richTextFocusOverlayState{}
}

type richTextFocusOverlayState struct {
	ui.StateBase
	show bool
}

func (s *richTextFocusOverlayState) Build(ctx ui.BuildContext) ui.Widget {
	w := s.Widget().(richTextFocusOverlay)
	entries := []ui.OverlayEntry{}
	if s.show {
		entries = append(entries, ui.OverlayEntry{Alignment: ui.BottomLeft, Child: ui.Text{Value: "https://example.com"}})
	}
	return ui.Overlay{
		Child: ui.Align{Alignment: ui.TopLeft, Child: ui.RichText{Spans: []ui.TextSpan{
			{
				Text:      "docs",
				OnPressed: func(ctx ui.EventContext) {},
				OnFocus: func() {
					*w.Events = append(*w.Events, "focus")
					s.SetState(func() { s.show = true })
				},
				OnFocusExit: func() {
					*w.Events = append(*w.Events, "exit")
					s.SetState(func() { s.show = false })
				},
			},
		}}},
		Entries: entries,
	}
}

func TestRichTextFocusedSpanIsPainted(t *testing.T) {
	app := ui.NewApp(ui.RichText{Spans: []ui.TextSpan{
		{Text: "one", OnPressed: func(ctx ui.EventContext) {}},
		{Text: " "},
		{Text: "two", OnPressed: func(ctx ui.EventContext) {}},
	}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused first span underline = %#v, want double", got)
	}
	if got := p.Cell(4, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused second span underline = %#v, want single", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused first span underline = %#v, want single", got)
	}
	if got := p.Cell(4, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused second span underline = %#v, want double", got)
	}
}

func TestRichTextFocusedSpanClearsWhenFocusLeaves(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "docs", OnPressed: func(ctx ui.EventContext) {}},
		}},
		ui.Button{Label: "next"},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused span underline = %#v, want double", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused span underline = %#v, want single", got)
	}
}

func TestTextOnPressedActivatesWithMouseAndKeyboard(t *testing.T) {
	pressed := 0
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) { pressed++ }})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	if pressed != 2 {
		t.Fatalf("pressed = %d, want 2", pressed)
	}
}

func TestTextOnPressedSetsMouseShape(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape over text link = %q, want clickable", got)
	}
	app.Send(vaxis.Mouse{Col: 8, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	if got := app.MouseShape(); got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape outside text link = %q, want default", got)
	}
}

func TestTextOnPressedIsPaintedWhenFocused(t *testing.T) {
	app := ui.NewApp(ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused text underline = %#v, want double", got)
	}
}

func TestTextOnPressedClearsFocusUnderlineWhenFocusLeaves(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}},
		ui.Button{Label: "next"},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineDouble {
		t.Fatalf("focused text underline = %#v, want double", got)
	}
	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Pump(ui.Size{Width: 20, Height: 1})
	p = ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused text underline = %#v, want single", got)
	}
}

func TestTextOnPressedPaintsSingleUnderlineWhenUnfocused(t *testing.T) {
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "before"},
		ui.Text{Value: "docs", OnPressed: func(ctx ui.EventContext) {}},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 20, Height: 1})
	app.Paint(p)
	if got := p.Cell(10, 0).UnderlineStyle; got != ui.UnderlineSingle {
		t.Fatalf("unfocused interactive text underline = %#v, want single", got)
	}
}

func TestTextWithoutOnPressedDoesNotTakeFocus(t *testing.T) {
	pressed := false
	app := ui.NewApp(ui.Row(
		ui.Text{Value: "plain"},
		ui.Button{Label: "next", OnPressed: func(ctx ui.EventContext) { pressed = true }},
	))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	if !pressed {
		t.Fatal("expected focus to skip plain text")
	}
}
