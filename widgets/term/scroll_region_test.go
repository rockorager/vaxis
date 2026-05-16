package term

import (
	"strings"
	"testing"
)

func TestScrollUpLeftRightRegionPreservesOutsideColumns(t *testing.T) {
	vt := New()
	vt.resize(10, 10)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 2
	vt.cursor.col = 2

	vt.scrollUp(1)

	if got, want := trimScreenString(vt.String()), "AEF423\nDHI756\nG   89"; got != want {
		t.Fatalf("screen mismatch after scroll up: got %q want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 2 {
		t.Fatalf("cursor moved during scroll up: got %d,%d want 2,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestScrollDownLeftRightRegionPreservesOutsideColumns(t *testing.T) {
	vt := New()
	vt.resize(10, 10)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 2
	vt.cursor.col = 2

	vt.scrollDown(1)

	if got, want := trimScreenString(vt.String()), "A   23\nDBC156\nGEF489\n HI7"; got != want {
		t.Fatalf("screen mismatch after scroll down: got %q want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 2 {
		t.Fatalf("cursor moved during scroll down: got %d,%d want 2,2", vt.cursor.row, vt.cursor.col)
	}
}

func TestInsertDeleteLinesRespectLeftRightRegion(t *testing.T) {
	vt := New()
	vt.resize(6, 4)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.il(1)

	if got, want := vt.String(), "ABC123\nD   56\nGEF489\n HI7  "; got != want {
		t.Fatalf("screen mismatch after IL: got %q want %q", got, want)
	}

	vt.dl(1)

	if got, want := vt.String(), "ABC123\nDEF456\nGHI789\n      "; got != want {
		t.Fatalf("screen mismatch after DL: got %q want %q", got, want)
	}
}

func TestPartialRowErasePreservesRowMetadata(t *testing.T) {
	screen := newScreenBuffer(4, 2, 0)
	screen.row(0).wrapped = true
	screen.row(0).wrapContinuation = true

	screen.eraseRow(0, 1, 2, 0)

	if !screen.row(0).wrapped {
		t.Fatal("partial row erase cleared wrapped metadata")
	}
	if !screen.row(0).wrapContinuation {
		t.Fatal("partial row erase cleared wrap continuation metadata")
	}
}

func trimScreenString(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
