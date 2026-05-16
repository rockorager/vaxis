package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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

	screen.scrollUp(0, 1, 0, 2, 1, 0)

	if got, want := screen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := screen.scrollbackString(0), "a  "; got != want {
		t.Fatalf("scrollback line = %q, want %q", got, want)
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
