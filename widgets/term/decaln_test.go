package term

import "testing"

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
