package uitest

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type App struct {
	app     *ui.App
	size    ui.Size
	painter *ui.Painter
}

func New(root ui.Widget) *App {
	return &App{app: ui.NewApp(root), size: ui.Size{Width: 80, Height: 24}}
}

func (a *App) Pump(width, height int) {
	if width > 0 && height > 0 {
		a.size = ui.Size{Width: width, Height: height}
	}
	a.app.Pump(a.size)
	a.painter = ui.NewPainter(a.size)
	a.app.Paint(a.painter)
}

func (a *App) Send(ev ui.Event) {
	a.app.Send(ev)
}

func (a *App) Key(text string) {
	a.Send(vaxis.Key{Text: text, Keycode: firstRune(text)})
}

func (a *App) Enter() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyEnter})
}

func (a *App) Tab() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyTab})
}

func (a *App) ShiftTab() {
	a.Send(vaxis.Key{Keycode: vaxis.KeyTab, Modifiers: vaxis.ModShift})
}

func (a *App) Click(x, y int) {
	a.Send(vaxis.Mouse{Col: x, Row: y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
}

func (a *App) Cell(x, y int) ui.Cell {
	return a.painter.Cell(x, y)
}

func (a *App) ShouldQuit() bool {
	return a.app.ShouldQuit()
}

func (a *App) Contains(text string) bool {
	return strings.Contains(a.Text(), text)
}

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
