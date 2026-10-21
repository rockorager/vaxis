package vaxis_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/widgets/term"
)

// Run the actual TTY backend in a child process; the terminal emulator supplies
// protocol replies over a real PTY rather than the in-memory Console adapter.
func TestPrimaryScreenPTYChild(t *testing.T) {
	if os.Getenv("VAXIS_PRIMARY_PTY_CHILD") != "1" {
		return
	}
	fmt.Print("shell prefix\r\npartial output")
	vx, err := vaxis.New(vaxis.Options{NoSignals: true, PrimaryScreen: &vaxis.PrimaryScreenOptions{
		Mode: vaxis.PrimarySplit, RegionHeight: 2, ClearOnExit: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer vx.Close()
	paint := func() {
		vx.Window().Clear()
		vx.Window().Print(vaxis.Segment{Text: "live footer\ninput: abc"})
		vx.Window().ShowCursor(9, 1, vaxis.CursorBeam)
		if err := vx.RenderFrame(); err != nil {
			t.Fatal(err)
		}
	}
	paint()
	for event := range vx.Events() {
		if key, ok := event.(vaxis.Key); ok {
			switch key.String() {
			case "a":
				for i := 0; i < 10; i++ {
					vx.AppendString(fmt.Sprintf("output %d\n", i))
					paint()
				}
			case "q":
				vx.Close()
				fmt.Print("shell restored")
				return
			}
		}
	}
}

func TestPrimaryScreenPTY(t *testing.T) {
	vt := term.New()
	vt.Focus()
	closed := make(chan error, 1)
	vt.Attach(func(event vaxis.Event) {
		if event, ok := event.(term.EventClosed); ok {
			closed <- event.Error
		}
	})
	cmd := exec.Command(os.Args[0], "-test.run=^TestPrimaryScreenPTYChild$")
	cmd.Env = append(os.Environ(), "VAXIS_PRIMARY_PTY_CHILD=1")
	if err := vt.StartWithSize(cmd, 40, 6); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(vt.Close)
	waitRow := func(row int, want string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if strings.TrimRight(vt.RowString(row), " ") == want {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("row %d never became %q; screen=%q", row, want, vt.Rows())
	}
	waitRow(2, "live footer")
	primaryRow(t, vt, 0, "shell prefix")
	primaryRow(t, vt, 1, "partial output")
	t.Logf("startup: %q", vt.Rows())
	vt.Update(vaxis.Key{Keycode: 'a', Text: "a"})
	waitRow(3, "output 9")
	waitRow(4, "live footer")
	waitRow(5, "input: abc")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		at := vt.Snapshot()
		if at.CursorVisible && at.CursorRow == 5 && at.CursorCol == 9 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if at := vt.Snapshot(); !at.CursorVisible || at.CursorRow != 5 || at.CursorCol != 9 {
		t.Fatalf("input cursor = %+v", at)
	}
	t.Logf("pinned: %q", vt.Rows())
	vt.Update(vaxis.Key{Keycode: 'q', Text: "q"})
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("child did not restore terminal and exit")
	}
	primaryRow(t, vt, 4, "shell restoredPASS")
	t.Logf("restored: %q", vt.Rows())
	for i := 0; i < 10; i++ {
		vt.Update(vaxis.Key{Keycode: vaxis.KeyPgUp, Modifiers: vaxis.ModShift})
	}
	primaryRow(t, vt, 0, "shell prefix")
	primaryRow(t, vt, 1, "partial output")
	primaryRow(t, vt, 2, "output 0")
}
