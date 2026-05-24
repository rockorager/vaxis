package ui_test

import (
	"testing"

	"go.rockorager.dev/vaxis/ui"
)

func TestCursorWidgetReportsGlobalPaintPosition(t *testing.T) {
	app := ui.NewApp(ui.Padding(ui.Insets{Left: 2, Top: 1}, ui.Cursor{
		Col:   3,
		Row:   0,
		Shape: ui.CursorBeam,
		Child: ui.Text{Value: "cursor"},
	}))
	app.Pump(ui.Size{Width: 10, Height: 3})
	p := ui.NewPainter(ui.Size{Width: 10, Height: 3})
	app.Paint(p)

	cursor, ok := p.Cursor()
	if !ok {
		t.Fatal("expected cursor")
	}
	if cursor.Col != 5 || cursor.Row != 1 || cursor.Shape != ui.CursorBeam {
		t.Fatalf("cursor = %#v, want col=5 row=1 shape=beam", cursor)
	}
}

func TestCursorWidgetClipsOutOfBoundsCursor(t *testing.T) {
	app := ui.NewApp(ui.Cursor{Col: 20, Row: 0, Shape: ui.CursorBlock, Child: ui.Text{Value: "x"}})
	app.Pump(ui.Size{Width: 3, Height: 1})
	p := ui.NewPainter(ui.Size{Width: 3, Height: 1})
	app.Paint(p)
	if cursor, ok := p.Cursor(); ok {
		t.Fatalf("cursor = %#v, want hidden", cursor)
	}
}
