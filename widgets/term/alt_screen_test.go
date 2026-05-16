package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestMode47AltScreenRetainsContent(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{47}, '?'))
	if !vt.mode.smcup {
		t.Fatal("mode 47 did not enter alternate screen")
	}
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("initial alt screen = %q, want %q", got, want)
	}

	printText(vt, "2B")
	if got, want := vt.String(), "  2B \n     "; got != want {
		t.Fatalf("alt screen after write = %q, want %q", got, want)
	}

	vt.decrst(testCSI('l', []uint32{47}, '?'))
	if vt.mode.smcup {
		t.Fatal("mode 47 did not leave alternate screen")
	}
	if got, want := vt.String(), "1A   \n     "; got != want {
		t.Fatalf("primary screen = %q, want %q", got, want)
	}

	vt.decset(testCSI('h', []uint32{47}, '?'))
	if got, want := vt.String(), "  2B \n     "; got != want {
		t.Fatalf("mode 47 did not retain alternate content: got %q want %q", got, want)
	}
}

func TestMode47CopiesCursorStateBothDirections(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	primaryFG := vaxis.RGBColor(0xff, 0, 0x7f)
	altFG := vaxis.RGBColor(0, 0xff, 0)
	vt.cursor.Foreground = primaryFG
	vt.cursor.Style.Foreground = primaryFG
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.decset(testCSI('h', []uint32{47}, '?'))
	if vt.cursor.row != 1 || vt.cursor.col != 2 {
		t.Fatalf("alternate cursor = %d,%d, want copied 1,2", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.cursor.Foreground; got != primaryFG {
		t.Fatalf("alternate cursor fg = %v, want %v", got, primaryFG)
	}

	vt.cursor.Foreground = altFG
	vt.cursor.Style.Foreground = altFG
	vt.cursor.row = 0
	vt.cursor.col = 4

	vt.decrst(testCSI('l', []uint32{47}, '?'))
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("primary cursor = %d,%d, want copied 0,4", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.cursor.Foreground; got != altFG {
		t.Fatalf("primary cursor fg = %v, want %v", got, altFG)
	}
}

func TestMode1047ClearsAltScreenOnExit(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{1047}, '?'))
	printText(vt, "2B")
	vt.decrst(testCSI('l', []uint32{1047}, '?'))
	vt.decset(testCSI('h', []uint32{1047}, '?'))

	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("mode 1047 did not clear alternate content: got %q want %q", got, want)
	}
}

func TestMode1047CopiesCursorStateBothDirections(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	primaryFG := vaxis.RGBColor(0xff, 0, 0x7f)
	altFG := vaxis.RGBColor(0, 0xff, 0)
	vt.cursor.Foreground = primaryFG
	vt.cursor.Style.Foreground = primaryFG
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.decset(testCSI('h', []uint32{1047}, '?'))
	if vt.cursor.row != 1 || vt.cursor.col != 2 {
		t.Fatalf("alternate cursor = %d,%d, want copied 1,2", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.cursor.Foreground; got != primaryFG {
		t.Fatalf("alternate cursor fg = %v, want %v", got, primaryFG)
	}

	vt.cursor.Foreground = altFG
	vt.cursor.Style.Foreground = altFG
	vt.cursor.row = 0
	vt.cursor.col = 4

	vt.decrst(testCSI('l', []uint32{1047}, '?'))
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("primary cursor = %d,%d, want copied 0,4", vt.cursor.row, vt.cursor.col)
	}
	if got := vt.cursor.Foreground; got != altFG {
		t.Fatalf("primary cursor fg = %v, want %v", got, altFG)
	}
}

func TestMode1049RestoresPrimaryCursorAndClearsAltOnEntry(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{1049}, '?'))
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("initial 1049 alt screen = %q, want %q", got, want)
	}
	printText(vt, "2B")

	vt.decrst(testCSI('l', []uint32{1049}, '?'))
	printText(vt, "C")
	if got, want := vt.String(), "1AC  \n     "; got != want {
		t.Fatalf("primary screen after 1049 restore = %q, want %q", got, want)
	}

	vt.decset(testCSI('h', []uint32{1049}, '?'))
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("mode 1049 did not clear alternate content on entry: got %q want %q", got, want)
	}
}

func TestMode1049RepeatedEnableDoesNotClobberPrimarySavedCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{1049}, '?'))
	vt.cursor.row = 1
	vt.cursor.col = 4
	vt.decset(testCSI('h', []uint32{1049}, '?'))

	vt.decrst(testCSI('l', []uint32{1049}, '?'))
	vt.update(testPrint("C"))

	if got, want := vt.String(), "1AC  \n     "; got != want {
		t.Fatalf("primary screen after repeated 1049 restore = %q, want %q", got, want)
	}
}

func TestAltScreenClearsViewportOffset(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.decset(testCSI('h', []uint32{1049}, '?'))

	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset = %d, want 0", got)
	}
}

func TestMode1048SavesAndRestoresCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.cursor.row = 1
	vt.cursor.col = 4
	vt.lastCol = true

	vt.decset(testCSI('h', []uint32{1048}, '?'))
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false
	vt.decrst(testCSI('l', []uint32{1048}, '?'))

	if vt.cursor.row != 1 || vt.cursor.col != 4 {
		t.Fatalf("restored cursor = %d,%d, want 1,4", vt.cursor.row, vt.cursor.col)
	}
	if !vt.lastCol {
		t.Fatal("pending wrap was not restored")
	}
}

func TestMode1048UsesActiveScreenCursorState(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.decset(testCSI('h', []uint32{47}, '?'))
	vt.cursor.row = 1
	vt.cursor.col = 4

	vt.decset(testCSI('h', []uint32{1048}, '?'))
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.decrst(testCSI('l', []uint32{1048}, '?'))

	if vt.cursor.row != 1 || vt.cursor.col != 4 {
		t.Fatalf("restored alt cursor = %d,%d, want 1,4", vt.cursor.row, vt.cursor.col)
	}
}

func TestAltScreenSwitchClearsActiveHyperlink(t *testing.T) {
	vt := New()
	vt.resize(5, 2)

	vt.osc("8;id=primary;https://example.com")
	printText(vt, "A")
	if got := vt.activeScreen.cell(0, 0).Hyperlink; got != "https://example.com" {
		t.Fatalf("primary printed hyperlink = %q, want active hyperlink", got)
	}

	vt.decset(testCSI('h', []uint32{47}, '?'))
	printText(vt, "B")
	if got := vt.activeScreen.cell(0, 1).Hyperlink; got != "" {
		t.Fatalf("alternate printed hyperlink = %q, want empty", got)
	}

	vt.decrst(testCSI('l', []uint32{47}, '?'))
	printText(vt, "C")
	if got := vt.activeScreen.cell(0, 2).Hyperlink; got != "" {
		t.Fatalf("primary printed hyperlink after return = %q, want empty", got)
	}
	if got := vt.activeScreen.cell(0, 0).Hyperlink; got != "https://example.com" {
		t.Fatalf("existing primary cell hyperlink = %q, want preserved hyperlink", got)
	}
}
