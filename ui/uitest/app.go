// Package uitest provides small helpers for testing ui widgets without a terminal.
package uitest

import (
	"strings"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ui"
)

// App wraps a ui.App with a fixed-size painter for tests.
type App struct {
	app     *ui.App
	size    ui.Size
	painter *ui.Painter
}

// New creates a test app mounted with root.
func New(root ui.Widget) *App {
	return &App{app: ui.NewApp(root), size: ui.Size{Width: 80, Height: 24}}
}

// Pump rebuilds, lays out, and paints the app at width and height.
func (a *App) Pump(width, height int) {
	if width > 0 && height > 0 {
		a.size = ui.Size{Width: width, Height: height}
	}
	a.app.Pump(a.size)
	a.painter = ui.NewPainter(a.size)
	a.app.Paint(a.painter)
}

// Send dispatches ev to the app.
func (a *App) Send(ev ui.Event) {
	a.app.Send(ev)
}

// Key sends a printable key event.
func (a *App) Key(text string) {
	a.Send(vaxis.Key{Text: text, Keycode: firstRune(text)})
}

// Enter sends an Enter key event.
func (a *App) Enter() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
}

// Tab sends a Tab key event.
func (a *App) Tab() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyTab})
}

// ShiftTab sends a Shift+Tab key event.
func (a *App) ShiftTab() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift})
}

// Click sends a left mouse press at x,y.
func (a *App) Click(x, y int) {
	a.Send(vaxis.Mouse{Col: x, Row: y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
}

// Cell returns the painted cell at x,y.
func (a *App) Cell(x, y int) ui.Cell {
	return a.painter.Cell(x, y)
}

// ShouldQuit reports whether the wrapped app has requested quit.
func (a *App) ShouldQuit() bool {
	return a.app.ShouldQuit()
}

// Contains reports whether the painted text contains text.
func (a *App) Contains(text string) bool {
	return strings.Contains(a.Text(), text)
}

// Text returns the painter cell graphemes as a flat string.
func (a *App) Text() string {
	if a.painter == nil {
		return ""
	}
	var b strings.Builder
	for _, cell := range a.painter.Cells() {
		b.WriteString(cell.Grapheme)
	}
	return b.String()
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}
