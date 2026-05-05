package vaxis

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

type backgroundResponseWriter struct {
	ch   chan string
	resp string
}

func (w backgroundResponseWriter) Write(p []byte) (int, error) {
	w.ch <- w.resp
	return len(p), nil
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
		w: backgroundResponseWriter{
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
