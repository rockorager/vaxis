package vaxis

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// suspendTestConsole is a Console that answers the DA1 query Vaxis uses to
// detect startup completion and to unblock its input parser on Suspend.
type suspendTestConsole struct {
	r *io.PipeReader
	w *io.PipeWriter

	mu      sync.Mutex
	out     bytes.Buffer
	setRaws int
}

func newSuspendTestConsole() *suspendTestConsole {
	r, w := io.Pipe()
	return &suspendTestConsole{r: r, w: w}
}

func (c *suspendTestConsole) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *suspendTestConsole) Write(p []byte) (int, error) {
	c.mu.Lock()
	c.out.Write(p)
	c.mu.Unlock()
	if bytes.Contains(p, []byte(primaryAttributes)) {
		go func() { _, _ = c.w.Write([]byte("\x1b[?62c")) }()
	}
	return len(p), nil
}

func (c *suspendTestConsole) Fd() uintptr {
	return 0
}

func (c *suspendTestConsole) SetRaw() error {
	c.mu.Lock()
	c.setRaws++
	c.mu.Unlock()
	return nil
}

func (c *suspendTestConsole) Reset() error {
	return nil
}

func (c *suspendTestConsole) Size() (int, int, int, int, error) {
	return 20, 5, 0, 0, nil
}

func (c *suspendTestConsole) Close() error {
	_ = c.r.Close()
	return c.w.Close()
}

func (c *suspendTestConsole) output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.String()
}

func (c *suspendTestConsole) rawCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setRaws
}

func newSuspendTestVaxis(t *testing.T) (*Vaxis, *suspendTestConsole) {
	t.Helper()
	console := newSuspendTestConsole()
	vx, err := New(Options{WithConsole: console, NoSignals: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { withinTimeout(t, "Close", vx.Close) })
	return vx, console
}

func withinTimeout(t *testing.T, name string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s did not return", name)
	}
}

func TestSuspendTwiceAndCloseAfterSuspendDoNotHang(t *testing.T) {
	vx, _ := newSuspendTestVaxis(t)
	withinTimeout(t, "Suspend", func() { _ = vx.Suspend() })
	withinTimeout(t, "second Suspend", func() { _ = vx.Suspend() })
	withinTimeout(t, "Close after Suspend", vx.Close)
}

func TestResumeWithoutSuspendDoesNotReopenTerminal(t *testing.T) {
	vx, console := newSuspendTestVaxis(t)

	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	if got := console.rawCount(); got != 1 {
		t.Fatalf("SetRaw calls after Resume without Suspend = %d, want 1", got)
	}

	withinTimeout(t, "Suspend", func() { _ = vx.Suspend() })
	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	if got := console.rawCount(); got != 2 {
		t.Fatalf("SetRaw calls after Suspend and Resume = %d, want 2", got)
	}
}

func TestRenderWhileSuspendedWritesNothing(t *testing.T) {
	vx, console := newSuspendTestVaxis(t)

	withinTimeout(t, "Suspend", func() { _ = vx.Suspend() })
	before := console.output()
	vx.Window().Print(Segment{Text: "hidden"})
	vx.Render()
	if after := console.output(); after != before {
		t.Fatalf("Render while suspended wrote %q", strings.TrimPrefix(after, before))
	}

	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	vx.Window().Print(Segment{Text: "shown"})
	vx.Render()
	if after := console.output(); !strings.Contains(strings.TrimPrefix(after, before), "shown") {
		t.Fatal("Render after Resume did not draw the frame")
	}
}

func TestMouseShapeIsRestoredAfterResume(t *testing.T) {
	vx, console := newSuspendTestVaxis(t)

	pointer := tparm(mouseShape, MouseShapeClickable)
	vx.SetMouseShape(MouseShapeClickable)
	vx.Render()
	if !strings.Contains(console.output(), pointer) {
		t.Fatal("initial Render did not set the mouse shape")
	}

	withinTimeout(t, "Suspend", func() { _ = vx.Suspend() })
	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	before := console.output()
	vx.SetMouseShape(MouseShapeClickable)
	vx.Render()
	if !strings.Contains(strings.TrimPrefix(console.output(), before), pointer) {
		t.Fatal("Render after Resume did not restore the mouse shape")
	}
}
