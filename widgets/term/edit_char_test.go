package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ansi"
)

func TestInsertBlanks(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(2)

	if got, want := trimScreenString(vt.String()), "  ABC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksMissingParameterDefaultsToOne(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.update(testCSI('@', nil))

	if got, want := vt.String(), " ABC "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksZeroParameterInsertsOne(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.update(testCSI('@', []uint32{0}))

	if got, want := vt.String(), " ABC "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksSemanticZeroClearsPendingWrapOnly(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABCDE")

	if !vt.lastCol {
		t.Fatal("setup did not enter pending wrap")
	}

	vt.ich(0)

	if got, want := vt.String(), "ABCDE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.lastCol {
		t.Fatal("semantic zero insert blanks did not clear pending wrap")
	}
}

func TestInsertBlanksIgnoresMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.update(testCSI('@', []uint32{1, 1}))

	if got, want := vt.String(), "ABC  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksIgnoresPrivateForm(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.update(testCSI('@', []uint32{2}, '?'))

	if got, want := vt.String(), "ABC  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksPushesOffEnd(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(2)

	if got, want := trimScreenString(vt.String()), "  A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksPreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))
	bg := vaxis.IndexColor(2)
	vt.cursor.Background = bg

	vt.ich(2)

	if got, want := vt.String(), "  ABC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < 2; col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("inserted blank background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestEraseCellResetsForegroundAndKeepsBackground(t *testing.T) {
	vt := New()
	vt.resize(3, 1)
	fg := vaxis.IndexColor(1)
	bg := vaxis.IndexColor(2)
	vt.cursor.Style = vaxis.Style{
		Foreground:     fg,
		Background:     vaxis.IndexColor(3),
		UnderlineColor: vaxis.IndexColor(4),
		UnderlineStyle: vaxis.UnderlineSingle,
	}
	printText(vt, "A")
	vt.cursor.col = 0
	vt.cursor.Style = vaxis.Style{Background: bg}

	vt.ech(1)

	cell := vt.activeScreen.cell(0, 0)
	if got := cell.Foreground; got != 0 {
		t.Fatalf("erased cell foreground = %v, want default", got)
	}
	if got := cell.UnderlineColor; got != 0 {
		t.Fatalf("erased cell underline color = %v, want default", got)
	}
	if got := cell.UnderlineStyle; got != vaxis.UnderlineOff {
		t.Fatalf("erased cell underline style = %v, want off", got)
	}
	if got := cell.Background; got != bg {
		t.Fatalf("erased cell background = %v, want %v", got, bg)
	}
	if got := cell.Grapheme; got != "" {
		t.Fatalf("erased cell grapheme = %q, want empty", got)
	}
}

func TestBlankProducingOperationsResetStyleAndKeepBackground(t *testing.T) {
	type pos struct {
		row row
		col column
	}
	fg := vaxis.IndexColor(1)
	bg := vaxis.IndexColor(2)
	styled := vaxis.Style{
		Foreground:     fg,
		Background:     vaxis.IndexColor(3),
		UnderlineColor: vaxis.IndexColor(4),
		UnderlineStyle: vaxis.UnderlineSingle,
		Attribute:      vaxis.AttrBold,
	}

	tests := []struct {
		name   string
		width  int
		height int
		row    row
		col    column
		run    func(*Model)
		checks []pos
	}{
		{
			name:   "ICH inserted blanks",
			width:  6,
			height: 1,
			col:    1,
			run:    func(vt *Model) { vt.ich(2) },
			checks: []pos{{0, 1}, {0, 2}},
		},
		{
			name:   "DCH blank fill",
			width:  6,
			height: 1,
			col:    2,
			run:    func(vt *Model) { vt.dch(2) },
			checks: []pos{{0, 4}, {0, 5}},
		},
		{
			name:   "ECH erased chars",
			width:  6,
			height: 1,
			col:    1,
			run:    func(vt *Model) { vt.ech(2) },
			checks: []pos{{0, 1}, {0, 2}},
		},
		{
			name:   "EL right",
			width:  5,
			height: 1,
			col:    2,
			run:    func(vt *Model) { vt.el(0, false) },
			checks: []pos{{0, 2}, {0, 3}, {0, 4}},
		},
		{
			name:   "EL left",
			width:  5,
			height: 1,
			col:    2,
			run:    func(vt *Model) { vt.el(1, false) },
			checks: []pos{{0, 0}, {0, 1}, {0, 2}},
		},
		{
			name:   "EL complete",
			width:  5,
			height: 1,
			col:    2,
			run:    func(vt *Model) { vt.el(2, false) },
			checks: []pos{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}},
		},
		{
			name:   "ED below",
			width:  3,
			height: 3,
			row:    1,
			col:    1,
			run:    func(vt *Model) { vt.ed(0, false) },
			checks: []pos{{1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}},
		},
		{
			name:   "ED above",
			width:  3,
			height: 3,
			row:    1,
			col:    1,
			run:    func(vt *Model) { vt.ed(1, false) },
			checks: []pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}},
		},
		{
			name:   "ED complete",
			width:  3,
			height: 2,
			row:    1,
			col:    1,
			run:    func(vt *Model) { vt.ed(2, false) },
			checks: []pos{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(tt.width, tt.height)
			for r := row(0); r < row(tt.height); r += 1 {
				for c := column(0); c < column(tt.width); c += 1 {
					vt.activeScreen.setCell(r, c, cell{Cell: vaxis.Cell{
						Character: vaxis.Character{Grapheme: "X", Width: 1},
						Style:     styled,
					}})
				}
			}
			vt.cursor.row = tt.row
			vt.cursor.col = tt.col
			vt.cursor.Style = vaxis.Style{Background: bg}

			tt.run(vt)

			for _, check := range tt.checks {
				cell := vt.activeScreen.cell(check.row, check.col)
				if got := cell.Foreground; got != 0 {
					t.Fatalf("cell %d,%d foreground = %v, want default", check.col, check.row, got)
				}
				if got := cell.Attribute; got != 0 {
					t.Fatalf("cell %d,%d attributes = %v, want default", check.col, check.row, got)
				}
				if got := cell.UnderlineColor; got != 0 {
					t.Fatalf("cell %d,%d underline color = %v, want default", check.col, check.row, got)
				}
				if got := cell.UnderlineStyle; got != vaxis.UnderlineOff {
					t.Fatalf("cell %d,%d underline style = %v, want off", check.col, check.row, got)
				}
				if got := cell.Background; got != bg {
					t.Fatalf("cell %d,%d background = %v, want %v", check.col, check.row, got, bg)
				}
				if got := cell.Grapheme; got != "" {
					t.Fatalf("cell %d,%d grapheme = %q, want empty", check.col, check.row, got)
				}
			}
		})
	}
}

