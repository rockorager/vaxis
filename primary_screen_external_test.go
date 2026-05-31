package vaxis_test

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/widgets/term"
)

type primaryConsole struct {
	mu   sync.Mutex
	out  bytes.Buffer
	in   strings.Reader
	cols int
	rows int
}

func newPrimaryConsole(cols, rows int) *primaryConsole {
	return &primaryConsole{cols: cols, rows: rows, in: *strings.NewReader("\x1b[?1;2c")}
}

func (c *primaryConsole) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.in.Len() == 0 {
		return 0, io.EOF
	}
	return c.in.Read(p)
}

func (c *primaryConsole) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.Write(p)
}

func (c *primaryConsole) Fd() uintptr {
	return 0
}

func (c *primaryConsole) SetRaw() error {
	return nil
}

func (c *primaryConsole) Reset() error {
	return nil
}

func (c *primaryConsole) Size() (int, int, int, int, error) {
	return c.cols, c.rows, 0, 0, nil
}

func (c *primaryConsole) Close() error {
	return nil
}

func (c *primaryConsole) ResetOutput() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.out.Reset()
}

func (c *primaryConsole) Output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.String()
}

func TestPrimaryScreenAppendAndRegionResumeShellBelowOutput(t *testing.T) {
	console := newPrimaryConsole(80, 10)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	console.ResetOutput()

	vt := term.New()
	vt.Resize(80, 10)
	vt.WriteString("shell prompt\r\n")

	for i := 1; i <= 6; i++ {
		vx.AppendString("tick line\n")
		win := vx.Window()
		win.Clear()
		win.Print(vaxis.Segment{Text: "live region\nframe"})
		vx.Render()
		vt.WriteString(console.Output())
		console.ResetOutput()
	}

	vx.Close()
	vt.WriteString(console.Output())
	vt.WriteString("shell prompt 2\r\n")

	rows := vt.Rows()
	if got := strings.TrimRight(rows[len(rows)-2], " "); got != "shell prompt 2" {
		t.Fatalf("row before cursor = %q, want shell prompt 2; rows=%#v", got, rows)
	}
}

func TestPrimaryScreenAppendPushesRenderedRegionDown(t *testing.T) {
	console := newPrimaryConsole(80, 10)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	console.ResetOutput()

	vt := term.New()
	vt.Resize(80, 10)
	vt.WriteString("shell prompt\r\n")

	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prompt"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "region"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}

	vx.AppendString("append 1\n")
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())

	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prompt"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "append 1"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(2), " "), "region"; got != want {
		t.Fatalf("row 2 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
}

func TestPrimaryScreenShellPromptAfterExitFollowsRegion(t *testing.T) {
	console := newPrimaryConsole(80, 10)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	console.ResetOutput()

	vt := term.New()
	vt.Resize(80, 10)
	vt.WriteString("shell prompt\r\n")

	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region 1\nregion 2\nregion 3"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prompt"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "region 1"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(2), " "), "region 2"; got != want {
		t.Fatalf("row 2 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(3), " "), "region 3"; got != want {
		t.Fatalf("row 3 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}

	vx.Close()
	vt.WriteString(console.Output())
	vt.WriteString("shell prompt 2\r\n")

	if got, want := strings.TrimRight(vt.RowString(4), " "), "shell prompt 2"; got != want {
		t.Fatalf("row 4 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
}

func TestPrimaryScreenResizeWithoutAppendKeepsRegionInPlace(t *testing.T) {
	console := newPrimaryConsole(20, 8)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	console.ResetOutput()

	vt := term.New()
	vt.Resize(20, 8)
	vt.WriteString("shell prompt\r\n")
	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	console.cols = 30
	vx.Resize(vaxis.Resize{Cols: 30, Rows: 8})
	vt.Resize(30, 8)
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prompt"; got != want {
		t.Fatalf("wider row 0 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "region"; got != want {
		t.Fatalf("wider row 1 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}

	console.cols = 15
	vx.Resize(vaxis.Resize{Cols: 15, Rows: 8})
	vt.Resize(15, 8)
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())

	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prompt"; got != want {
		t.Fatalf("narrower row 0 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "region"; got != want {
		t.Fatalf("narrower row 1 = %q, want %q; rows=%#v", got, want, vt.Rows())
	}
}

func TestPrimaryScreenResizeWrappingRegionDoesNotScroll(t *testing.T) {
	console := newPrimaryConsole(20, 8)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	console.ResetOutput()

	vt := term.New()
	vt.Resize(20, 8)
	vt.WriteString("shell prompt\r\n")
	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region text"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	console.cols = 6
	vx.Resize(vaxis.Resize{Cols: 6, Rows: 8})
	vt.Resize(6, 8)
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region text"})
	vx.Render()
	vt.WriteString(console.Output())

	rows := vt.Rows()
	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "prompt"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(2), " "), "region"; got != want {
		t.Fatalf("row 2 = %q, want %q; rows=%#v", got, want, rows)
	}
	for i, row := range rows[3:] {
		if strings.TrimRight(row, " ") != "" {
			t.Fatalf("row %d should stay empty after non-wrapping region paint: rows=%#v", i+3, rows)
		}
	}
}

func TestPrimaryScreenAppendWrapAfterResizeKeepsRegionAfterAppend(t *testing.T) {
	console := newPrimaryConsole(20, 8)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	console.ResetOutput()

	vt := term.New()
	vt.Resize(20, 8)
	vt.WriteString("shell prompt\r\n")
	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	console.cols = 10
	vx.Resize(vaxis.Resize{Cols: 10, Rows: 8})
	vt.Resize(10, 8)
	vx.AppendString("append wraps\n")
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())

	rows := vt.Rows()
	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prom"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "pt"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(2), " "), "append wra"; got != want {
		t.Fatalf("row 2 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(3), " "), "ps"; got != want {
		t.Fatalf("row 3 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(4), " "), "region"; got != want {
		t.Fatalf("row 4 = %q, want %q; rows=%#v", got, want, rows)
	}
}

func TestPrimaryScreenResizeAfterAppendKeepsRegionAfterWrappedAppend(t *testing.T) {
	console := newPrimaryConsole(20, 8)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		WithConsole:  console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	console.ResetOutput()

	vt := term.New()
	vt.Resize(20, 8)
	vt.WriteString("shell prompt\r\n")
	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	vx.AppendString("append wraps\n")
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()

	console.cols = 10
	vx.Resize(vaxis.Resize{Cols: 10, Rows: 8})
	vt.Resize(10, 8)
	win = vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: "region"})
	vx.Render()
	vt.WriteString(console.Output())

	rows := vt.Rows()
	if got, want := strings.TrimRight(vt.RowString(0), " "), "shell prom"; got != want {
		t.Fatalf("row 0 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(1), " "), "pt"; got != want {
		t.Fatalf("row 1 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(2), " "), "append wra"; got != want {
		t.Fatalf("row 2 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(3), " "), "ps"; got != want {
		t.Fatalf("row 3 = %q, want %q; rows=%#v", got, want, rows)
	}
	if got, want := strings.TrimRight(vt.RowString(4), " "), "region"; got != want {
		t.Fatalf("row 4 = %q, want %q; rows=%#v", got, want, rows)
	}
}
