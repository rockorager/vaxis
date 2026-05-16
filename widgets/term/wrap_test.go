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

func TestCursorMovementPreservesSoftWrapMetadata(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Model)
		move  func(*Model)
	}{
		{
			name: "CUU",
			setup: func(vt *Model) {
				vt.cursor.row = 1
				vt.cursor.col = 1
			},
			move: func(vt *Model) {
				vt.cuu(1)
			},
		},
		{
			name: "CUD",
			setup: func(vt *Model) {
				vt.cursor.row = 0
				vt.cursor.col = 1
			},
			move: func(vt *Model) {
				vt.cud(1)
			},
		},
		{
			name: "CUF",
			setup: func(vt *Model) {
				vt.cursor.row = 0
				vt.cursor.col = 0
			},
			move: func(vt *Model) {
				vt.cuf(1)
			},
		},
		{
			name: "CUB",
			setup: func(vt *Model) {
				vt.cursor.row = 0
				vt.cursor.col = 1
			},
			move: func(vt *Model) {
				vt.cub(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 3)
			printText(vt, "abcdeX")
			tt.setup(vt)
			vt.lastCol = true

			tt.move(vt)

			if vt.lastCol {
				t.Fatal("cursor movement did not clear pending wrap")
			}
			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("cursor movement cleared source row wrap metadata")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("cursor movement cleared destination row continuation metadata")
			}
		})
	}
}

func TestHorizontalTabMovementPreservesPendingWrapAndSoftWrapMetadata(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Model)
		move  func(*Model)
	}{
		{
			name: "CHT",
			setup: func(vt *Model) {
				vt.cursor.row = 0
				vt.cursor.col = 0
			},
			move: func(vt *Model) {
				vt.cht(1)
			},
		},
		{
			name: "CBT",
			setup: func(vt *Model) {
				vt.cursor.row = 0
				vt.cursor.col = 4
			},
			move: func(vt *Model) {
				vt.cbt(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 3)
			printText(vt, "abcdeX")
			tt.setup(vt)
			vt.lastCol = true

			tt.move(vt)

			if !vt.lastCol {
				t.Fatal("horizontal tab movement cleared pending wrap")
			}
			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("horizontal tab movement cleared source row wrap metadata")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("horizontal tab movement cleared destination row continuation metadata")
			}
		})
	}
}

func TestEditOperationsResetSoftWrap(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Model)
	}{
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
				vt.el(0, false)
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

func TestInsertBlanksPreservesSoftWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "abcdeX")
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = true

	vt.ich(1)

	if vt.lastCol {
		t.Fatal("insert blanks did not clear pending wrap")
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("insert blanks cleared source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("insert blanks cleared destination row continuation metadata")
	}
}

func TestOriginModeTogglePreservesSoftWrapMetadata(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "set", enabled: true},
		{name: "reset", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 3)
			vt.margin.top = 1
			vt.margin.bottom = 2
			vt.margin.left = 1
			vt.margin.right = 4
			printText(vt, "abcdeX")
			vt.cursor.row = 0
			vt.cursor.col = 4
			vt.lastCol = true
			vt.mode.decom = !tt.enabled

			vt.setDECMode(6, tt.enabled)

			if vt.lastCol {
				t.Fatal("origin mode toggle did not clear pending wrap")
			}
			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("origin mode toggle cleared source row wrap metadata")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("origin mode toggle cleared destination row continuation metadata")
			}
			wantRow := row(0)
			wantCol := column(0)
			if tt.enabled {
				wantRow = vt.margin.top
				wantCol = vt.margin.left
			}
			if vt.cursor.row != wantRow || vt.cursor.col != wantCol {
				t.Fatalf("cursor = %d,%d, want %d,%d", vt.cursor.row, vt.cursor.col, wantRow, wantCol)
			}
		})
	}
}

