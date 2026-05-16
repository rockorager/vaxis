package term

import "testing"

func TestMode47AltScreenRetainsContent(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{47}, '?'))
	if !vt.mode.smcup {
		t.Fatal("mode 47 did not enter alternate screen")
	}
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("initial alt screen = %q, want %q", got, want)
	}

	printText(vt, "2B")
	if got, want := vt.String(), "  2B \n     "; got != want {
		t.Fatalf("alt screen after write = %q, want %q", got, want)
	}

	vt.decrst(testCSI('l', []uint32{47}, '?'))
	if vt.mode.smcup {
		t.Fatal("mode 47 did not leave alternate screen")
	}
	if got, want := vt.String(), "1A   \n     "; got != want {
		t.Fatalf("primary screen = %q, want %q", got, want)
	}

	vt.decset(testCSI('h', []uint32{47}, '?'))
	if got, want := vt.String(), "  2B \n     "; got != want {
		t.Fatalf("mode 47 did not retain alternate content: got %q want %q", got, want)
	}
}

func TestMode1047ClearsAltScreenOnExit(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{1047}, '?'))
	printText(vt, "2B")
	vt.decrst(testCSI('l', []uint32{1047}, '?'))
	vt.decset(testCSI('h', []uint32{1047}, '?'))

	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("mode 1047 did not clear alternate content: got %q want %q", got, want)
	}
}

func TestMode1049RestoresPrimaryCursorAndClearsAltOnEntry(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "1A")

	vt.decset(testCSI('h', []uint32{1049}, '?'))
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("initial 1049 alt screen = %q, want %q", got, want)
	}
	printText(vt, "2B")

	vt.decrst(testCSI('l', []uint32{1049}, '?'))
	printText(vt, "C")
	if got, want := vt.String(), "1AC  \n     "; got != want {
		t.Fatalf("primary screen after 1049 restore = %q, want %q", got, want)
	}

	vt.decset(testCSI('h', []uint32{1049}, '?'))
	if got, want := vt.String(), "     \n     "; got != want {
		t.Fatalf("mode 1049 did not clear alternate content on entry: got %q want %q", got, want)
	}
}

func TestAltScreenClearsViewportOffset(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	writeViewportLines(vt, "1ABCD", "2EFGH", "3IJKL", "4ABCD")
	vt.scrollViewport(1)

	vt.decset(testCSI('h', []uint32{1049}, '?'))

	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset = %d, want 0", got)
	}
}
