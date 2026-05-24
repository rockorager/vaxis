package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestViewportReadsScrollback(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	vt.primaryScreen.setCell(0, 0, cellString("a"))
	vt.primaryScreen.setCell(1, 0, cellString("b"))

	vt.scrollUp(1)
	vt.scrollViewport(1)

	if got, want := lineString(vt.visibleLine(0)), "a  "; got != want {
		t.Fatalf("top visible line = %q, want %q", got, want)
	}
	if got, want := lineString(vt.visibleLine(1)), "b  "; got != want {
		t.Fatalf("bottom visible line = %q, want %q", got, want)
	}
}

func TestViewportKeyScrollsByPage(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	for i := 0; i < 5; i += 1 {
		vt.primaryScreen.setCell(0, 0, cellString("x"))
		vt.scrollUp(1)
	}

	handled := vt.handleViewportKey(vaxis.Key{
		Keycode:   vaxis.KeyPgUp,
		Modifiers: vaxis.ModShift,
	})

	if !handled {
		t.Fatal("Shift+PageUp was not handled by viewport")
	}
	if got, want := vt.scrollOffset, 2; got != want {
		t.Fatalf("scroll offset = %d, want %d", got, want)
	}
}

func TestViewportGhosttyScrollbackVariousCases(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")

	if got, want := viewportString(vt), "2EFGH\n3IJKL\n4ABCD"; got != want {
		t.Fatalf("active viewport = %q, want %q", got, want)
	}

	vt.scrollViewport(1)
	if got, want := viewportString(vt), "1ABCD\n2EFGH\n3IJKL"; got != want {
		t.Fatalf("scrolled viewport = %q, want %q", got, want)
	}

	vt.scrollViewport(1)
	if got, want := viewportString(vt), "1ABCD\n2EFGH\n3IJKL"; got != want {
		t.Fatalf("viewport beyond top = %q, want %q", got, want)
	}

	vt.scrollViewport(-1)
	if got, want := viewportString(vt), "2EFGH\n3IJKL\n4ABCD"; got != want {
		t.Fatalf("viewport at bottom = %q, want %q", got, want)
	}
}

func TestViewportGhosttyScrollbackDoesNotMoveWhenNotAtBottom(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")

	vt.scrollViewport(1)
	if got, want := viewportString(vt), "2EFGH\n3IJKL\n4ABCD"; got != want {
		t.Fatalf("initial scrolled viewport = %q, want %q", got, want)
	}

	vt.scrollUp(1)
	setScreenLine(vt.primaryScreen, 2, "6IJKL")
	if got, want := viewportString(vt), "2EFGH\n3IJKL\n4ABCD"; got != want {
		t.Fatalf("viewport after scrollback grow = %q, want %q", got, want)
	}
}

func TestViewportGhosttyScrollbackMultiRowDelta(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH", "6IJKL")

	vt.scrollViewport(3)
	if got, want := viewportString(vt), "1ABCD\n2EFGH\n3IJKL"; got != want {
		t.Fatalf("top viewport = %q, want %q", got, want)
	}

	vt.scrollViewport(-5)
	if got, want := vt.scrollOffset, 0; got != want {
		t.Fatalf("scroll offset after multi-row down = %d, want %d", got, want)
	}
	if got, want := viewportString(vt), "4ABCD\n5EFGH\n6IJKL"; got != want {
		t.Fatalf("active viewport = %q, want %q", got, want)
	}
}

func TestViewportClampsWhenScrollbackPruned(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.primaryScreen.state.scrollbackLimit = 2
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")

	vt.scrollViewport(2)
	if got, want := vt.scrollOffset, 2; got != want {
		t.Fatalf("initial scroll offset = %d, want %d", got, want)
	}

	appendViewportLines(vt, "6IJKL", "7ABCD", "8EFGH", "9IJKL")
	if got, want := vt.scrollOffset, 2; got != want {
		t.Fatalf("pruned scroll offset = %d, want %d", got, want)
	}
	if got, want := viewportString(vt), "5EFGH\n6IJKL\n7ABCD"; got != want {
		t.Fatalf("viewport after pruning = %q, want %q", got, want)
	}
}

