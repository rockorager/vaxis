package vaxis_test

import (
	"strings"
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/widgets/term"
)

// Regression coverage for the legacy inline Render entry point. Checked
// positioning, including partial-line startup, is covered by RenderFrame tests.
func inlineRenderPrimary(t *testing.T, height int, prefix string) (*vaxis.Vaxis, *primaryConsole, *term.Model) {
	t.Helper()
	console := newPrimaryConsole(40, 12)
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse:  true,
		NoSignals:     true,
		WithConsole:   console,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{RegionHeight: height},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(vx.Close)
	console.ResetOutput()
	vt := term.New()
	vt.Resize(40, 12)
	vt.WriteString(prefix)
	return vx, console, vt
}

func inlineRenderFrame(vx *vaxis.Vaxis, console *primaryConsole, vt *term.Model, text string) {
	win := vx.Window()
	win.Clear()
	win.Print(vaxis.Segment{Text: text})
	vx.Render()
	vt.WriteString(console.Output())
	console.ResetOutput()
}

func primaryRow(t *testing.T, vt *term.Model, row int, want string) {
	t.Helper()
	if got := strings.TrimRight(vt.RowString(row), " "); got != want {
		t.Errorf("row %d = %q, want %q; screen=%q", row, got, want, vt.Rows())
	}
}

func TestInlineRenderHiddenCursorControl(t *testing.T) {
	vx, console, vt := inlineRenderPrimary(t, 2, "shell one\r\nshell two\r\n")
	inlineRenderFrame(vx, console, vt, "old one\nold two")
	inlineRenderFrame(vx, console, vt, "new one\nnew two")
	primaryRow(t, vt, 0, "shell one")
	primaryRow(t, vt, 1, "shell two")
	primaryRow(t, vt, 2, "new one")
	primaryRow(t, vt, 3, "new two")
}

func TestInlineRenderVisibleCursorPosition(t *testing.T) {
	vx, console, vt := inlineRenderPrimary(t, 2, "shell one\r\nshell two\r\nshell three\r\n")
	inlineRenderFrame(vx, console, vt, "live one\nlive two")
	vx.Window().ShowCursor(3, 1, vaxis.CursorBeam)
	inlineRenderFrame(vx, console, vt, "live one\nlive two")
	// Mark the actual hardware cursor position in the emulator. The region's
	// origin is row 3, so its local (3, 1) is physical (3, 4).
	vt.WriteString("@")
	primaryRow(t, vt, 1, "shell two")
	primaryRow(t, vt, 4, "liv@ two")
}

func TestInlineRenderVisibleCursorThenRepaint(t *testing.T) {
	vx, console, vt := inlineRenderPrimary(t, 2, "shell one\r\nshell two\r\nshell three\r\n")
	inlineRenderFrame(vx, console, vt, "old one\nold two")
	vx.Window().ShowCursor(3, 1, vaxis.CursorBeam)
	inlineRenderFrame(vx, console, vt, "old one\nold two")
	inlineRenderFrame(vx, console, vt, "new one\nnew two")
	primaryRow(t, vt, 0, "shell one")
	primaryRow(t, vt, 2, "shell three")
	primaryRow(t, vt, 3, "new one")
	primaryRow(t, vt, 4, "new two")
}

func TestInlineRenderGrowPreservesPrefix(t *testing.T) {
	vx, console, vt := inlineRenderPrimary(t, 2, "shell one\r\nshell two\r\n")
	inlineRenderFrame(vx, console, vt, "old one\nold two")
	vx.SetPrimaryScreenRegionHeight(4)
	inlineRenderFrame(vx, console, vt, "new one\nnew two\nnew three\nnew four")
	primaryRow(t, vt, 0, "shell one")
	primaryRow(t, vt, 1, "shell two")
	primaryRow(t, vt, 2, "new one")
	primaryRow(t, vt, 5, "new four")
}

func TestInlineRenderResumePreservesExternalOutput(t *testing.T) {
	vx, console, vt := inlineRenderPrimary(t, 2, "shell one\r\n")
	inlineRenderFrame(vx, console, vt, "old one\nold two")
	if err := vx.Suspend(); err != nil {
		t.Fatal(err)
	}
	vt.WriteString(console.Output())
	console.ResetOutput()
	vt.WriteString("external one\r\nexternal two\r\n")
	if err := vx.Resume(); err != nil {
		t.Fatal(err)
	}
	inlineRenderFrame(vx, console, vt, "new one\nnew two")
	primaryRow(t, vt, 3, "external one")
	primaryRow(t, vt, 4, "external two")
	primaryRow(t, vt, 5, "new one")
	primaryRow(t, vt, 6, "new two")
}
