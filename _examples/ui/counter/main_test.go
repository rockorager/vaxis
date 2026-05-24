package main

import (
	"testing"

	"go.rockorager.dev/vaxis/ui/uitest"
)

func TestCounterExampleKeyboard(t *testing.T) {
	app := uitest.New(Counter{})
	app.Pump(20, 5)
	if !app.Contains("count: 0") {
		t.Fatalf("initial frame missing count: %q", app.Text())
	}
	if got := app.Cell(6, 1).Character.Grapheme; got != "c" {
		t.Fatalf("padded count starts at cell (6,1) = %q, want c", got)
	}
	if got := app.Cell(8, 2).Character.Grapheme; got != "+" {
		t.Fatalf("increment button label at cell (8,2) = %q, want +", got)
	}

	app.Enter()
	app.Pump(20, 5)
	if !app.Contains("count: -1") {
		t.Fatalf("decremented frame missing count: %q", app.Text())
	}

	app.Tab()
	app.Enter()
	app.Pump(20, 5)
	if !app.Contains("count: 0") {
		t.Fatalf("incremented frame missing count: %q", app.Text())
	}
}

func TestCounterExampleMouseAndQuit(t *testing.T) {
	app := uitest.New(Counter{})
	app.Pump(20, 5)
	app.Click(8, 2)
	app.Pump(20, 5)
	if !app.Contains("count: 1") {
		t.Fatalf("mouse increment frame missing count: %q", app.Text())
	}

	app.Key("q")
	if !app.ShouldQuit() {
		t.Fatal("expected q shortcut to request quit")
	}
}
