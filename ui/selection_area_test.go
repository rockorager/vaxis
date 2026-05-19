package ui_test

import (
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

func TestSelectionAreaSelectsTextWithMouse(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "abcd"}})
	app.Pump(ui.Size{Width: 10, Height: 1})

	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
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

func TestSelectionAreaCopiesSelectedText(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "abcd"}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bc" {
		t.Fatalf("copies = %#v, want bc", backend.copies)
	}
}

func TestSelectionAreaDoubleClickCopiesWord(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 12, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "alpha beta"}}), backend, ui.NewFrameScheduler(time.Second/60))
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

func TestSelectionAreaTripleClickCopiesLine(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 12, Height: 2})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "alpha beta\ngamma"}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	mouse := vaxis.Mouse{Col: 2, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}
	for i := 0; i < 3; i++ {
		mouse.EventType = vaxis.EventPress
		runner.HandleEvent(mouse, now)
		mouse.EventType = vaxis.EventRelease
		runner.HandleEvent(mouse, now)
	}
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "alpha beta\n" {
		t.Fatalf("copies = %#v, want alpha beta\\n", backend.copies)
	}
}

func TestSelectionAreaSelectsRichTextAcrossSpans(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.RichText{Spans: []ui.TextSpan{
		{Text: "ab"},
		{Text: "cd"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bc" {
		t.Fatalf("copies = %#v, want bc", backend.copies)
	}
}

func TestSelectionAreaDoubleClickSelectsRichTextWordAcrossSpans(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 12, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.RichText{Spans: []ui.TextSpan{
		{Text: "al"},
		{Text: "pha beta"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	mouse := vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventRelease
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventPress
	runner.HandleEvent(mouse, now)
	mouse.EventType = vaxis.EventRelease
	runner.HandleEvent(mouse, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "alpha" {
		t.Fatalf("copies = %#v, want alpha", backend.copies)
	}
}

func TestSelectionAreaUsesLocalTextCoordinates(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 12, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Padding(ui.Symmetric(2, 0), ui.SelectionArea{Child: ui.Text{Value: "abcd"}})), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 5, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 5, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bc" {
		t.Fatalf("copies = %#v, want bc", backend.copies)
	}
}

func TestSelectionAreaSelectAllCopiesText(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "abcd"}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "abcd" {
		t.Fatalf("copies = %#v, want abcd", backend.copies)
	}
}

func TestSelectionAreaMouseSelectionCopiesClippedVisibleText(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 3, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowClip,
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "abc" {
		t.Fatalf("copies = %#v, want abc", backend.copies)
	}
}

func TestSelectionAreaMouseSelectionCopiesEllipsisVisibleText(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 3, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowEllipsis,
		MaxLines: 1,
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab" {
		t.Fatalf("copies = %#v, want ab", backend.copies)
	}
}

func TestSelectionAreaSelectAllCopiesHiddenText(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 3, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowEllipsis,
		MaxLines: 1,
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "abcdef" {
		t.Fatalf("copies = %#v, want abcdef", backend.copies)
	}
}

func TestSelectionAreaSelectAllPaintsOverflowVisibleText(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowVisible,
	}})
	app.Pump(ui.Size{Width: 3, Height: 1})

	app.Send(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Send(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 3, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 6, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(5, 0).Background; got != want {
		t.Fatalf("overflow selected background = %#v, want %#v", got, want)
	}
}

func TestSelectionAreaMouseSelectionCopiesVisibleMaxLines(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 8, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "ab\ncd",
		MaxLines: 1,
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab" {
		t.Fatalf("copies = %#v, want ab", backend.copies)
	}
}

func TestSelectionAreaSoftWrapDoesNotCopySyntheticNewline(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 3, Height: 2})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		SoftWrap: true,
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "abcdef" {
		t.Fatalf("copies = %#v, want abcdef", backend.copies)
	}
}

func TestSelectionAreaSelectsAcrossTextWidgets(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 2})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 2})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first line background = %#v, want %#v", got, want)
	}
	if got := p.Cell(1, 1).Background; got != want {
		t.Fatalf("selected second line background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 1).Background; got == want {
		t.Fatalf("unselected second line cell background = %#v, should not be selection background", got)
	}
}

func TestSelectionAreaCopiesAcrossTextWidgets(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 2})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bcd\nef" {
		t.Fatalf("copies = %#v, want bcd\\nef", backend.copies)
	}
}

func TestSelectionAreaSelectsAcrossTextWidgetsInReverse(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 2})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bcd\nef" {
		t.Fatalf("copies = %#v, want bcd\\nef", backend.copies)
	}
}

func TestSelectionAreaSelectAllCopiesAllTextWidgets(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 2})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "ab"},
		ui.Text{Value: "cd"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "ab\ncd" {
		t.Fatalf("copies = %#v, want ab\\ncd", backend.copies)
	}
}

func TestSelectionContainerDisabledExcludesSubtreeFromCopy(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 16, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "aa"},
		ui.SelectionContainer{Disabled: true, Child: ui.Text{Value: "bb"}},
		ui.Text{Value: "cc"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "aa\ncc" {
		t.Fatalf("copies = %#v, want aa\\ncc", backend.copies)
	}
}

func TestSelectionContainerDisabledDoesNotStartSelection(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.SelectionContainer{
		Disabled: true,
		Child:    ui.Text{Value: "abcd"},
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 0 {
		t.Fatalf("copies = %#v, want none", backend.copies)
	}
}

func TestSelectionContainerEnabledIsTransparent(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 10, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.SelectionContainer{
		Child: ui.Text{Value: "abcd"},
	}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "bc" {
		t.Fatalf("copies = %#v, want bc", backend.copies)
	}
}

func TestSelectionAreaSelectAllSkipsTextFieldContents(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "before\nafter" {
		t.Fatalf("copies = %#v, want before\\nafter", backend.copies)
	}
}

func TestSelectionAreaFocusedTextFieldHandlesSelectAllAndCopy(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now)
	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "field" {
		t.Fatalf("copies = %#v, want field", backend.copies)
	}
}

func TestSelectionAreaMousePressInTextFieldDoesNotStartOuterSelection(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 3})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 5, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion}, now)
	runner.HandleEvent(vaxis.Mouse{Col: 5, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 0 {
		t.Fatalf("copies = %#v, want none", backend.copies)
	}
}

func TestSelectionAreaSelectAllSkipsTextAreaContents(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 4})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "before\nafter" {
		t.Fatalf("copies = %#v, want before\\nafter", backend.copies)
	}
}

func TestSelectionAreaFocusedTextAreaHandlesSelectAllAndCopy(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 4})
	runner := ui.NewRunner(ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now)
	runner.HandleEvent(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl}, now)
	runner.HandleEvent(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl}, now)
	if len(backend.copies) != 1 || backend.copies[0] != "area" {
		t.Fatalf("copies = %#v, want area", backend.copies)
	}
}
