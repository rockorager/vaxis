package vaxis

import (
	"bytes"
	"strings"
	"testing"
)

type primaryTestTTY struct {
	bytes.Buffer
	size Resize
}

func (t *primaryTestTTY) Fd() uintptr {
	return 0
}

func (t *primaryTestTTY) SetRaw() error {
	return nil
}

func (t *primaryTestTTY) Reset() error {
	return nil
}

func (t *primaryTestTTY) Size() (Resize, error) {
	return t.size, nil
}

func (t *primaryTestTTY) StartInput(*Vaxis) error {
	return nil
}

func (t *primaryTestTTY) StopInput() error {
	return nil
}

func (t *primaryTestTTY) Close() error {
	return nil
}

func newPrimaryTestVaxis(cols, rows, regionHeight int) (*Vaxis, *primaryTestTTY) {
	tty := &primaryTestTTY{size: Resize{Cols: cols, Rows: rows}}
	vx := &Vaxis{
		tty:           tty,
		screenNext:    newScreen(),
		screenLast:    newScreen(),
		primaryScreen: &primaryScreen{regionHeight: regionHeight},
		winSize:       tty.size,
		ready:         true,
		charCache:     make(map[string]int),
	}
	vx.tw = newWriter(vx)
	winCols, winRows := vx.surfaceSize(tty.size)
	vx.screenNext.resize(winCols, winRows)
	vx.screenLast.resize(winCols, winRows)
	return vx, tty
}

func TestPrimaryScreenWindowUsesRegionSize(t *testing.T) {
	vx, _ := newPrimaryTestVaxis(10, 4, 2)

	win := vx.Window()
	if got, want := win.Width, 10; got != want {
		t.Fatalf("window width = %d, want %d", got, want)
	}
	if got, want := win.Height, 2; got != want {
		t.Fatalf("window height = %d, want %d", got, want)
	}
}

func TestPrimaryScreenSetRegionHeightResizesWindow(t *testing.T) {
	vx, _ := newPrimaryTestVaxis(10, 4, 1)

	vx.SetPrimaryScreenRegionHeight(3)

	win := vx.Window()
	if got, want := win.Width, 10; got != want {
		t.Fatalf("window width = %d, want %d", got, want)
	}
	if got, want := win.Height, 3; got != want {
		t.Fatalf("window height = %d, want %d", got, want)
	}
}

func TestPrimaryScreenRenderAppendsThenRendersRegion(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	vx.AppendString("log line\n")
	vx.Window().Print(Segment{Text: "status"})

	vx.Render()
	out := tty.String()
	if strings.Contains(out, decset(alternateScreen)) {
		t.Fatalf("primary render entered alternate screen: %q", out)
	}
	if !strings.Contains(out, "log line\r\n") {
		t.Fatalf("primary render missing append output: %q", out)
	}
	if !strings.Contains(out, "status") {
		t.Fatalf("primary render missing region output: %q", out)
	}
	if appendIndex, regionIndex := strings.Index(out, "log line\r\n"), strings.Index(out, "status"); appendIndex < 0 || regionIndex < 0 || appendIndex > regionIndex {
		t.Fatalf("append output should precede region output: %q", out)
	}
}

func TestPrimaryScreenAppendOnlyDoesNotPaintBlankRegion(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	vx.AppendString("log line\n")

	vx.Render()
	out := tty.String()
	if !strings.Contains(out, "log line\r\n") {
		t.Fatalf("primary render missing append output: %q", out)
	}
	if strings.Contains(out, strings.Repeat(" ", 12)) {
		t.Fatalf("append-only render should not paint blank region spaces: %q", out)
	}
}

func TestPrimaryScreenRendersStyledEmptyCellAsSpace(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	vx.Window().SetStyle(0, 0, Style{Attribute: AttrReverse})

	vx.Render()
	out := tty.String()
	if !strings.Contains(out, reverseSet+" "+sgrReset) {
		t.Fatalf("primary render should paint styled empty cells as spaces: %q", out)
	}
}

func TestPrimaryScreenTrimsOnlyDefaultTrailingBlanks(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	win := vx.Window()
	win.Print(Segment{Text: "x"})
	win.SetStyle(2, 0, Style{Attribute: AttrReverse})

	vx.Render()
	out := tty.String()
	if !strings.Contains(out, reverseSet+" "+sgrReset) {
		t.Fatalf("primary render should retain trailing styled blank: %q", out)
	}
	if strings.Contains(out, reverseSet+" "+sgrReset+" ") {
		t.Fatalf("primary render should trim default blanks after styled blank: %q", out)
	}
}

func TestPrimaryScreenAppendWriterQueuesAppendOutput(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	n, err := vx.AppendWriter().Write([]byte("log line\n"))
	if err != nil {
		t.Fatalf("AppendWriter Write error = %v", err)
	}
	if n != len("log line\n") {
		t.Fatalf("AppendWriter Write count = %d, want %d", n, len("log line\n"))
	}

	vx.Render()
	out := tty.String()
	if !strings.Contains(out, "log line\r\n") {
		t.Fatalf("primary render missing append writer output: %q", out)
	}
}

func TestPrimaryScreenExitLeavesAppendOnlyCursorPosition(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	vx.AppendString("log line\n")
	vx.Render()
	before := tty.String()
	vx.exitPrimaryScreen()

	out := tty.String()
	if strings.Contains(out[len(before):], "\x1b[4;1H\n") {
		t.Fatalf("append-only primary exit should not force a new line below region: %q", out)
	}
}

func TestPrimaryScreenExitLeavesRenderedRegionCursorPosition(t *testing.T) {
	vx, tty := newPrimaryTestVaxis(12, 4, 1)
	vx.Window().Print(Segment{Text: "status"})
	vx.Render()
	before := tty.String()
	vx.exitPrimaryScreen()

	out := tty.String()
	if strings.Contains(out[len(before):], "\x1b[4;1H\n") {
		t.Fatalf("primary exit should not force cursor to bottom row: %q", out)
	}
}

func TestPrimaryScreenAppendPanicsOutsidePrimaryMode(t *testing.T) {
	vx := &Vaxis{}
	defer func() {
		if recover() == nil {
			t.Fatal("Append did not panic outside primary mode")
		}
	}()
	vx.AppendString("log\n")
}
