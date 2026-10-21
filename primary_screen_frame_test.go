package vaxis_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/widgets/term"
)

// A responding terminal, rather than a preloaded CPR: responses are derived
// from the emulator after it has consumed the actual bytes written by Vaxis.
type frameConsole struct {
	mu         sync.Mutex
	vt         *term.Model
	out        bytes.Buffer
	in         chan []byte
	done       chan struct{}
	once       sync.Once
	pending    []byte
	cols, rows int
	report     bool
}

func (c *frameConsole) Read(p []byte) (int, error) {
	if len(c.pending) == 0 {
		select {
		case c.pending = <-c.in:
		case <-c.done:
			return 0, io.EOF
		}
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

func (c *frameConsole) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.out.Write(p)
	c.vt.WriteString(string(p))
	if strings.Contains(string(p), "\x1b[c") {
		c.in <- []byte("\x1b[?1;2c")
	}
	if c.report && strings.Contains(string(p), "\x1b[6n") {
		visible := c.vt.Snapshot().CursorVisible
		c.vt.WriteString("\x1b[?25h")
		at := c.vt.Snapshot()
		if !visible {
			c.vt.WriteString("\x1b[?25l")
		}
		c.in <- []byte(fmt.Sprintf("\x1b[%d;%dR", at.CursorRow+1, at.CursorCol+1))
	}
	return len(p), nil
}

func (c *frameConsole) Fd() uintptr {
	return 0
}

func (c *frameConsole) SetRaw() error {
	return nil
}

func (c *frameConsole) Reset() error {
	return nil
}

func (c *frameConsole) Size() (int, int, int, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cols, c.rows, c.cols * 10, c.rows * 20, nil
}

func (c *frameConsole) Close() error {
	c.once.Do(func() { close(c.done) })
	return nil
}

func newFrameConsole(t *testing.T, mode vaxis.PrimaryScreenMode, height, rows int, prefix string, clear bool) (*vaxis.Vaxis, *frameConsole) {
	t.Helper()
	c := &frameConsole{vt: term.New(), in: make(chan []byte, 64), done: make(chan struct{}), cols: 40, rows: rows, report: true}
	c.vt.Resize(c.cols, rows)
	c.vt.Focus()
	c.vt.WriteString(prefix)
	vx, err := vaxis.New(vaxis.Options{
		NoSignals: true, DisableMouse: true, WithConsole: c,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{Mode: mode, RegionHeight: height, ClearOnExit: clear},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(vx.Close)
	return vx, c
}

func frameText(t *testing.T, vx *vaxis.Vaxis, text string) {
	t.Helper()
	vx.Window().Clear()
	vx.Window().Print(vaxis.Segment{Text: text})
	if err := vx.RenderFrame(); err != nil {
		t.Fatal(err)
	}
}

func TestPrimaryFrameStartsAfterPartialLine(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 12, "partial shell output", false)
	frameText(t, vx, "live one\nlive two")
	primaryRow(t, c.vt, 0, "partial shell output")
	primaryRow(t, c.vt, 1, "live one")
	primaryRow(t, c.vt, 2, "live two")
	if row, _, valid := vx.PrimaryScreenOrigin(); !valid || row != 1 {
		t.Fatalf("origin = %d, %v", row, valid)
	}
}

func TestPrimaryFrameNoCPRDoesNotPaint(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 12, "keep this", false)
	c.report = false
	vx.Window().Print(vaxis.Segment{Text: "must not paint"})
	if err := vx.RenderFrame(); err == nil {
		t.Fatal("missing positioning error")
	}
	primaryRow(t, c.vt, 0, "keep this")
	if _, _, valid := vx.PrimaryScreenOrigin(); valid {
		t.Fatal("origin valid after timeout")
	}
	c.report = true
	frameText(t, vx, "retry")
	primaryRow(t, c.vt, 1, "retry")
}

func TestPrimaryFrameSplitSettlesAndKeepsFooter(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimarySplit, 2, 6, "shell\r\n", true)
	frameText(t, vx, "footer one\nfooter two")
	for i := 0; i < 8; i++ {
		vx.AppendString(fmt.Sprintf("log %d\n", i))
		frameText(t, vx, "footer one\nfooter two")
		want := min(2+i, 4)
		if row, _, valid := vx.PrimaryScreenOrigin(); !valid || row != want {
			t.Fatalf("frame %d origin=%d valid=%v want=%d", i, row, valid, want)
		}
		primaryRow(t, c.vt, want, "footer one")
		primaryRow(t, c.vt, want-1, fmt.Sprintf("log %d", i))
	}
	vx.Close()
	primaryRow(t, c.vt, 3, "log 7")
	primaryRow(t, c.vt, 4, "")
}

func TestPrimaryFrameMainAndFullHeightSplit(t *testing.T) {
	for _, mode := range []vaxis.PrimaryScreenMode{vaxis.PrimaryMain, vaxis.PrimarySplit} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			vx, c := newFrameConsole(t, mode, 3, 3, "shell\r\n", false)
			frameText(t, vx, "one\ntwo\nthree")
			if row, _, valid := vx.PrimaryScreenOrigin(); !valid || row != 0 {
				t.Fatalf("origin=%d valid=%v", row, valid)
			}
			vx.AppendString("log\n")
			frameText(t, vx, "one\ntwo\nthree")
			primaryRow(t, c.vt, 0, "one")
			primaryRow(t, c.vt, 2, "three")
		})
	}
}

