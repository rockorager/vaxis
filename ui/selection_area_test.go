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