func TestInsertBlanksMoreThanLineWidth(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(5)

	if got, want := trimScreenString(vt.String()), ""; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksAtLastColumnBlanksCell(t *testing.T) {
	vt := New()
	vt.resize(3, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.ich(1)

	if got, want := vt.String(), "AB "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksOutsideLeftRightRegionDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 5
	vt.lastCol = true

	vt.ich(2)

	if got, want := vt.String(), "ABC123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.lastCol {
		t.Fatal("insert blanks outside region did not reset pending wrap")
	}
}

func TestInsertBlanksInsideLeftRightRegion(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 2

	vt.ich(2)

	if got, want := vt.String(), "AB  C3"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksShiftHyperlinks(t *testing.T) {
	vt := New()
	vt.OSC8 = true
	vt.resize(10, 1)
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(2)

	if got, want := vt.String(), "  ABC     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < 2; col++ {
		if got := vt.activeScreen.cell(0, col).Hyperlink; got != "" {
			t.Fatalf("inserted blank hyperlink at col %d = %q, want empty", col, got)
		}
	}
	for col := column(2); col < 5; col++ {
		if got, want := vt.activeScreen.cell(0, col).Hyperlink, "https://example.com"; got != want {
			t.Fatalf("shifted hyperlink at col %d = %q, want %q", col, got, want)
		}
	}
}

func TestInsertBlanksPushesHyperlinkOffEnd(t *testing.T) {
	vt := New()
	vt.OSC8 = true
	vt.resize(3, 1)
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(3)

	if got, want := vt.String(), "   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < 3; col++ {
		if got := vt.activeScreen.cell(0, col).Hyperlink; got != "" {
			t.Fatalf("cell hyperlink at col %d = %q, want empty", col, got)
		}
	}
}

