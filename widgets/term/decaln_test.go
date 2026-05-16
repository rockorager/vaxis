package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDECALNFillsScreenAndHomesCursor(t *testing.T) {
	vt := New()
	vt.resize(2, 2)
	printText(vt, "A")
	vt.cr()
	vt.lf()
	printText(vt, "B")

	vt.update(testESC('8', '#'))

	if got, want := vt.String(), "EE\nEE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}

func TestDECALNResetsMarginsAndOriginMode(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	vt.mode.decom = true
	vt.margin.top = 1
	vt.margin.bottom = 2
	vt.margin.left = 1
	vt.margin.right = 2

	vt.update(testESC('8', '#'))
	vt.scrollDown(1)

	if vt.mode.decom {
		t.Fatal("origin mode remained enabled")
	}
	if vt.margin.top != 0 || vt.margin.left != 0 || vt.margin.bottom != 2 || vt.margin.right != 2 {
		t.Fatalf("margins = top %d bottom %d left %d right %d, want full screen", vt.margin.top, vt.margin.bottom, vt.margin.left, vt.margin.right)
	}
	if got, want := vt.String(), "   \nEEE\nEEE"; got != want {
		t.Fatalf("screen mismatch after scroll down: got %q want %q", got, want)
	}
}

func TestDECALNClearsWrapMetadata(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "abcd")

	if !vt.activeScreen.row(0).wrapped || !vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("test setup did not create soft wrap metadata")
	}

	vt.update(testESC('8', '#'))

	if vt.activeScreen.row(0).wrapped {
		t.Fatal("DECALN kept wrapped metadata")
	}
	if vt.activeScreen.row(1).wrapContinuation {
		t.Fatal("DECALN kept wrap continuation metadata")
	}
}

func TestDECALNClearsStyleAttributesButPreservesColors(t *testing.T) {
	vt := New()
	vt.resize(2, 1)
	fg := vaxis.IndexColor(2)
	bg := vaxis.IndexColor(3)
	vt.cursor.Style = vaxis.Style{
		Foreground:      fg,
		Background:      bg,
		UnderlineColor:  vaxis.IndexColor(4),
		UnderlineStyle:  vaxis.UnderlineSingle,
		Attribute:       vaxis.AttrBold | vaxis.AttrItalic,
		Hyperlink:       "https://example.com",
		HyperlinkParams: "id=1",
	}
	vt.cursor.protected = true

	vt.update(testESC('8', '#'))

	c := vt.activeScreen.cell(0, 0)
	if got, want := c.Grapheme, "E"; got != want {
		t.Fatalf("DECALN cell = %q, want %q", got, want)
	}
	if got := c.Foreground; got != fg {
		t.Fatalf("DECALN foreground = %v, want %v", got, fg)
	}
	if got := c.Background; got != bg {
		t.Fatalf("DECALN background = %v, want %v", got, bg)
	}
	if c.Attribute != 0 || c.UnderlineStyle != vaxis.UnderlineOff || c.UnderlineColor != 0 || c.Hyperlink != "" || c.HyperlinkParams != "" {
		t.Fatalf("DECALN kept style attributes: %+v", c.Style)
	}
	if c.protected {
		t.Fatal("DECALN fill cell kept protected flag")
	}
	if vt.cursor.Foreground != fg || vt.cursor.Background != bg {
		t.Fatalf("DECALN cursor colors = %v/%v, want %v/%v", vt.cursor.Foreground, vt.cursor.Background, fg, bg)
	}
	if vt.cursor.Attribute != 0 || vt.cursor.UnderlineStyle != vaxis.UnderlineOff || vt.cursor.UnderlineColor != 0 || vt.cursor.Hyperlink != "" || vt.cursor.HyperlinkParams != "" {
		t.Fatalf("DECALN kept cursor style attributes: %+v", vt.cursor.Style)
	}
	if !vt.cursor.protected {
		t.Fatal("DECALN cleared cursor protected mode")
	}
}
