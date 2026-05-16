package term

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDECCOLMIgnoredWithoutMode40(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.mode.deccolm = true

	vt.update(testCSI('h', []uint32{3}, '?'))

	if got, want := vt.width(), 5; got != want {
		t.Fatalf("width = %d, want %d", got, want)
	}
	if got, want := vt.height(), 5; got != want {
		t.Fatalf("height = %d, want %d", got, want)
	}
	if vt.mode.deccolm {
		t.Fatal("DECCOLM stayed set without mode 40")
	}
}

func TestDECCOLMSetAndResetResizeWidth(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.update(testCSI('h', []uint32{40}, '?'))

	vt.update(testCSI('h', []uint32{3}, '?'))

	if got, want := vt.width(), 132; got != want {
		t.Fatalf("width after DECCOLM set = %d, want %d", got, want)
	}
	if got, want := vt.height(), 5; got != want {
		t.Fatalf("height after DECCOLM set = %d, want %d", got, want)
	}
	if !vt.mode.deccolm {
		t.Fatal("DECCOLM mode was not set")
	}

	vt.update(testCSI('l', []uint32{3}, '?'))

	if got, want := vt.width(), 80; got != want {
		t.Fatalf("width after DECCOLM reset = %d, want %d", got, want)
	}
	if got, want := vt.height(), 5; got != want {
		t.Fatalf("height after DECCOLM reset = %d, want %d", got, want)
	}
	if vt.mode.deccolm {
		t.Fatal("DECCOLM mode stayed set after reset")
	}
}

func TestDECCOLMClearsDisplayAndCursor(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.update(testPrint("A"))
	vt.cursor.row = 3
	vt.cursor.col = 4
	vt.update(testCSI('h', []uint32{40}, '?'))

	vt.update(testCSI('l', []uint32{3}, '?'))

	if got, want := vt.String(), strings.Join([]string{
		strings.Repeat(" ", 80),
		strings.Repeat(" ", 80),
		strings.Repeat(" ", 80),
		strings.Repeat(" ", 80),
		strings.Repeat(" ", 80),
	}, "\n"); got != want {
		t.Fatalf("screen after DECCOLM = %q, want blank", got)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor after DECCOLM = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}

func TestDECCOLMResetsPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	for _, r := range "ABCDE" {
		vt.update(testPrint(string(r)))
	}
	if !vt.lastCol {
		t.Fatal("pending wrap was not set before DECCOLM")
	}
	vt.update(testCSI('h', []uint32{40}, '?'))

	vt.update(testCSI('l', []uint32{3}, '?'))

	if vt.lastCol {
		t.Fatal("pending wrap stayed set after DECCOLM")
	}
}

func TestDECCOLMPreservesCursorBackgroundForErase(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	bg := vaxis.RGBColor(0xff, 0, 0)
	vt.cursor.Background = bg
	vt.update(testCSI('h', []uint32{40}, '?'))

	vt.update(testCSI('l', []uint32{3}, '?'))

	if got := vt.activeScreen.cell(0, 0).Background; got != bg {
		t.Fatalf("background after DECCOLM = %v, want %v", got, bg)
	}
}

func TestDECCOLMResetsScrollRegion(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.mode.declrmm = true
	vt.margin.top = 1
	vt.margin.bottom = 2
	vt.margin.left = 2
	vt.margin.right = 4
	vt.update(testCSI('h', []uint32{40}, '?'))

	vt.update(testCSI('l', []uint32{3}, '?'))

	if !vt.mode.declrmm {
		t.Fatal("DECCOLM cleared left/right margin mode")
	}
	if vt.margin.top != 0 || vt.margin.bottom != 4 || vt.margin.left != 0 || vt.margin.right != 79 {
		t.Fatalf("margins after DECCOLM = top:%d bottom:%d left:%d right:%d, want 0,4,0,79", vt.margin.top, vt.margin.bottom, vt.margin.left, vt.margin.right)
	}
}

func TestEnableMode40DoesNotResizeToHostSize(t *testing.T) {
	vt := New()
	vt.resize(132, 5)
	vt.size = vaxis.Resize{Cols: 90, Rows: 7}

	vt.update(testCSI('h', []uint32{40}, '?'))

	if got, want := vt.width(), 132; got != want {
		t.Fatalf("width after mode 40 set = %d, want %d", got, want)
	}
	if got, want := vt.height(), 5; got != want {
		t.Fatalf("height after mode 40 set = %d, want %d", got, want)
	}
	if !vt.mode.enableMode3 {
		t.Fatal("mode 40 was not set")
	}
}

func TestEnableMode40WithoutHostSizeDoesNotResize(t *testing.T) {
	vt := New()
	vt.resize(132, 5)

	vt.update(testCSI('h', []uint32{40}, '?'))

	if got, want := vt.width(), 132; got != want {
		t.Fatalf("width after mode 40 set = %d, want %d", got, want)
	}
	if got, want := vt.height(), 5; got != want {
		t.Fatalf("height after mode 40 set = %d, want %d", got, want)
	}
}

func TestRestoreEnableMode40GatesDECCOLM(t *testing.T) {
	vt := New()
	vt.resize(5, 5)

	vt.update(testCSI('h', []uint32{40}, '?'))
	vt.update(testCSI('s', []uint32{40}, '?'))
	vt.update(testCSI('l', []uint32{40}, '?'))
	vt.update(testCSI('r', []uint32{40}, '?'))

	if !vt.mode.enableMode3 {
		t.Fatal("mode 40 was not restored")
	}

	vt.update(testCSI('h', []uint32{3}, '?'))

	if got, want := vt.width(), 132; got != want {
		t.Fatalf("width after restored mode 40 DECCOLM = %d, want %d", got, want)
	}
	if !vt.mode.deccolm {
		t.Fatal("DECCOLM did not set after restoring mode 40")
	}
}