func TestInsertBlanksDeletesParsedWideGrapheme(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027hABC👨\u200d👩\u200d👧")
	vt.update(testCSI('H', []uint32{1, 1}))
	vt.ich(4)

	if got, want := vt.String(), "    A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < column(vt.width()); col++ {
		cell := vt.activeScreen.cell(0, col)
		if cell.Grapheme == "👨\u200d👩\u200d👧" || cell.Width > 1 {
			t.Fatalf("grapheme survived after ICH delete at col %d: %#v", col, cell.Character)
		}
	}
}

func TestInsertBlanksShiftsParsedWideGrapheme(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	parseAndApply(t, vt, "\x1b[?2027hA👨\u200d👩\u200d👧")
	vt.update(testCSI('H', []uint32{1, 1}))
	vt.ich(1)

	if got, want := vt.String(), " A👨\u200d👩\u200d👧  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.activeScreen.cell(0, 2)
	if got, want := head.Grapheme, "👨\u200d👩\u200d👧"; got != want {
		t.Fatalf("shifted grapheme = %q, want %q", got, want)
	}
	if got, want := head.Width, 2; got != want {
		t.Fatalf("shifted grapheme width = %d, want %d", got, want)
	}
	tail := vt.activeScreen.cell(0, 3)
	if got, want := tail.Grapheme, " "; got != want {
		t.Fatalf("shifted grapheme tail = %q, want %q", got, want)
	}
	if got, want := tail.Width, 0; got != want {
		t.Fatalf("shifted grapheme tail width = %d, want %d", got, want)
	}
}

func TestDeleteCharsShiftHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(8, 1)
	printText(vt, "12")
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.osc("8;;")
	printText(vt, "xyz")
	vt.cursor.col = 0

	vt.dch(2)

	if got, want := vt.String(), "ABCxyz  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < 3; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got, want := cell.Hyperlink, "https://example.com"; got != want {
			t.Fatalf("shifted hyperlink at col %d = %q, want %q", col, got, want)
		}
		if got, want := cell.HyperlinkParams, "id=link"; got != want {
			t.Fatalf("shifted hyperlink params at col %d = %q, want %q", col, got, want)
		}
	}
	for col := column(3); col < 8; col += 1 {
		if got := vt.activeScreen.cell(0, col).Hyperlink; got != "" {
			t.Fatalf("cell hyperlink at col %d = %q, want empty", col, got)
		}
	}
}

func TestDeleteCharsPreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.update(testCSI('H', []uint32{1, 3}))
	bg := vaxis.IndexColor(4)
	vt.cursor.Background = bg

	vt.dch(2)

	if got, want := vt.String(), "AB23  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(4); col < column(vt.width()); col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("blank fill background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestEraseCharsClearsHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.osc("8;;")
	printText(vt, "DEF")
	vt.cursor.col = 1

	vt.ech(3)

	if got, want := vt.String(), "A   EF"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(1); col < 4; col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if got := cell.Hyperlink; got != "" {
			t.Fatalf("erased cell %d hyperlink = %q, want empty", col, got)
		}
		if got := cell.HyperlinkParams; got != "" {
			t.Fatalf("erased cell %d hyperlink params = %q, want empty", col, got)
		}
	}
	if got, want := vt.activeScreen.cell(0, 0).Hyperlink, "https://example.com"; got != want {
		t.Fatalf("preserved cell hyperlink = %q, want %q", got, want)
	}
}

