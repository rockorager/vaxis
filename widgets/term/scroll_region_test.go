package term

import (
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
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

func TestScrollUpLeftRightRegionDoesNotCreateScrollback(t *testing.T) {
	vt := New()
	vt.resize(6, 3)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3

	vt.scrollUp(1)

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
	if got, want := vt.String(), "AEF423\nDHI756\nG   89"; got != want {
		t.Fatalf("screen mismatch after scroll up: got %q want %q", got, want)
	}
}

func TestScrollUpTopRegionCreatesScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	setScreenLine(vt.primaryScreen, 0, "1    ")
	setScreenLine(vt.primaryScreen, 1, "2    ")
	setScreenLine(vt.primaryScreen, 2, "3    ")
	setScreenLine(vt.primaryScreen, 3, "X    ")
	vt.margin.top = 0
	vt.margin.bottom = 2
	vt.margin.left = 0
	vt.margin.right = 4

	vt.scrollUp(1)
	setScreenLine(vt.primaryScreen, 2, "Y    ")

	if got, want := vt.primaryScreen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(0), "1    "; got != want {
		t.Fatalf("scrollback line = %q, want %q", got, want)
	}
	if got, want := vt.String(), "2    \n3    \nY    \nX    \n     "; got != want {
		t.Fatalf("screen mismatch after scroll up: got %q want %q", got, want)
	}
}

func TestIndexBottomOfPrimaryScreenPreservesBackgroundOnBlankLine(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	bg := vaxis.RGBColor(0xff, 0, 0)
	vt.cursor.row = 4
	vt.cursor.col = 0
	vt.update(testPrint("A"))
	vt.cursor.Background = bg

	vt.ind()

	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.primaryScreen.cell(4, col).Background; got != bg {
			t.Fatalf("blank row background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestIndexBottomOfScrollRegionPreservesBackgroundOnBlankLine(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	bg := vaxis.RGBColor(0xff, 0, 0)
	vt.margin.top = 0
	vt.margin.bottom = 2
	setScreenLine(vt.primaryScreen, 0, "1    ")
	setScreenLine(vt.primaryScreen, 1, "2    ")
	setScreenLine(vt.primaryScreen, 2, "3    ")
	setScreenLine(vt.primaryScreen, 3, "X    ")
	vt.cursor.row = 2
	vt.cursor.col = 0
	vt.cursor.Background = bg

	vt.ind()

	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.primaryScreen.cell(2, col).Background; got != bg {
			t.Fatalf("blank scroll-region row background at col %d = %v, want %v", col, got, bg)
		}
	}
}

func TestScrollUpMovesHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	printText(vt, "ABC")
	vt.cr()
	vt.lf()
	vt.osc("8;id=link;https://example.com")
	printText(vt, "DEF")
	vt.osc("8;;")
	vt.cr()
	vt.lf()
	printText(vt, "GHI")

	vt.scrollUp(1)

	for col := column(0); col < 3; col += 1 {
		if got, want := vt.primaryScreen.cell(0, col).Hyperlink, "https://example.com"; got != want {
			t.Fatalf("moved hyperlink at col %d = %q, want %q", col, got, want)
		}
	}
	for col := column(0); col < 3; col += 1 {
		if got := vt.primaryScreen.cell(1, col).Hyperlink; got != "" {
			t.Fatalf("unlinked row hyperlink at col %d = %q, want empty", col, got)
		}
	}
}

func TestScrollUpClearsHyperlinkOnBlankLine(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.osc("8;;")
	vt.cr()
	vt.lf()
	printText(vt, "DEF")
	vt.cr()
	vt.lf()
	printText(vt, "GHI")

	vt.scrollUp(3)

	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.primaryScreen.cell(0, col).Hyperlink; got != "" {
			t.Fatalf("blank row hyperlink at col %d = %q, want empty", col, got)
		}
	}
}

func TestIndexBottomOfScrollRegionClearsHyperlinkOnBlankLine(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 1
	vt.margin.bottom = 2
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.osc("8;id=link;https://example.com")
	printText(vt, "A")
	vt.osc("8;;")
	vt.ind()
	vt.cr()
	printText(vt, "B")
	vt.ind()
	vt.cr()
	printText(vt, "C")

	if got, want := vt.primaryScreen.cell(1, 0).Hyperlink, ""; got != want {
		t.Fatalf("top scroll-region hyperlink = %q, want empty", got)
	}
	if got, want := vt.primaryScreen.cell(2, 0).Hyperlink, ""; got != want {
		t.Fatalf("bottom scroll-region hyperlink = %q, want empty", got)
	}
}

