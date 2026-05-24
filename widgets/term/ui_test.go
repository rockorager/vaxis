package term

import (
	"testing"

	"go.rockorager.dev/vaxis/ansi"
	"go.rockorager.dev/vaxis/ui"
)

func TestSnapshotReturnsVisibleCells(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(ansi.Print{Grapheme: "h", Width: 1})
	vt.update(ansi.Print{Grapheme: "i", Width: 1})

	snapshot := vt.Snapshot()
	if len(snapshot.Cells) == 0 {
		t.Fatal("snapshot had no cells")
	}
	if got := snapshot.Cells[0].Cell.Grapheme + snapshot.Cells[1].Cell.Grapheme; got != "hi" {
		t.Fatalf("snapshot text = %q, want hi", got)
	}
}

func TestTerminalWidgetPaintsModelSnapshot(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(ansi.Print{Grapheme: "o", Width: 1})
	vt.update(ansi.Print{Grapheme: "k", Width: 1})
	app := ui.NewApp(Terminal{Model: vt})
	app.Pump(ui.Size{Width: 5, Height: 2})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 2})
	app.Paint(p)

	if got := p.Cell(0, 0).Grapheme + p.Cell(1, 0).Grapheme; got != "ok" {
		t.Fatalf("painted text = %q, want ok", got)
	}
}

func TestTerminalWidgetHidesCursorWhenUnfocused(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.Focus()
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "other", OnPressed: func(ui.EventContext) {}},
		Terminal{Model: vt},
	))
	app.Pump(ui.Size{Width: 5, Height: 2})
	p := ui.NewPainter(ui.Size{Width: 5, Height: 2})
	app.Paint(p)

	if cursor, ok := p.Cursor(); ok {
		t.Fatalf("unfocused terminal painted cursor %#v", cursor)
	}
}

func TestTerminalWidgetUsesUIEventCallback(t *testing.T) {
	vt := New()
	var got ui.Event
	app := ui.NewApp(Terminal{
		Model: vt,
		OnEvent: func(ctx ui.EventContext, ev ui.Event) ui.EventResult {
			got = ev
			return ui.EventHandled
		},
	})
	app.Pump(ui.Size{Width: 5, Height: 2})
	vt.dispatchEvent(EventTitle("hello"))
	app.Pump(ui.Size{Width: 5, Height: 2})

	if title, ok := got.(EventTitle); !ok || title != "hello" {
		t.Fatalf("event = %#v, want EventTitle hello", got)
	}
}
