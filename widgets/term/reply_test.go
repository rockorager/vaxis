package term

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func newReplyTestModel(t *testing.T, opts ...Option) (*Model, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	vt := New(opts...)
	vt.pty = w
	vt.startReplyWorker()
	t.Cleanup(func() {
		vt.stopReplyWorker()
		_ = w.Close()
		_ = r.Close()
	})
	return vt, r
}

func readReply(t *testing.T, r *os.File, n int) string {
	t.Helper()
	ch := make(chan string, 1)
	go func() {
		buf := make([]byte, n)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			ch <- ""
			return
		}
		ch <- string(buf)
	}()
	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for terminal reply")
		return ""
	}
}

func TestReplyQueuePreservesOrder(t *testing.T) {
	vt, r := newReplyTestModel(t)

	started := make(chan struct{})
	release := make(chan struct{})
	vt.enqueueReply(func(ctx context.Context) (string, bool) {
		close(started)
		select {
		case <-release:
			return "first", true
		case <-ctx.Done():
			return "", false
		}
	})
	vt.enqueueReplyString("second")

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first reply did not start")
	}
	close(release)

	if got := readReply(t, r, len("firstsecond")); got != "firstsecond" {
		t.Fatalf("expected replies in FIFO order, got %q", got)
	}
}

func TestReplyQueueDropsTimedOutReplyAndContinues(t *testing.T) {
	oldTimeout := termReplyTimeout
	termReplyTimeout = 10 * time.Millisecond
	defer func() { termReplyTimeout = oldTimeout }()

	vt, r := newReplyTestModel(t)
	vt.enqueueReply(func(ctx context.Context) (string, bool) {
		<-ctx.Done()
		return "", false
	})
	vt.enqueueReplyString("second")

	if got := readReply(t, r, len("second")); got != "second" {
		t.Fatalf("expected timed out reply to be dropped, got %q", got)
	}
}
