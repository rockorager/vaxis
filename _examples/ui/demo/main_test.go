package main

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
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
	if !app.Contains("Text layout") || !app.Contains("row 1") {
		t.Fatalf("text page missing content: %q", app.Text())
	}

	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Controls") || !app.Contains("count: 0") {
		t.Fatalf("controls page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Lists") || !app.Contains("target             status") || !app.Contains("deploy target 101") {
		t.Fatalf("lists page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Animation") || !app.Contains("status: running") {
		t.Fatalf("animation page missing content: %q", app.Text())
	}
	app.Key("p")
	app.Pump(90, 20)
	app.Key("p")
	app.Pump(90, 20)
	app.Tab()
	app.Tab()
	app.Tab()
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

func TestDemoExampleLeavesArrowsForTextField(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)

	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Key("a")
	app.Pump(90, 20)
	app.Key("b")
	app.Pump(90, 20)
	app.Send(uitestKeyLeft())
	app.Key("x")
	app.Pump(90, 20)
	if !app.Contains("axb") {
		t.Fatalf("expected left arrow to move text field cursor, got %q", app.Text())
	}
	if !app.Contains("Controls") {
		t.Fatalf("left arrow should not navigate pages: %q", app.Text())
	}
}

func TestDemoExampleTextAreaAcceptsMultilineInput(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)

	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Tab()
	app.Key("alpha")
	app.Enter()
	app.Key("beta")
	app.Pump(90, 20)
	if !app.Contains("alpha") || !app.Contains("beta") {
		t.Fatalf("expected textarea to show multiline input: %q", app.Text())
	}
}

func TestDemoExampleTextPageScrolls(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("row 1") || app.Contains("row 6") {
		t.Fatalf("expected text page scroll view to start at top: %q", app.Text())
	}

	app.Send(vaxis.Mouse{Col: 4, Row: 10, Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress})
	app.Pump(90, 20)
	if app.Contains("row 1") || !app.Contains("row 6") {
		t.Fatalf("expected mouse wheel to scroll text page viewport: %q", app.Text())
	}
}

func uitestKeyLeft() ui.Event {
	return vaxis.Key{Keycode: vaxis.KeyLeft}
}

func TestDemoExampleQuitShortcut(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(80, 20)
	app.Key("q")
	if !app.ShouldQuit() {
		t.Fatal("expected q shortcut to request quit")
	}
}