func TestMarginSetPreservesSoftWrapMetadata(t *testing.T) {
	tests := []struct {
		name      string
		setupMode func(*Model)
		setMargin func(*Model)
	}{
		{
			name: "DECSTBM",
			setMargin: func(vt *Model) {
				vt.decstbm(testCSI('r', []uint32{2, 4}))
			},
		},
		{
			name: "DECSLRM",
			setupMode: func(vt *Model) {
				vt.mode.declrmm = true
			},
			setMargin: func(vt *Model) {
				vt.decslrm(testCSI('s', []uint32{2, 4}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 5)
			if tt.setupMode != nil {
				tt.setupMode(vt)
			}
			printText(vt, "abcdeX")
			vt.cursor.row = 0
			vt.cursor.col = 4
			vt.lastCol = true

			tt.setMargin(vt)

			if vt.lastCol {
				t.Fatal("margin set did not clear pending wrap")
			}
			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("margin set cleared source row wrap metadata")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("margin set cleared destination row continuation metadata")
			}
			if vt.cursor.row != 0 || vt.cursor.col != 0 {
				t.Fatalf("cursor = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
			}
		})
	}
}

func TestWraparoundModeTogglePreservesPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(3, 2)

	printText(vt, "ABC")
	vt.setDECMode(7, false)

	if !vt.lastCol {
		t.Fatal("wraparound reset cleared pending wrap")
	}
	if vt.cursor.row != 0 || vt.cursor.col != 2 {
		t.Fatalf("cursor after reset = %d,%d, want 0,2", vt.cursor.row, vt.cursor.col)
	}

	printText(vt, "X")

	if got, want := vt.String(), "ABX\n   "; got != want {
		t.Fatalf("screen after disabled wrap print = %q, want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("disabled wrap print cleared pending wrap")
	}

	vt.setDECMode(7, true)
	printText(vt, "Y")

	if got, want := vt.String(), "ABX\nY  "; got != want {
		t.Fatalf("screen after re-enabled wrap print = %q, want %q", got, want)
	}
}

func TestDisabledWraparoundDropsWideCharacterWithOneColumnRemaining(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.decawm = false
	printText(vt, "AAAA")

	vt.update(ansi.Print{Grapheme: "🚨", Width: 2})

	if got, want := vt.String(), "AAAA "; got != want {
		t.Fatalf("screen after disabled wrap wide print = %q, want %q", got, want)
	}
	if got := vt.activeScreen.cell(0, 4); got.Grapheme != "" || got.Width != 0 {
		t.Fatalf("last cell = %#v, want blank", got.Character)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 0,4", vt.cursor.row, vt.cursor.col)
	}
}

func TestDisabledWraparoundDropsWideCharacterWithNoColumnsRemaining(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.decawm = false
	printText(vt, "AAAAA")

	vt.update(ansi.Print{Grapheme: "🚨", Width: 2})

	if got, want := vt.String(), "AAAAA"; got != want {
		t.Fatalf("screen after disabled wrap wide print = %q, want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 4).Grapheme, "A"; got != want {
		t.Fatalf("last cell grapheme = %q, want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 0,4", vt.cursor.row, vt.cursor.col)
	}
}

func TestDisabledWraparoundKeepsNarrowGraphemeWhenVariationSelectorWouldWidenPastEdge(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.mode.decawm = false
	vt.mode.graphemeCluster = true
	printText(vt, "AAAA")
	vt.update(ansi.Print{Grapheme: "❤", Width: 1})

	vt.update(ansi.Print{Grapheme: "\uFE0F", Width: 0})

	if got, want := vt.String(), "AAAA❤"; got != want {
		t.Fatalf("screen after disabled wrap VS16 = %q, want %q", got, want)
	}
	cell := vt.activeScreen.cell(0, 4)
	if got, want := cell.Grapheme, "❤"; got != want {
		t.Fatalf("last cell grapheme = %q, want %q", got, want)
	}
	if got, want := cell.Width, 1; got != want {
		t.Fatalf("last cell width = %d, want %d", got, want)
	}
}

func TestWraparoundModeTogglePreservesSoftWrapMetadata(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "set", enabled: true},
		{name: "reset", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 3)
			printText(vt, "abcdeX")
			vt.cursor.row = 0
			vt.cursor.col = 5
			vt.lastCol = true
			vt.mode.decawm = !tt.enabled

			vt.setDECMode(7, tt.enabled)

			if !vt.lastCol {
				t.Fatal("wraparound mode toggle cleared pending wrap")
			}
			if !vt.activeScreen.row(0).wrapped {
				t.Fatal("wraparound mode toggle cleared source row wrap metadata")
			}
			if !vt.activeScreen.row(1).wrapContinuation {
				t.Fatal("wraparound mode toggle cleared destination row continuation metadata")
			}
			wantCol := column(5)
			if !tt.enabled {
				wantCol = 4
			}
			if vt.cursor.row != 0 || vt.cursor.col != wantCol {
				t.Fatalf("cursor = %d,%d, want 0,%d", vt.cursor.row, vt.cursor.col, wantCol)
			}
		})
	}
}

func TestIndexPreservesSoftWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "abcdeX")
	vt.cursor.row = 0
	vt.cursor.col = 4
	vt.lastCol = true

	vt.ind()

	if vt.lastCol {
		t.Fatal("index did not clear pending wrap")
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("index cleared source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("index cleared destination row continuation metadata")
	}
	if vt.cursor.row != 1 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 1,4", vt.cursor.row, vt.cursor.col)
	}
}

func TestReverseIndexAtTopMarginPreservesPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "abcde")
	vt.cursor.row = 0
	vt.cursor.col = 5
	vt.lastCol = true

	vt.ri()

	if !vt.lastCol {
		t.Fatal("reverse index at top margin cleared pending wrap")
	}
	if vt.cursor.row != 0 || vt.cursor.col != 5 {
		t.Fatalf("cursor = %d,%d, want 0,5", vt.cursor.row, vt.cursor.col)
	}
	printText(vt, "X")
	if got, want := vt.String(), "     \nXbcde\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.row(0).wrapped {
		t.Fatal("reverse index did not preserve source row wrap metadata")
	}
	if !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("reverse index did not preserve pending wrap continuation")
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

	vt.el(2, false)

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

	vt.ed(0, false)

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

func TestDisabledWraparoundWideCharacterWithOneCellRemainingIgnored(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "AAAA")
	vt.update(testCSI('l', []uint32{7}, '?'))

	vt.update(ansi.Print{Grapheme: "🚨", Width: 2})

	if got, want := vt.String(), "AAAA \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 0,4", vt.cursor.row, vt.cursor.col)
	}
	cell := vt.activeScreen.cell(0, 4)
	if cell.Grapheme != "" || cell.Width != 0 {
		t.Fatalf("last cell = %#v, want blank", cell.Character)
	}
	if vt.lastCol {
		t.Fatal("ignored wide character set pending wrap")
	}
}

func TestDisabledWraparoundWideCharacterAtLastColumnIgnored(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "AAAAA")
	vt.update(testCSI('l', []uint32{7}, '?'))

	vt.update(ansi.Print{Grapheme: "🚨", Width: 2})

	if got, want := vt.String(), "AAAAA\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 0,4", vt.cursor.row, vt.cursor.col)
	}
	cell := vt.activeScreen.cell(0, 4)
	if cell.Grapheme != "A" || cell.Width != 1 {
		t.Fatalf("last cell = %#v, want A", cell.Character)
	}
	if !vt.lastCol {
		t.Fatal("ignored wide character cleared pending wrap")
	}
}

func TestWideCharacterWrapAtRightMarginDoesNotMarkSoftWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	printText(vt, "123456789")
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{3, 5}))
	vt.update(testCSI('H', []uint32{1, 5}))

	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	if got, want := vt.String(), "123456789 \n  😀       \n          "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 1 || vt.cursor.col != 4 {
		t.Fatalf("cursor = %d,%d, want 1,4", vt.cursor.row, vt.cursor.col)
	}
	if vt.activeScreen.row(0).wrapped {
		t.Fatal("right-margin wrap marked source row as soft-wrapped")
	}
	if vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("right-margin wrap marked destination row as wrap continuation")
	}
}

func TestPrintRightMarginWrap(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	printText(vt, "123456789")
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{3, 5}))
	vt.update(testCSI('H', []uint32{1, 5}))

	printText(vt, "XY")

	if got, want := vt.String(), "1234X6789 \n  Y       \n          "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.activeScreen.row(0).wrapped {
		t.Fatal("right-margin wrap marked source row as soft-wrapped")
	}
	if vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("right-margin wrap marked destination row as wrap continuation")
	}
}

func TestPrintRightMarginOutsideDoesNotWrapAtMargin(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	printText(vt, "123456789")
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{3, 5}))
	vt.update(testCSI('H', []uint32{1, 6}))

	printText(vt, "XY")

	if got, want := vt.String(), "12345XY89 \n          \n          "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestPrintRightMarginOutsideWrapsAtScreenEdge(t *testing.T) {
	vt := New()
	vt.resize(10, 3)
	printText(vt, "123456789")
	vt.update(testCSI('h', []uint32{69}, '?'))
	vt.update(testCSI('s', []uint32{3, 5}))
	vt.update(testCSI('H', []uint32{1, 10}))

	printText(vt, "XY")

	if got, want := vt.String(), "123456789X\n  Y       \n          "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func printText(vt *Model, text string) {
	for _, r := range text {
		vt.update(ansi.Print{Grapheme: string(r), Width: 1})
	}
}
