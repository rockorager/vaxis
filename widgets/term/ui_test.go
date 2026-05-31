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

func TestWriteStringParsesTerminalOutput(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.WriteString("hi\x1b[2;3Hok")

	snapshot := vt.Snapshot()
	cellAt := func(col, row int) string {
		for _, cell := range snapshot.Cells {
			if cell.Col == col && cell.Row == row {
				return cell.Cell.Grapheme
			}
		}
		return ""
	}
	if got := cellAt(0, 0) + cellAt(1, 0); got != "hi" {
		t.Fatalf("first text = %q, want hi", got)
	}
	if got := cellAt(2, 1) + cellAt(3, 1); got != "ok" {
		t.Fatalf("cursor-positioned text = %q, want ok", got)
	}
}

func TestWriteParsesTerminalOutput(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	n, err := vt.Write([]byte("a\x1b[3Gb"))
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if n != len("a\x1b[3Gb") {
		t.Fatalf("Write count = %d, want %d", n, len("a\x1b[3Gb"))
	}

	if got, want := vt.String(), "a b  "; got != want {
		t.Fatalf("screen = %q, want %q", got, want)
	}
}

func TestRowsReturnVisibleTextByRow(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.WriteString("one\r\ntwo")

	if got, want := vt.RowString(0), "one  "; got != want {
		t.Fatalf("row 0 = %q, want %q", got, want)
	}
	if got, want := vt.RowString(1), "two  "; got != want {
		t.Fatalf("row 1 = %q, want %q", got, want)
	}
	if got := vt.RowString(2); got != "" {
		t.Fatalf("out-of-range row = %q, want empty", got)
	}
	rows := vt.Rows()
	if len(rows) != 2 {
		t.Fatalf("Rows len = %d, want 2", len(rows))
	}
	if rows[0] != "one  " || rows[1] != "two  " {
		t.Fatalf("Rows = %#v, want one/two", rows)
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