func TestEraseCharsPreservesBackgroundSGR(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))
	bg := vaxis.IndexColor(6)
	vt.cursor.Background = bg

	vt.ech(2)

	if got, want := vt.String(), "  C  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(0); col < 2; col += 1 {
		if got := vt.activeScreen.cell(0, col).Background; got != bg {
			t.Fatalf("erased cell background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestInsertBlanksSplitWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "123")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(1)

	if got, want := vt.String(), " 123 "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksSplitWideCharacterFromTail(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 2}))

	vt.ich(1)

	if got, want := vt.String(), "   12"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksWideCharacterStraddlingRightMargin(t *testing.T) {
	vt := New()
	vt.resize(10, 1)
	printText(vt, "ABCD")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.margin.right = 4
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.ich(1)

	if got, want := trimScreenString(vt.String()), "AB CD"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	tail := vt.activeScreen.cell(0, 5)
	if tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("cell beyond margin = %#v, want blank", tail.Character)
	}
}

func TestInsertBlanksWideCharacterTailBeyondRightMarginCleared(t *testing.T) {
	vt := New()
	vt.resize(10, 1)
	for i := 0; i < 5; i += 1 {
		vt.update(ansi.Print{Grapheme: "中", Width: 2})
	}
	vt.margin.left = 0
	vt.margin.right = 8
	vt.cursor.col = 0
	vt.update(testPrint("a"))

	vt.ich(8)

	if got, want := trimScreenString(vt.String()), "a"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	tail := vt.activeScreen.cell(0, 9)
	if tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("cell beyond margin = %#v, want blank", tail.Character)
	}
}

func TestPrintOverWideCharacterClearsTail(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.update(testPrint("A"))

	if got, want := vt.String(), "A    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	tail := vt.activeScreen.cell(0, 1)
	if tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("tail cell = %#v, want blank", tail.Character)
	}
}

func TestPrintOverWideCharacterTailClearsHead(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.update(testCSI('H', []uint32{1, 2}))

	vt.update(testPrint("X"))

	if got, want := vt.String(), " X   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.activeScreen.cell(0, 0)
	if head.Grapheme != "" || head.Width != 0 {
		t.Fatalf("head cell = %#v, want blank", head.Character)
	}
}

func TestDeleteCharsOutsideLeftRightRegionPreservesPendingWrap(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 5
	vt.lastCol = true

	vt.dch(2)

	if got, want := vt.String(), "ABC123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("delete chars outside region reset pending wrap")
	}
}

func TestDeleteCharsZeroParameterDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABCDE")
	vt.update(testCSI('H', []uint32{1, 2}))
	vt.lastCol = true

	vt.update(testCSI('P', []uint32{0}))

	if got, want := vt.String(), "ABCDE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("DCH zero parameter reset pending wrap")
	}
}

func TestDeleteCharsDefaultParameterDeletesOne(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABCDE")
	vt.update(testCSI('H', []uint32{1, 2}))

	vt.update(testCSI('P', nil))

	if got, want := vt.String(), "ACDE "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsIgnoresMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABCDE")
	vt.update(testCSI('H', []uint32{1, 2}))

	vt.update(testCSI('P', []uint32{1, 1}))

	if got, want := vt.String(), "ABCDE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsInsideLeftRightRegion(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 3

	vt.dch(1)

	if got, want := vt.String(), "ABC2 3"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsWideCharacterAcrossRightMargin(t *testing.T) {
	vt := New()
	vt.resize(8, 1)
	printText(vt, "123456")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	vt.margin.left = 1
	vt.margin.right = 6
	vt.cursor.col = 1

	vt.dch(1)

	if got, want := trimScreenString(vt.String()), "13456"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	for col := column(5); col < column(vt.width()); col += 1 {
		cell := vt.activeScreen.cell(0, col)
		if cell.Width > 1 {
			t.Fatalf("wide cell survived at col %d: %#v", col, cell.Character)
		}
	}
}

func TestDeleteCharsSplitWideCharacterFromTail(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "A")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.dch(1)

	if got, want := vt.String(), "A 123 "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsSplitWideCharacterFromHead(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.dch(1)

	if got, want := vt.String(), " 123  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsPreservesWideCharacterShiftedFromEnd(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "A")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "123")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.dch(1)

	if got, want := trimScreenString(vt.String()), "橋 123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.activeScreen.cell(0, 0).Width, 2; got != want {
		t.Fatalf("shifted wide head width = %d, want %d", got, want)
	}
	if got := vt.activeScreen.cell(0, 1).Grapheme; got != " " {
		t.Fatalf("shifted wide tail grapheme = %q, want space", got)
	}
	if got := vt.activeScreen.cell(0, 2).Grapheme; got != "1" {
		t.Fatalf("cell after shifted wide character = %q, want 1", got)
	}
}

