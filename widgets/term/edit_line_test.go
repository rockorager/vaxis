package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestInsertLinesZeroParameterDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1
	vt.lastCol = true

	vt.update(testCSI('L', []uint32{0}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("IL zero parameter reset pending wrap")
	}
}

func TestDeleteLinesZeroParameterDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1
	vt.lastCol = true

	vt.update(testCSI('M', []uint32{0}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("DL zero parameter reset pending wrap")
	}
}

func TestInsertDeleteLinesDefaultParameterActsOnce(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1

	vt.update(testCSI('L', nil))
	if got, want := vt.String(), "AAA\n   \nBBB"; got != want {
		t.Fatalf("screen after default IL = %q want %q", got, want)
	}

	vt.update(testCSI('M', nil))
	if got, want := vt.String(), "AAA\nBBB\n   "; got != want {
		t.Fatalf("screen after default DL = %q want %q", got, want)
	}
}

func TestInsertDeleteLinesIgnoreMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1

	vt.update(testCSI('L', []uint32{1, 1}))
	vt.update(testCSI('M', []uint32{1, 1}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertDeleteLinesOutsideRegionDoNotResetPendingWrap(t *testing.T) {
	tests := []struct {
		name  string
		final rune
	}{
		{name: "insert", final: 'L'},
		{name: "delete", final: 'M'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 3)
			vt.margin.top = 1
			vt.margin.bottom = 2
			vt.cursor.row = 0
			vt.lastCol = true

			vt.update(testCSI(tt.final, nil))

			if !vt.lastCol {
				t.Fatal("operation outside region reset pending wrap")
			}
		})
	}
}

func TestDeleteLinesWideCharactersSplitByLeftRightRegionBoundaries(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	setScreenLine(vt.primaryScreen, 0, "AAAAA")
	setWideCell(vt.primaryScreen, 1, 0, "橋")
	vt.primaryScreen.setCell(1, 2, cellString("B"))
	setWideCell(vt.primaryScreen, 1, 3, "橋")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.dl(1)

	if got, want := vt.String(), "A B A\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.primaryScreen.cell(0, 1).Width != 0 {
		t.Fatalf("left split wide cell width = %d, want 0", vt.primaryScreen.cell(0, 1).Width)
	}
	if vt.primaryScreen.cell(0, 3).Width != 0 {
		t.Fatalf("right split wide cell width = %d, want 0", vt.primaryScreen.cell(0, 3).Width)
	}
}

func TestDeleteLinesFullRegionClearsWideCharacterAtRightMargin(t *testing.T) {
	vt := New()
	vt.resize(8, 4)

	for r := 0; r < 4; r += 1 {
		setWideCell(vt.primaryScreen, r, 5, "橋")
	}
	vt.margin.left = 2
	vt.margin.right = 5
	vt.cursor.row = 0
	vt.cursor.col = 2

	vt.dl(4)

	for r := row(0); r < 4; r += 1 {
		if got := vt.primaryScreen.cell(r, 5).Width; got != 0 {
			t.Fatalf("row %d right-margin cell width = %d, want blank", r, got)
		}
		tail := vt.primaryScreen.cell(r, 6)
		if tail.Width == 0 && tail.Grapheme == "" {
			continue
		}
		t.Fatalf("row %d orphaned tail beyond margin = %#v, want blank", r, tail.Character)
	}
}

func TestDeleteLinesClearsStaleWrapMetadataFromMovedRows(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "AAAAABBBB")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "CCC")

	if !vt.primaryScreen.row(1).wrapped {
		t.Fatal("setup row 1 was not wrapped")
	}

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.dl(1)

	if got, want := trimScreenString(vt.String()), "BBBB\n😀 CCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.primaryScreen.row(0).wrapped {
		t.Fatal("deleted lines kept stale wrapped metadata on moved row")
	}
	if vt.primaryScreen.row(1).wrapContinuation {
		t.Fatal("deleted lines kept stale wrap-continuation metadata on moved row")
	}
}

func TestEraseLineRightWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(10, 1)
	printText(vt, "AB")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "DE")
	vt.update(testCSI('H', []uint32{1, 4}))

	vt.el(0, false)

	if got, want := trimScreenString(vt.String()), "AB"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.primaryScreen.cell(0, 2)
	if head.Grapheme != "" || head.Width != 0 {
		t.Fatalf("wide head after EL right = %#v, want blank", head.Character)
	}
}

func TestEraseLineRightPreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 2}))
	bg := vaxis.IndexColor(3)
	vt.cursor.Background = bg

	vt.el(0, false)

	if got, want := vt.String(), "A    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(1); col < column(vt.width()); col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("erased cell background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestEraseLineLeftWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(10, 1)
	printText(vt, "AB")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "DE")
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.el(1, false)

	if got, want := trimScreenString(vt.String()), "    DE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	tail := vt.primaryScreen.cell(0, 3)
	if tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("wide tail after EL left = %#v, want blank", tail.Character)
	}
}

func TestEraseLineLeftPreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 2}))
	bg := vaxis.IndexColor(4)
	vt.cursor.Background = bg

	vt.el(1, false)

	if got, want := vt.String(), "  C  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col <= 1; col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("erased cell background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestEraseLineCompletePreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 2}))
	bg := vaxis.IndexColor(5)
	vt.cursor.Background = bg

	vt.el(2, false)

	if got, want := vt.String(), "     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("erased cell background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func setWideCell(screen screenBuffer, r int, c int, grapheme string) {
	screen.setCell(row(r), column(c), cell{
		Cell: vaxis.Cell{
			Character: vaxis.Character{
				Grapheme: grapheme,
				Width:    2,
			},
		},
	})
	screen.setCell(row(r), column(c+1), cell{
		Cell: vaxis.Cell{
			Character: vaxis.Character{
				Grapheme: " ",
			},
		},
	})
}
