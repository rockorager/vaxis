package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestScreenBufferHandleSharesState(t *testing.T) {
	primary := newScreenBuffer(4, 2, defaultScrollbackLines)
	active := primary

	active.setCell(0, 0, cellString("x"))

	if got, want := primary.String(), "x   \n    "; got != want {
		t.Fatalf("primary screen mismatch: got %q want %q", got, want)
	}
}

func TestScreenBufferCapturesFullScreenScrollback(t *testing.T) {
	screen := newScreenBuffer(3, 2, defaultScrollbackLines)
	screen.setCell(0, 0, cellString("a"))
	screen.setCell(1, 0, cellString("b"))
	screen.row(0).wrapped = true

	screen.scrollUp(0, 1, 0, 2, 1, 0)

	if got, want := screen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := screen.scrollbackString(0), "a  "; got != want {
		t.Fatalf("scrollback line = %q, want %q", got, want)
	}
	if !screen.scrollbackRow(0).wrapped {
		t.Fatal("scrollback line did not preserve wrapped metadata")
	}
	if got, want := screen.String(), "b  \n   "; got != want {
		t.Fatalf("screen mismatch after scroll: got %q want %q", got, want)
	}
}

func TestScreenBufferDoesNotCapturePartialRegionScrollback(t *testing.T) {
	screen := newScreenBuffer(3, 3, defaultScrollbackLines)
	screen.setCell(1, 0, cellString("b"))

	screen.scrollUp(1, 2, 0, 2, 1, 0)

	if got := screen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
}

func TestScreenBufferScrollbackTrimsByLineLimit(t *testing.T) {
	screen := newScreenBuffer(2, 1, 2)

	for _, text := range []string{"a", "b", "c"} {
		screen.setCell(0, 0, cellString(text))
		screen.scrollUp(0, 0, 0, 1, 1, 0)
	}

	if got, want := screen.scrollbackLen(), 2; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := screen.scrollbackString(0), "b "; got != want {
		t.Fatalf("oldest scrollback line = %q, want %q", got, want)
	}
	if got, want := screen.scrollbackString(1), "c "; got != want {
		t.Fatalf("newest scrollback line = %q, want %q", got, want)
	}
}

func TestScreenBufferScrollPreservesRowMetadata(t *testing.T) {
	screen := newScreenBuffer(3, 3, 0)
	screen.row(1).wrapped = true
	screen.row(1).wrapContinuation = true

	screen.scrollUp(0, 2, 0, 2, 1, 0)

	if !screen.row(0).wrapped {
		t.Fatal("wrapped metadata did not move with scrolled row")
	}
	if !screen.row(0).wrapContinuation {
		t.Fatal("wrap continuation metadata did not move with scrolled row")
	}
	if screen.row(2).wrapped {
		t.Fatal("blank inserted row kept stale wrapped metadata")
	}
	if screen.row(2).wrapContinuation {
		t.Fatal("blank inserted row kept stale wrap continuation metadata")
	}
}

func cellString(s string) cell {
	return cell{
		Cell: vaxis.Cell{
			Character: vaxis.Character{
				Grapheme: s,
				Width:    1,
			},
		},
	}
}