func TestDeleteCharsWideCharacterBoundaryConditions(t *testing.T) {
	vt := New()
	vt.resize(8, 1)
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "a")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "b")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	if got, want := trimScreenString(vt.String()), "😀 a😀 b😀"; got != want {
		t.Fatalf("screen mismatch before DCH: got %q want %q", got, want)
	}

	vt.update(testCSI('H', []uint32{1, 2}))
	vt.dch(3)

	if got, want := trimScreenString(vt.String()), "  b😀"; got != want {
		t.Fatalf("screen mismatch after DCH: got %q want %q", got, want)
	}
}

func TestDeleteCharsWideCharacterWrapBoundaryConditions(t *testing.T) {
	vt := New()
	vt.resize(8, 3)
	printText(vt, ".......")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "abcde")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "......")

	if got, want := trimScreenString(vt.String()), ".......\n😀 abcde\n😀 ......"; got != want {
		t.Fatalf("screen mismatch before DCH: got %q want %q", got, want)
	}

	vt.update(testCSI('H', []uint32{2, 2}))
	vt.dch(3)

	if got, want := trimScreenString(vt.String()), ".......\n cde\n😀 ......"; got != want {
		t.Fatalf("screen mismatch after DCH: got %q want %q", got, want)
	}
	if vt.activeScreen.row(1).wrapped {
		t.Fatal("DCH did not clear edited row wrap metadata")
	}
	if vt.activeScreen.row(2).wrapContinuation {
		t.Fatal("DCH did not clear following row wrap-continuation metadata")
	}
}

func TestEraseCharsWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "BC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ech(1)
	printText(vt, "X")

	if got, want := vt.String(), "X BC "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestEraseCharsWideCharacterFromTail(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "A")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "BC")
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.ech(1)
	printText(vt, "X")

	if got, want := vt.String(), "A XBC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestEraseCharsWideCharacterBoundaryConditions(t *testing.T) {
	vt := New()
	vt.resize(8, 1)
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "a")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "b")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})

	vt.update(testCSI('H', []uint32{1, 2}))
	vt.ech(3)

	if got, want := trimScreenString(vt.String()), "     b😀"; got != want {
		t.Fatalf("screen mismatch after ECH: got %q want %q", got, want)
	}
}

func TestEraseCharsWideCharacterWrapBoundaryConditions(t *testing.T) {
	vt := New()
	vt.resize(8, 3)
	printText(vt, ".......")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "abcde")
	vt.update(ansi.Print{Grapheme: "😀", Width: 2})
	printText(vt, "......")

	vt.update(testCSI('H', []uint32{2, 2}))
	vt.ech(3)

	if got, want := trimScreenString(vt.String()), ".......\n    cde\n😀 ......"; got != want {
		t.Fatalf("screen mismatch after ECH: got %q want %q", got, want)
	}
}

func TestEraseCharsIgnoresMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "ABCDE")
	vt.update(testCSI('H', []uint32{1, 2}))

	vt.update(testCSI('X', []uint32{1, 1}))

	if got, want := vt.String(), "ABCDE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
