package vaxis

import (
	"bytes"
	"strings"
	"testing"
)

func newWriterTestVaxis(out *bytes.Buffer) *Vaxis {
	vx := &Vaxis{
		screenNext: newScreen(),
		screenLast: newScreen(),
		charCache:  make(map[string]int),
	}
	vx.screenNext.resize(2, 1)
	vx.screenLast.resize(2, 1)
	vx.tw = &writer{
		buf:      bytes.NewBuffer(make([]byte, 0, 256)),
		terminal: &terminalWriter{w: out},
		vx:       vx,
	}
	return vx
}

func TestRenderFrameAlwaysHidesCursorBeforeDrawing(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.cursorLast.visible = false
	vx.cursorNext = cursorState{row: 0, col: 1, style: CursorBlock, visible: true}
	vx.screenNext.setCell(0, 0, Cell{
		Character: Character{Grapheme: "a", Width: 1},
	})

	vx.render()
	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	hide := hideCursorSeq
	show := vx.showCursor()
	if !strings.Contains(got, hide) {
		t.Fatalf("render output did not hide cursor: %q", got)
	}
	if !strings.Contains(got, show) {
		t.Fatalf("render output did not restore visible cursor: %q", got)
	}
	if strings.Index(got, hide) > strings.Index(got, show) {
		t.Fatalf("cursor was shown before it was hidden: %q", got)
	}
}

func TestControlWriteBypassesRenderFrame(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.caps.synchronizedUpdate = true

	if _, err := vx.tw.WriteControlString("control"); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if got != "control" {
		t.Fatalf("control write = %q, want raw control bytes", got)
	}
	if vx.tw.Len() != 0 {
		t.Fatalf("frame buffer len = %d, want 0", vx.tw.Len())
	}
	if strings.Contains(got, hideCursorSeq) || strings.Contains(got, syncUpdateStartSeq) {
		t.Fatalf("control write included render frame sequences: %q", got)
	}
}

func TestRenderWriteUsesFrameWrapping(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.caps.synchronizedUpdate = true

	if _, err := vx.tw.WriteString("render"); err != nil {
		t.Fatal(err)
	}
	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		hideCursorSeq,
		syncUpdateStartSeq,
		"render",
		sgrReset,
		syncUpdateEndSeq,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("render output %q missing %q", got, want)
		}
	}
}

func TestRenderFrameLeavesCursorHiddenWhenNextFrameCursorHidden(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.cursorLast.visible = true
	vx.cursorNext = cursorState{row: 0, col: 1, style: CursorBlock, visible: false}
	vx.screenNext.setCell(0, 0, Cell{
		Character: Character{Grapheme: "a", Width: 1},
	})

	vx.render()
	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, hideCursorSeq) {
		t.Fatalf("render output did not hide cursor: %q", got)
	}
	if strings.Contains(got, showCursorSeq) {
		t.Fatalf("render output showed hidden cursor: %q", got)
	}
}

func TestFlushDoesNotShowCursorForHiddenCursorPositionChange(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.cursorLast = cursorState{row: 0, col: 0, style: CursorBlock, visible: false}
	vx.cursorNext = cursorState{row: 0, col: 1, style: CursorBlock, visible: false}

	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}

	if got := out.String(); got != "" {
		t.Fatalf("hidden cursor position change wrote %q, want no output", got)
	}
}
