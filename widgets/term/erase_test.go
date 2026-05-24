package term

import (
	"testing"

	"go.rockorager.dev/vaxis/ansi"
)

func TestEraseDisplayIgnoresInvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "multiple", params: []uint32{2, 1}},
		{name: "unknown", params: []uint32{4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 2)
			setScreenLine(vt.primaryScreen, 0, "ABC")
			setScreenLine(vt.primaryScreen, 1, "DEF")

			vt.update(testCSI('J', tt.params))

			if got, want := vt.String(), "ABC\nDEF"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
		})
	}
}

func TestEraseLineIgnoresInvalidParameters(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "multiple", params: []uint32{2, 1}},
		{name: "unknown", params: []uint32{3}},
		{name: "right unless pending wrap", params: []uint32{4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 1)
			setScreenLine(vt.primaryScreen, 0, "ABC")

			vt.update(testCSI('K', tt.params))

			if got, want := vt.String(), "ABC"; got != want {
				t.Fatalf("screen mismatch: got %q want %q", got, want)
			}
		})
	}
}

func TestEraseDisplayScrollCompleteClearsScreenAndScrollback(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	appendViewportLines(vt, "111", "222")
	setScreenLine(vt.primaryScreen, 0, "ABC")
	setScreenLine(vt.primaryScreen, 1, "DEF")
	vt.scrollOffset = 1

	vt.update(testCSI('J', []uint32{22}))

	if got, want := vt.String(), "   \n   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback len = %d, want 0", got)
	}
	if got := vt.scrollOffset; got != 0 {
		t.Fatalf("scroll offset = %d, want 0", got)
	}
}

func TestEraseDisplayCompleteAtSemanticPromptScrollClearsPrimary(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "out")
	vt.cr()
	vt.lf()
	vt.osc("133;P")
	printText(vt, "$")
	vt.update(testCSI('J', []uint32{2}))

	if got, want := vt.String(), "     \n     \n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackLen(), 2; got != want {
		t.Fatalf("scrollback len = %d, want %d", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(0), "out  "; got != want {
		t.Fatalf("scrollback row 0 = %q, want %q", got, want)
	}
	if got, want := vt.primaryScreen.scrollbackString(1), "$    "; got != want {
		t.Fatalf("scrollback row 1 = %q, want %q", got, want)
	}
}

func TestEraseDisplayCompleteAtSemanticPromptDoesNotScrollClearAlternate(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.update(testCSI('h', []uint32{1047}, '?'))
	vt.osc("133;P")
	printText(vt, "$")
	vt.update(testCSI('J', []uint32{2}))

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("primary scrollback len = %d, want 0", got)
	}
	if got := vt.altScreen.scrollbackLen(); got != 0 {
		t.Fatalf("alternate scrollback len = %d, want 0", got)
	}
}

func TestEraseDisplayCompleteWithoutSemanticPromptDoesNotScrollClear(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	printText(vt, "out")
	vt.cr()
	vt.lf()
	printText(vt, "$")
	vt.update(testCSI('J', []uint32{2}))

	if got := vt.primaryScreen.scrollbackLen(); got != 0 {
		t.Fatalf("scrollback len = %d, want 0", got)
	}
}

func TestEraseDisplayBelowSplitsWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	printText(vt, "AB")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "C")
	vt.cr()
	vt.lf()
	printText(vt, "DE")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "F")
	vt.cr()
	vt.lf()
	printText(vt, "GH")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "I")
	vt.update(testCSI('H', []uint32{2, 4}))

	vt.ed(0, false)

	if got, want := trimScreenString(vt.String()), "AB橋 C\nDE"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.primaryScreen.cell(1, 2)
	tail := vt.primaryScreen.cell(1, 3)
	if head.Grapheme != "" || head.Width != 0 || tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("split wide cells after ED below = head %#v tail %#v, want blanks", head.Character, tail.Character)
	}
}

func TestEraseDisplayAboveSplitsWideCharacter(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	printText(vt, "AB")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "C")
	vt.cr()
	vt.lf()
	printText(vt, "DE")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "F")
	vt.cr()
	vt.lf()
	printText(vt, "GH")
	vt.update(ansi.Print{Grapheme: "橋", Width: 2})
	printText(vt, "I")
	vt.update(testCSI('H', []uint32{2, 3}))

	vt.ed(1, false)

	if got, want := trimScreenString(vt.String()), "\n    F\nGH橋 I"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	head := vt.primaryScreen.cell(1, 2)
	tail := vt.primaryScreen.cell(1, 3)
	if head.Grapheme != "" || head.Width != 0 || tail.Grapheme != "" || tail.Width != 0 {
		t.Fatalf("split wide cells after ED above = head %#v tail %#v, want blanks", head.Character, tail.Character)
	}
}