func TestPrimaryFrameResizeCursorAndResume(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 12, "shell one\r\nshell two\r\n", false)
	frameText(t, vx, "old one\nold two")
	vx.Window().ShowCursor(3, 1, vaxis.CursorBeam)
	frameText(t, vx, "old one\nold two")
	if at := c.vt.Snapshot(); at.CursorRow != 3 || at.CursorCol != 3 {
		t.Fatalf("cursor=%+v", at)
	}
	vx.SetPrimaryScreenRegionHeight(4)
	frameText(t, vx, "new one\nnew two\nnew three\nnew four")
	primaryRow(t, c.vt, 0, "shell one")
	primaryRow(t, c.vt, 5, "new four")
	if err := vx.Suspend(); err != nil {
		t.Fatal(err)
	}
	c.vt.WriteString("external\r\n")
	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	frameText(t, vx, "resumed")
	primaryRow(t, c.vt, 6, "external")
	primaryRow(t, c.vt, 7, "resumed")
}

func TestPrimaryFrameTinySplitPreservesHistory(t *testing.T) {
	for _, rows := range []int{1, 2, 3, 4} {
		t.Run(fmt.Sprint(rows), func(t *testing.T) {
			vx, c := newFrameConsole(t, vaxis.PrimarySplit, min(rows, 2), rows, "", true)
			frameText(t, vx, "footer")
			for i := 0; i < 8; i++ {
				vx.AppendString(fmt.Sprintf("log %d\n", i))
				frameText(t, vx, "footer")
			}
			primaryRow(t, c.vt, max(0, rows-2), "footer")
			// Inspect actual terminal history, not just emitted output bytes.
			for i := 0; i < 20; i++ {
				c.vt.Update(vaxis.Key{Keycode: vaxis.KeyPgUp, Modifiers: vaxis.ModShift})
			}
			primaryRow(t, c.vt, 0, "log 0")
		})
	}
}

func TestPrimaryFrameResizeReflowPreservesPrefix(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 12, "shell-prefix-abcdefghij\r\n", false)
	frameText(t, vx, "old first row\nold second row")
	c.mu.Lock()
	c.cols = 10
	c.vt.Resize(10, 12)
	c.mu.Unlock()
	vx.Resize(vaxis.Resize{Cols: 10, Rows: 12})
	frameText(t, vx, "new first\nnew second")
	primaryRow(t, c.vt, 0, "shell-pref")
	primaryRow(t, c.vt, 1, "ix-abcdefg")
	primaryRow(t, c.vt, 2, "hij")
	primaryRow(t, c.vt, 3, "new first")
	primaryRow(t, c.vt, 4, "new second")
}

func TestPrimaryFrameCloseWhileSuspendedDoesNotEraseExternalOutput(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 1, 6, "shell\r\n", true)
	frameText(t, vx, "temporary")
	if err := vx.Suspend(); err != nil {
		t.Fatal(err)
	}
	c.vt.WriteString("external")
	vx.Close()
	vx.Close()
	primaryRow(t, c.vt, 0, "shell")
	primaryRow(t, c.vt, 1, "external")
}

func TestPrimaryFramePhysicalResizeWithUnchangedSurface(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 6, "shell\r\n", false)
	frameText(t, vx, "old one\nold two")
	_, generation, _ := vx.PrimaryScreenOrigin()
	c.mu.Lock()
	c.rows = 8
	c.vt.Resize(40, 8)
	c.mu.Unlock()
	vx.Resize(vaxis.Resize{Cols: 40, Rows: 8})
	if _, _, valid := vx.PrimaryScreenOrigin(); valid {
		t.Fatal("physical resize retained a stale position")
	}
	frameText(t, vx, "new one\nnew two")
	row, next, valid := vx.PrimaryScreenOrigin()
	if !valid || row != 1 || next == generation {
		t.Fatalf("origin=%d generation=%d valid=%v", row, next, valid)
	}
	primaryRow(t, c.vt, 0, "shell")
	primaryRow(t, c.vt, 1, "new one")
	primaryRow(t, c.vt, 2, "new two")
}

func TestPrimaryFrameMousePlacementGeneration(t *testing.T) {
	vx, c := newFrameConsole(t, vaxis.PrimaryInline, 2, 12, "shell\r\n", false)
	frameText(t, vx, "live")
	row, generation, valid := vx.PrimaryScreenOrigin()
	if !valid || row != 1 || generation == 0 {
		t.Fatalf("initial origin=%d generation=%d valid=%v", row, generation, valid)
	}
	c.in <- []byte("\x1b[<0;3;2M\x1b[<0;3;1m")
	for _, wantRow := range []int{1, 0} {
		for {
			select {
			case event := <-vx.Events():
				if mouse, ok := event.(vaxis.Mouse); ok {
					if mouse.Row != wantRow || mouse.Col != 2 || mouse.SurfaceGeneration != generation {
						t.Fatalf("mouse=%+v, want row=%d generation=%d", mouse, wantRow, generation)
					}
					goto received
				}
			case <-time.After(time.Second):
				t.Fatal("missing mouse event")
			}
		}
	received:
	}
	vx.AppendString("output\n")
	frameText(t, vx, "live")
	row, next, valid := vx.PrimaryScreenOrigin()
	if !valid || row != 2 || next == generation {
		t.Fatalf("moved origin=%d generation=%d valid=%v", row, next, valid)
	}
}
