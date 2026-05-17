package main

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestCounterExampleKeyboard(t *testing.T) {
	app := uitest.New(Counter{})
	app.Pump(20, 5)
	if !app.Contains("count: 0") {
		t.Fatalf("initial frame missing count: %q", app.Text())
	}
	if got := app.Cell(6, 1).Character.Grapheme; got != "c" {
		t.Fatalf("centered count starts at cell (6,1) = %q, want c", got)
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(20, 5)
	if !app.Contains("count: -1") {
		t.Fatalf("decremented frame missing count: %q", app.Text())
	}

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(20, 5)
	if !app.Contains("count: 0") {
		t.Fatalf("incremented frame missing count: %q", app.Text())
	}
}

func TestCounterExampleMouseAndQuit(t *testing.T) {
	app := uitest.New(Counter{})
	app.Pump(20, 5)
	app.Send(vaxis.Mouse{Col: 10, Row: 2, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(20, 5)
	if !app.Contains("count: 1") {
		t.Fatalf("mouse increment frame missing count: %q", app.Text())
	}

	app.Send(vaxis.Key{Text: "q", Keycode: 'q'})
	if !app.ShouldQuit() {
		t.Fatal("expected q shortcut to request quit")
	}
}
