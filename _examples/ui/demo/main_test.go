package main

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func TestDemoExampleNavigationAndControls(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	if !app.Contains("Vaxis UI demo") || !app.Contains("Home") || !app.Contains("ticks: 0") {
		t.Fatalf("initial frame missing home content: %q", app.Text())
	}

	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Text layout") || !app.Contains("center aligned text") {
		t.Fatalf("text page missing content: %q", app.Text())
	}

	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Controls") || !app.Contains("count: 0") {
		t.Fatalf("controls page missing content: %q", app.Text())
	}

	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Enter()
	app.Pump(90, 20)
	if !app.Contains("count: -1") {
		t.Fatalf("decrement button did not update count: %q", app.Text())
	}
}

func TestDemoExampleCanNavigateAwayFromFocusedButton(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)

	app.Tab()
	app.Tab()
	app.Tab()
	app.Pump(90, 20)
	app.Key("p")
	app.Pump(90, 20)
	if !app.Contains("Text layout") {
		t.Fatalf("expected to navigate away from focused controls button: %q", app.Text())
	}
}

func TestDemoExampleQuitShortcut(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(80, 20)
	app.Key("q")
	if !app.ShouldQuit() {
		t.Fatal("expected q shortcut to request quit")
	}
}