func TestCursorHiddenWhenViewportScrolledBack(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	atomicStore(&vt.focused, true)
	vt.mode.dectcem = true
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.cursor.row = 2

	if _, _, ok := vt.cursorViewportPosition(); !ok {
		t.Fatal("cursor should be visible at active viewport")
	}

	vt.scrollViewport(1)

	if _, _, ok := vt.cursorViewportPosition(); ok {
		t.Fatal("cursor should be hidden while viewing scrollback")
	}
}

func TestCursorMappedWhenScrolledViewportIncludesCursorRow(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	atomicStore(&vt.focused, true)
	vt.mode.dectcem = true
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.cursor.row = 0
	vt.cursor.col = 2

	vt.scrollViewport(1)

	col, rw, ok := vt.cursorViewportPosition()
	if !ok {
		t.Fatal("cursor should be visible when active cursor row remains in viewport")
	}
	if col != 2 || rw != 1 {
		t.Fatalf("cursor viewport position = %d,%d, want 2,1", col, rw)
	}
}

func TestCursorHiddenDuringSynchronizedOutput(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	atomicStore(&vt.focused, true)
	vt.mode.dectcem = true

	if _, _, ok := vt.cursorViewportPosition(); !ok {
		t.Fatal("cursor should be visible before synchronized output")
	}

	vt.setSynchronizedOutput(true)
	defer vt.setSynchronizedOutput(false)

	if _, _, ok := vt.cursorViewportPosition(); ok {
		t.Fatal("cursor should be hidden during synchronized output")
	}
}

func TestKeyInputScrollsViewportToBottom(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.Update(vaxis.Key{Keycode: 'x', Text: "x", EventType: vaxis.EventPress})

	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset after key input = %d, want 0", got)
	}
}

func TestViewportKeyDoesNotForceBottom(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")
	vt.scrollViewport(1)

	vt.Update(vaxis.Key{Keycode: vaxis.KeyPgUp, Modifiers: vaxis.ModShift})

	if got, want := vt.scrollOffset, 2; got != want {
		t.Fatalf("scroll offset after viewport key = %d, want %d", got, want)
	}
}

func TestBracketedPasteWrappersScrollViewportToBottom(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.mode.paste = true
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.Update(vaxis.PasteStartEvent{})
	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset after paste start = %d, want 0", got)
	}

	vt.scrollViewport(1)
	vt.Update(vaxis.PasteEndEvent{})
	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset after paste end = %d, want 0", got)
	}
}

func TestOutputDoesNotScrollViewportToBottom(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.update(testPrint("x"))

	if got, want := vt.scrollOffset, 1; got != want {
		t.Fatalf("scroll offset after output = %d, want %d", got, want)
	}
}

func TestEraseDisplayClearsScrollback(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD", "5EFGH")
	vt.scrollViewport(2)

	vt.ed(3, false)

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback length = %d, want 0", got)
	}
	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset = %d, want 0", got)
	}
	if got, want := viewportString(vt), "3IJKL\n4ABCD\n5EFGH"; got != want {
		t.Fatalf("viewport after scrollback clear = %q, want %q", got, want)
	}
}

func writeViewportLines(vt *Model, lines ...string) {
	for i, line := range lines {
		if i < vt.height() {
			setScreenLine(vt.primaryScreen, i, line)
			continue
		}
		vt.scrollUp(1)
		setScreenLine(vt.primaryScreen, vt.height()-1, line)
	}
}

func appendViewportLines(vt *Model, lines ...string) {
	for _, line := range lines {
		vt.scrollUp(1)
		setScreenLine(vt.primaryScreen, vt.height()-1, line)
	}
}

func setScreenLine(screen screenBuffer, r int, text string) {
	line := screen.line(row(r))
	for col := range line {
		line[col].erase(0)
	}
	for col, ch := range text {
		if col >= len(line) {
			return
		}
		line[col] = cellString(string(ch))
	}
}

func viewportString(vt *Model) string {
	out := ""
	for r := 0; r < vt.height(); r += 1 {
		if r > 0 {
			out += "\n"
		}
		out += lineString(vt.visibleLine(r))
	}
	return out
}

func lineString(line []cell) string {
	out := ""
	for i := range line {
		out += line[i].rune()
	}
	return out
}
