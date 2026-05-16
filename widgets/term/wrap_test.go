package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestSoftWrapSetsRowContinuation(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	vt.update(ansi.Print{Grapheme: "a", Width: 1})
	vt.update(ansi.Print{Grapheme: "b", Width: 1})
	vt.update(ansi.Print{Grapheme: "c", Width: 1})
	vt.update(ansi.Print{Grapheme: "d", Width: 1})

	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("source row was not marked wrapped")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("destination row was not marked as wrap continuation")
	}
	if got, want := vt.String(), "abc\nd  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestSoftWrapAcrossScrollPreservesContinuation(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	setScreenLine(vt.primaryScreen, 0, "abc")
	vt.cursor.row = 0
	vt.cursor.col = 3
	vt.lastCol = true

	vt.update(ansi.Print{Grapheme: "d", Width: 1})
	vt.update(ansi.Print{Grapheme: "e", Width: 1})
	vt.update(ansi.Print{Grapheme: "f", Width: 1})
	vt.update(ansi.Print{Grapheme: "g", Width: 1})

	if !vt.primaryScreen.row(0).wrapped {
		t.Fatal("scrolled wrapped row did not preserve wrapped metadata")
	}
	if !vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("new bottom row was not marked as wrap continuation")
	}
}

func TestCursorMovementResetsPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abc")

	if !vt.lastCol {
		t.Fatal("terminal did not enter pending wrap state")
	}

	vt.cuu(1)

	if vt.lastCol {
		t.Fatal("cursor movement did not reset pending wrap")
	}
	vt.update(ansi.Print{Grapheme: "X", Width: 1})
	if got, want := vt.String(), "abX\n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestEditOperationsResetSoftWrap(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Model)
	}{
		{
			name: "insert characters",
			edit: func(vt *Model) {
				vt.ich(1)
			},
		},
		{
			name: "delete characters",
			edit: func(vt *Model) {
				vt.dch(1)
			},
		},
		{
			name: "erase characters",
			edit: func(vt *Model) {
				vt.ech(1)
			},
		},
		{
			name: "erase line",
			edit: func(vt *Model) {
				vt.el(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 2)
			printText(vt, "abcd")
			vt.cursor.row = 0
			vt.cursor.col = 0

			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("source row was not marked wrapped before edit")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("destination row was not marked as wrap continuation before edit")
			}

			tt.edit(vt)

			if vt.activeScreen.row(0).wrapped {
				t.Fatal("edit did not clear source row wrap")
			}
			if vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("edit did not clear destination row wrap continuation")
			}
		})
	}
}

func TestInsertLinesMovesRowMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 4)
	setScreenLine(vt.primaryScreen, 0, "aaa")
	setScreenLine(vt.primaryScreen, 1, "bbb")
	setScreenLine(vt.primaryScreen, 2, "ccc")
	setScreenLine(vt.primaryScreen, 3, "ddd")
	vt.activeScreen.row(1).wrapped = true
	vt.activeScreen.row(2).wrapContinuation = true
	vt.cursor.row = 0
	vt.cursor.col = 0

	vt.il(1)

	if got, want := vt.String(), "   \naaa\nbbb\nccc"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.row(2).wrapped {
		t.Fatal("inserted lines did not move wrapped metadata with row")
	}
	if !vt.activeScreen.row(3).wrapContinuation {
		t.Fatal("inserted lines did not move wrap continuation metadata with row")
	}
	if vt.activeScreen.row(0).wrapped || vt.activeScreen.row(0).wrapContinuation {
		t.Fatal("inserted blank row kept stale wrap metadata")
	}
}

func TestDeleteLinesMovesRowMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 4)
	setScreenLine(vt.primaryScreen, 0, "aaa")
	setScreenLine(vt.primaryScreen, 1, "bbb")
	setScreenLine(vt.primaryScreen, 2, "ccc")
	setScreenLine(vt.primaryScreen, 3, "ddd")
	vt.activeScreen.row(1).wrapped = true
	vt.activeScreen.row(2).wrapContinuation = true
	vt.cursor.row = 0
	vt.cursor.col = 0

	vt.dl(1)

	if got, want := vt.String(), "bbb\nccc\nddd\n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("deleted lines did not move wrapped metadata with row")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("deleted lines did not move wrap continuation metadata with row")
	}
	if vt.activeScreen.row(3).wrapped || vt.activeScreen.row(3).wrapContinuation {
		t.Fatal("deleted blank row kept stale wrap metadata")
	}
}

func TestEraseLineCompletePreservesSoftWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abcd")
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.el(2)

	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("complete line erase cleared source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("complete line erase cleared destination row continuation metadata")
	}
	if got, want := vt.String(), "   \nd  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestEraseDisplayClearsFullyErasedRowMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	printText(vt, "abcdefg")
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.ed(0)

	if vt.activeScreen.row(1).wrapped {
		t.Fatal("erase display below kept wrapped metadata on fully erased row")
	}
	if vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("erase display below kept wrap continuation metadata on fully erased row")
	}
	if vt.activeScreen.row(2).wrapContinuation {
		t.Fatal("erase display below kept wrap continuation metadata on trailing erased row")
	}
}

func TestSaveCursorPreservesPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.cursor.col = 4
	vt.update(ansi.Print{Grapheme: "A", Width: 1})
	vt.decsc()

	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = false
	vt.update(ansi.Print{Grapheme: "B", Width: 1})
	vt.decrc()
	vt.update(ansi.Print{Grapheme: "X", Width: 1})

	if got, want := vt.String(), "B   A\nX    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("restored pending wrap did not mark source row wrapped")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("restored pending wrap did not mark destination row continuation")
	}
}

func printText(vt *Model, text string) {
	for _, r := range text {
		vt.update(ansi.Print{Grapheme: string(r), Width: 1})
	}
}
