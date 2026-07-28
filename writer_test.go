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

func TestRenderIsDeferredWhileNotVisible(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.visibility.known = true
	vx.visibility.potentiallyVisible = false
	vx.refresh = true
	vx.cursorNext = cursorState{row: 0, col: 1, style: CursorBlock, visible: true}
	vx.screenNext.setCell(0, 0, Cell{
		Character: Character{Grapheme: "a", Width: 1},
	})

	vx.Render()

	if out.Len() != 0 || vx.tw.Len() != 0 {
		t.Fatalf("hidden render produced output: terminal=%q frame=%q", out.String(), vx.tw.buf.String())
	}
	if got := vx.screenLast.cell(0, 0).Grapheme; got != "" {
		t.Fatalf("hidden render committed screen cell %q", got)
	}
	if vx.cursorLast != (cursorState{}) {
		t.Fatalf("hidden render committed cursor state %#v", vx.cursorLast)
	}
	if !vx.refresh {
		t.Fatal("hidden render cleared pending refresh")
	}
	if vx.renders != 0 {
		t.Fatalf("hidden render count = %d, want 0", vx.renders)
	}

	vx.setVisibility(true)
	vx.Render()

	if !strings.Contains(out.String(), "a") {
		t.Fatalf("visible render output = %q, want pending cell", out.String())
	}
	if got := vx.screenLast.cell(0, 0).Grapheme; got != "a" {
		t.Fatalf("visible render committed screen cell %q, want a", got)
	}
	if vx.cursorLast != vx.cursorNext {
		t.Fatalf("visible render cursor = %#v, want %#v", vx.cursorLast, vx.cursorNext)
	}
	if vx.refresh {
		t.Fatal("visible render did not clear refresh")
	}
	if vx.renders != 1 {
		t.Fatalf("visible render count = %d, want 1", vx.renders)
	}
}

func TestDisableVisibilityReportsDoesNotSuppressRender(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.visibility.disabled = true
	vx.setVisibility(false)
	vx.screenNext.setCell(0, 0, Cell{
		Character: Character{Grapheme: "a", Width: 1},
	})

	vx.Render()

	if !strings.Contains(out.String(), "a") {
		t.Fatalf("render output = %q, want pending cell", out.String())
	}
}

func TestReturningToVisibleDoesNotForceRefresh(t *testing.T) {
	vx := &Vaxis{
		visibility: visibilityState{known: true},
	}

	vx.setVisibility(true)

	if vx.refresh {
		t.Fatal("visible transition forced a full refresh")
	}
}

func TestRenderUsesPlainSGRForSingleUnderlineAfterDouble(t *testing.T) {
	var out bytes.Buffer
	vx := newWriterTestVaxis(&out)
	vx.caps.styledUnderlines = true
	vx.screenNext.setCell(0, 0, Cell{
		Character: Character{Grapheme: "x", Width: 1},
		Style:     Style{UnderlineStyle: UnderlineDouble},
	})
	vx.screenNext.setCell(1, 0, Cell{
		Character: Character{Grapheme: "y", Width: 1},
		Style:     Style{UnderlineStyle: UnderlineSingle},
	})

	vx.render()
	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "\x1b[4:2mx\x1b[4my") {
		t.Fatalf("render output = %q, want double underline followed by plain single underline", got)
	}
	if strings.Contains(got, "\x1b[4:1m") {
		t.Fatalf("render output used extended SGR for single underline: %q", got)
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