func TestIndexBottomOfScrollRegionCreatesScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 0
	vt.margin.bottom = 2
	setScreenLine(vt.primaryScreen, 0, "1    ")
	setScreenLine(vt.primaryScreen, 1, "2    ")
	setScreenLine(vt.primaryScreen, 2, "3    ")
	setScreenLine(vt.primaryScreen, 3, "X    ")
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.ind()
	vt.update(testPrint("Y"))

	if got, want := vt.primaryScreen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(0), "1    "; got != want {
		t.Fatalf("scrollback line = %q, want %q", got, want)
	}
	if got, want := vt.String(), "2    \n3    \nY    \nX    \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestIndexBottomOfScrollRegionWithScrollbackDisabled(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.primaryScreen.state.scrollbackLimit = 0
	vt.margin.top = 0
	vt.margin.bottom = 2
	vt.cursor.row = 3
	vt.cursor.col = 0
	vt.update(testPrint("B"))
	vt.cursor.row = 2
	vt.cursor.col = 0
	vt.update(testPrint("A"))
	vt.cursor.row = 2
	vt.cursor.col = 1

	vt.ind()
	vt.update(testPrint("X"))

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
	if got, want := trimScreenString(vt.String()), "\nA\n X\nB"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestIndexBottomOfScrollRegionBlankLinePreservesBackground(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	bg := vaxis.RGBColor(0xff, 0, 0)
	vt.margin.top = 0
	vt.margin.bottom = 2
	setScreenLine(vt.primaryScreen, 0, "1    ")
	setScreenLine(vt.primaryScreen, 1, "2    ")
	setScreenLine(vt.primaryScreen, 2, "3    ")
	setScreenLine(vt.primaryScreen, 3, "X    ")
	vt.cursor.row = 2
	vt.cursor.col = 0
	vt.cursor.Background = bg

	vt.ind()

	if got, want := vt.primaryScreen.scrollbackLen(), 1; got != want {
		t.Fatalf("scrollback length = %d, want %d", got, want)
	}
	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.primaryScreen.cell(2, col).Background; got != bg {
			t.Fatalf("blank scroll-region row background at col %d = %v, want %v", col, got, bg)
		}
	}
	if got, want := vt.String(), "2    \n3    \n     \nX    \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestIndexOutsideLeftRightMarginDoesNotScrollRegion(t *testing.T) {
	vt := New()
	vt.resize(10, 5)
	setScreenLine(vt.primaryScreen, 0, "000AAA")
	setScreenLine(vt.primaryScreen, 1, "111BBB")
	setScreenLine(vt.primaryScreen, 2, "222CCC")
	vt.margin.top = 0
	vt.margin.bottom = 2
	vt.margin.left = 3
	vt.margin.right = 5
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.ind()
	vt.update(testPrint("X"))

	if got, want := trimScreenString(vt.String()), "000AAA\n111BBB\nX22CCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestIndexInsideLeftRightMarginScrollsRegion(t *testing.T) {
	vt := New()
	vt.resize(10, 5)
	setScreenLine(vt.primaryScreen, 0, "AAAAAA")
	setScreenLine(vt.primaryScreen, 1, "AAAAAA")
	setScreenLine(vt.primaryScreen, 2, "AAAAAA")
	vt.margin.top = 0
	vt.margin.bottom = 2
	vt.margin.left = 0
	vt.margin.right = 2
	vt.cursor.row = 2
	vt.cursor.col = 0

	vt.ind()

	if got, want := trimScreenString(vt.String()), "AAAAAA\nAAAAAA\n   AAA"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 2 || vt.cursor.col != 0 {
		t.Fatalf("cursor = %d,%d, want 2,0", vt.cursor.row, vt.cursor.col)
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

func TestReverseIndexLeftRightMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	setScreenLine(vt.primaryScreen, 0, "ABC")
	setScreenLine(vt.primaryScreen, 1, "DEF")
	setScreenLine(vt.primaryScreen, 2, "GHI")
	vt.margin.left = 1
	vt.margin.right = 2
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.ri()

	if got, want := trimScreenString(vt.String()), "A\nDBC\nGEF\n HI"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestReverseIndexOutsideLeftRightMarginsDoesNotScrollRegion(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	setScreenLine(vt.primaryScreen, 0, "ABC")
	setScreenLine(vt.primaryScreen, 1, "DEF")
	setScreenLine(vt.primaryScreen, 2, "GHI")
	vt.margin.left = 1
	vt.margin.right = 2
	vt.cursor.row = 0
	vt.cursor.col = 0

	vt.ri()

	if got, want := trimScreenString(vt.String()), "ABC\nDEF\nGHI"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestReverseIndexAboveScrollRegionClampsToTopOfScreen(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 0
	vt.cursor.col = 0
	vt.lastCol = true

	vt.ri()

	if got, want := vt.cursor.row, row(0); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if vt.lastCol {
		t.Fatal("reverse index did not reset pending wrap")
	}
}

func TestReverseIndexInsideScrollRegionMovesUpWithoutScrolling(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 3
	vt.cursor.col = 0

	vt.ri()

	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestNextLineUsesCarriageReturnSemantics(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.left = 2
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.nel()

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestNextLineOriginModeUsesLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.mode.decom = true
	vt.margin.left = 2
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.nel()

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestScrollDownMovesHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.osc("8;id=link;https://example.com")
	printText(vt, "ABC")
	vt.osc("8;;")
	vt.cr()
	vt.lf()
	printText(vt, "DEF")

	vt.scrollDown(1)

	for col := column(0); col < 3; col += 1 {
		if got, want := vt.primaryScreen.cell(1, col).Hyperlink, "https://example.com"; got != want {
			t.Fatalf("moved hyperlink at col %d = %q, want %q", col, got, want)
		}
	}
	for col := column(0); col < column(vt.width()); col += 1 {
		if got := vt.primaryScreen.cell(0, col).Hyperlink; got != "" {
			t.Fatalf("blank row hyperlink at col %d = %q, want empty", col, got)
		}
	}
}

func TestScrollDownLeftRightRegionMovesHyperlinks(t *testing.T) {
	vt := New()
	vt.resize(6, 3)
	vt.osc("8;id=link;https://example.com")
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	for col := column(0); col < 6; col += 1 {
		c := vt.primaryScreen.cell(0, col)
		c.Hyperlink = vt.cursor.Hyperlink
		c.HyperlinkParams = vt.cursor.HyperlinkParams
	}
	vt.osc("8;;")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3

	vt.scrollDown(1)

	for col := column(0); col < column(vt.width()); col += 1 {
		got := vt.primaryScreen.cell(0, col).Hyperlink
		if col >= 1 && col <= 3 {
			if got != "" {
				t.Fatalf("blanked region hyperlink at col %d = %q, want empty", col, got)
			}
			continue
		}
		if got != "https://example.com" {
			t.Fatalf("preserved outside hyperlink at col %d = %q, want link", col, got)
		}
	}
	for col := column(1); col <= 3; col += 1 {
		if got, want := vt.primaryScreen.cell(1, col).Hyperlink, "https://example.com"; got != want {
			t.Fatalf("moved region hyperlink at col %d = %q, want %q", col, got, want)
		}
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

func TestDeleteLinesLeftRightRegionFromTop(t *testing.T) {
	vt := New()
	vt.resize(10, 4)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 0
	vt.cursor.col = 1

	vt.dl(1)

	if got, want := trimScreenString(vt.String()), "AEF423\nDHI756\nG   89"; got != want {
		t.Fatalf("screen mismatch after top DL: got %q want %q", got, want)
	}
}

func TestDeleteLinesLeftRightRegionHighCount(t *testing.T) {
	vt := New()
	vt.resize(10, 4)
	setScreenLine(vt.primaryScreen, 0, "ABC123")
	setScreenLine(vt.primaryScreen, 1, "DEF456")
	setScreenLine(vt.primaryScreen, 2, "GHI789")
	vt.margin.left = 1
	vt.margin.right = 3
	vt.cursor.row = 1
	vt.cursor.col = 1

	vt.dl(100)

	if got, want := trimScreenString(vt.String()), "ABC123\nD   56\nG   89"; got != want {
		t.Fatalf("screen mismatch after high-count DL: got %q want %q", got, want)
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
