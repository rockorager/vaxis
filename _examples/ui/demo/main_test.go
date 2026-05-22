package main

import (
	"strings"
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
	if !app.Contains("Controls") || !app.Contains("count: 0") || !app.Contains("Stack base") || !app.Contains("checkbox: unchecked") || !app.Contains("radio: compact") {
		t.Fatalf("controls page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Lists") || !app.Contains("target             status") || !app.Contains("deploy target 101") || !app.Contains("Variable-height messages") || !app.Contains("echo a message") {
		t.Fatalf("lists page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Table") || !app.Contains("running tests") || !app.Contains("blocked on approval") {
		t.Fatalf("table page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Animation") || !app.Contains("status: running") {
		t.Fatalf("animation page missing content: %q", app.Text())
	}
	app.Key("n")
	app.Pump(90, 20)
	if !app.Contains("Theme") || !app.Contains("Semantic colors") || !app.Contains("Theme.Background") {
		t.Fatalf("theme page missing content: %q", app.Text())
	}
	app.Key("p")
	app.Pump(90, 20)
	app.Key("p")
	app.Pump(90, 20)
	app.Key("p")
	app.Pump(90, 20)
	app.Key("p")
	app.Pump(90, 20)
	app.Click(3, 17)
	app.Pump(90, 20)
	if !app.Contains("count: -1") {
		t.Fatalf("decrement button did not update count: %q", app.Text())
	}
}

func TestDemoExampleTabButtonsActivateFocusedPage(t *testing.T) {
	app := ui.NewApp(Demo{})
	app.Pump(ui.Size{Width: 90, Height: 20})

	app.Send(vaxis.Key{Keycode: vaxis.KeyTab})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
	app.Pump(ui.Size{Width: 90, Height: 20})
	p := ui.NewPainter(ui.Size{Width: 90, Height: 20})
	app.Paint(p)
	if !painterTextContains(p, "Text layout") {
		t.Fatalf("focused Text tab did not activate text page: %q", painterText(p))
	}
	if got := focusedDebugLabel(app); !strings.Contains(got, "Text") {
		t.Fatalf("focused tab after activation = %q, want Text tab", got)
	}
}

func TestDemoExampleCanNavigateAwayFromFocusedButton(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)
	app.Key("n")
	app.Pump(90, 20)

	app.Click(3, 14)
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

	app.Click(2, 9)
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

	app.Click(2, 14)
	app.Pump(90, 20)
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

func focusedDebugLabel(app *ui.App) string {
	for _, target := range app.DebugSnapshot().Focusables {
		if target.Focused {
			return target.Label
		}
	}
	return ""
}

func painterTextContains(p *ui.Painter, text string) bool {
	return strings.Contains(painterText(p), text)
}

func painterText(p *ui.Painter) string {
	var b strings.Builder
	for _, cell := range p.Cells() {
		b.WriteString(cell.Grapheme)
	}
	return b.String()
}

func TestDemoExampleQuitShortcut(t *testing.T) {
	app := uitest.New(Demo{})
	app.Pump(80, 20)
	app.Key("q")
	if !app.ShouldQuit() {
		t.Fatal("expected q shortcut to request quit")
	}
}
