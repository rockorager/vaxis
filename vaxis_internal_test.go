package vaxis

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

type queryResponseWriter struct {
	ch   chan string
	resp string
}

func (w queryResponseWriter) Write(p []byte) (int, error) {
	w.ch <- w.resp
	return len(p), nil
}

func TestKnownOSCColorResponsesDoNotBlockWhenChannelFull(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc4:  true,
			osc10: true,
			osc11: true,
		},
		chColor: make(chan string, 1),
		chFg:    make(chan string, 1),
		chBg:    make(chan string, 1),
	}
	vx.chColor <- "4;8;rgb:ff/ff/ff"
	vx.chFg <- "10;rgb:ff/ff/ff"
	vx.chBg <- "11;rgb:ff/ff/ff"

	parser := ansi.NewParser(strings.NewReader("\x1b]4;8;rgb:01/02/03\x07\x1b]10;rgb:01/02/03\x07\x1b]11;rgb:01/02/03\x07"), ansi.ParserModeOutput)
	defer parser.Close()

	done := make(chan struct{})
	go func() {
		for seq := range parser.Next() {
			vx.handleSequence(seq)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("known OSC color responses blocked with full channels")
	}
}

func TestQueryColorContextUsesFreshResponse(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc4: true,
		},
		chColor: make(chan string, 1),
	}
	vx.chColor <- "4;8;rgb:ff/ff/ff"
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w: queryResponseWriter{
			ch:   vx.chColor,
			resp: "4;8;rgb:01/02/03",
		},
		vx: vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if got := vx.QueryColorContext(ctx, IndexColor(8)); got != RGBColor(1, 2, 3) {
		t.Fatalf("expected fresh indexed color, got %#v", got)
	}
}

func TestRemoveImageRemovesFuturePlacements(t *testing.T) {
	vx := &Vaxis{}
	img := &Sixel{vx: vx, id: 7}
	vx.graphicsLast = []*placement{{id: 7}}
	vx.graphicsNext = []*placement{{id: 7}, {id: 8}}

	vx.RemoveImage(img)

	if len(vx.graphicsLast) != 1 {
		t.Fatalf("graphicsLast len = %d, want 1", len(vx.graphicsLast))
	}
	if len(vx.graphicsNext) != 1 || vx.graphicsNext[0].id != 8 {
		t.Fatalf("graphicsNext = %#v, want only id 8", vx.graphicsNext)
	}
}

func TestQueryColorContextTimesOut(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc4: true,
		},
		chColor: make(chan string, 1),
	}
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w:   io.Discard,
		vx:  vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	if got := vx.QueryColorContext(ctx, IndexColor(8)); got != Color(0) {
		t.Fatalf("expected default color after timeout, got %#v", got)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("QueryColorContext took too long to time out: %s", elapsed)
	}
}

func TestQueryForegroundContextUsesFreshResponse(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc10: true,
		},
		chFg: make(chan string, 1),
	}
	vx.chFg <- "10;rgb:ff/ff/ff"
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w: queryResponseWriter{
			ch:   vx.chFg,
			resp: "10;rgb:01/02/03",
		},
		vx: vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if got := vx.QueryForegroundContext(ctx); got != RGBColor(1, 2, 3) {
		t.Fatalf("expected fresh foreground color, got %#v", got)
	}
}

func TestQueryForegroundContextTimesOut(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc10: true,
		},
		chFg: make(chan string, 1),
	}
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w:   io.Discard,
		vx:  vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	if got := vx.QueryForegroundContext(ctx); got != Color(0) {
		t.Fatalf("expected default color after timeout, got %#v", got)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("QueryForegroundContext took too long to time out: %s", elapsed)
	}
}

func TestQueryBackgroundContextUsesFreshResponse(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc11: true,
		},
		chBg: make(chan string, 1),
	}
	vx.chBg <- "11;rgb:ff/ff/ff"
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w: queryResponseWriter{
			ch:   vx.chBg,
			resp: "11;rgb:01/02/03",
		},
		vx: vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if got := vx.QueryBackgroundContext(ctx); got != RGBColor(1, 2, 3) {
		t.Fatalf("expected fresh background color, got %#v", got)
	}
}

func TestQueryBackgroundContextTimesOut(t *testing.T) {
	vx := &Vaxis{
		caps: capabilities{
			osc11: true,
		},
		chBg: make(chan string, 1),
	}
	vx.tw = &writer{
		buf: bytes.NewBuffer(nil),
		w:   io.Discard,
		vx:  vx,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	if got := vx.QueryBackgroundContext(ctx); got != Color(0) {
		t.Fatalf("expected default color after timeout, got %#v", got)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("QueryBackgroundContext took too long to time out: %s", elapsed)
	}
}
