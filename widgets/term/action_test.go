package term

import (
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ansi"
)

func testESC(final rune, intermediates ...rune) ansi.ESC {
	var seq ansi.ESC
	seq.Final = final
	seq.NumIntermediate = len(intermediates)
	copy(seq.Intermediate[:], intermediates)
	return seq
}

func testCSI(final rune, params []uint32, intermediates ...rune) ansi.CSI {
	var seq ansi.CSI
	seq.Final = final
	seq.NumParameters = len(params)
	if len(params) <= len(seq.Parameters) {
		copy(seq.Parameters[:], params)
	} else {
		seq.ExtraParameters = append([]uint32(nil), params...)
	}
	seq.NumIntermediate = len(intermediates)
	copy(seq.Intermediate[:], intermediates)
	return seq
}

func testPrint(s string) ansi.Print {
	return ansi.Print{Grapheme: s, Width: 1}
}

func TestLockingShiftInSelectsG0(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(ansi.C0(0x0E))
	vt.update(testPrint("q"))
	vt.update(ansi.C0(0x0F))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "─q  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("SI selected %v, want G0", vt.charsets.selected)
	}
}

func TestSingleShiftAppliesToOneGraphicCharacter(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testPrint("q"))
	vt.update(testESC('0', '*'))
	vt.update(testESC('N'))
	vt.update(testPrint("q"))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "q─q "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("single shift restored %v, want G0", vt.charsets.selected)
	}
	if vt.charsets.singleShift {
		t.Fatal("single shift remained active after printing one graphic character")
	}
}

func TestSS3AppliesG3ToOneGraphicCharacter(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testPrint("q"))
	vt.update(testESC('0', '+'))
	vt.update(ansi.SS3('q'))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "q─q "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("SS3 restored %v, want G0", vt.charsets.selected)
	}
	if vt.charsets.singleShift {
		t.Fatal("SS3 single shift remained active after printing one graphic character")
	}
}

func TestLockingShiftTwoAndThreeSelectGCharsets(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '*'))
	vt.update(testESC('0', '+'))
	vt.update(testESC('n'))
	vt.update(testPrint("q"))
	vt.update(testESC('o'))
	vt.update(testPrint("q"))

	if got, want := vt.String(), "──  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g3 {
		t.Fatalf("LS3 selected %v, want G3", vt.charsets.selected)
	}
}

func TestCharsetBritishDesignation(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('A', '('))
	vt.update(testPrint("#"))

	if got, want := vt.String(), "£   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestTableCharsetPrintsNonASCIIAsSpace(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testPrint("😀"))

	if got, want := vt.String(), "    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorStyleIgnoresInvalidValues(t *testing.T) {
	vt := New()
	vt.cursor.style = vaxis.CursorBlock

	vt.update(testCSI('q', []uint32{5}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style = %d, want %d", got, want)
	}

	vt.update(testCSI('q', []uint32{9}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style after invalid value = %d, want %d", got, want)
	}

	vt.update(testCSI('q', []uint32{1, 2}, ' '))
	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeamBlinking); got != want {
		t.Fatalf("cursor style after invalid parameter count = %d, want %d", got, want)
	}
}

func TestCursorStyleRequiresSpaceIntermediate(t *testing.T) {
	vt := New()
	vt.cursor.style = vaxis.CursorBlock

	vt.update(testCSI('q', nil))
	vt.update(testCSI('q', []uint32{5}))

	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBlock); got != want {
		t.Fatalf("cursor style = %d, want %d", got, want)
	}
}

func TestCursorStyleDefaultMapsToSteadyBlock(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "missing"},
		{name: "zero", params: []uint32{0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.cursor.style = vaxis.CursorBeamBlinking
			vt.mode.cursorBlinking = true

			vt.update(testCSI('q', tt.params, ' '))

			if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBlock); got != want {
				t.Fatalf("cursor style = %d, want %d", got, want)
			}
			if vt.mode.cursorBlinking {
				t.Fatal("default cursor style kept blink mode")
			}
		})
	}
}

func TestEffectiveCursorStyleFollowsBlinkMode(t *testing.T) {
	tests := []struct {
		name     string
		style    vaxis.CursorStyle
		blinking bool
		want     vaxis.CursorStyle
	}{
		{name: "default steady", style: vaxis.CursorDefault, want: vaxis.CursorDefault},
		{name: "default blinking", style: vaxis.CursorDefault, blinking: true, want: vaxis.CursorBlockBlinking},
		{name: "block steady", style: vaxis.CursorBlockBlinking, want: vaxis.CursorBlock},
		{name: "underline blinking", style: vaxis.CursorUnderline, blinking: true, want: vaxis.CursorUnderlineBlinking},
		{name: "beam steady", style: vaxis.CursorBeamBlinking, want: vaxis.CursorBeam},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cursorStyleWithBlink(tt.style, tt.blinking); got != tt.want {
				t.Fatalf("cursor style = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRISClearsCursorPen(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testCSI('m', []uint32{1, 31, 44}))
	vt.cursor.protected = true
	vt.update(testCSI('q', []uint32{5}, ' '))
	vt.update(testESC('c'))

	if vt.cursor.Attribute != 0 || vt.cursor.Foreground != 0 || vt.cursor.Background != 0 {
		t.Fatalf("cursor pen after RIS = attr %d fg %v bg %v, want zero", vt.cursor.Attribute, vt.cursor.Foreground, vt.cursor.Background)
	}
	if vt.cursor.protected {
		t.Fatal("cursor protection after RIS remained enabled")
	}
	if vt.cursor.style != vaxis.CursorDefault {
		t.Fatalf("cursor style after RIS = %d, want default", vt.cursor.style)
	}
}

func TestRISClearsSavedCursorPen(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testCSI('m', []uint32{1, 31, 44}))
	vt.update(testCSI('q', []uint32{5}, ' '))
	vt.update(testESC('7'))
	vt.update(testESC('c'))
	vt.update(testESC('8'))

	if vt.cursor.Attribute != 0 || vt.cursor.Foreground != 0 || vt.cursor.Background != 0 {
		t.Fatalf("restored cursor pen after RIS = attr %d fg %v bg %v, want zero", vt.cursor.Attribute, vt.cursor.Foreground, vt.cursor.Background)
	}
	if vt.cursor.style != vaxis.CursorDefault {
		t.Fatalf("restored cursor style after RIS = %d, want default", vt.cursor.style)
	}
}

func TestRestoreCursorDoesNotRestoreVisualCursorStyle(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testCSI('q', []uint32{5}, ' '))
	vt.update(testESC('7'))
	vt.update(testCSI('q', []uint32{6}, ' '))
	vt.update(testESC('8'))

	if got, want := vt.cursor.style, vaxis.CursorStyle(vaxis.CursorBeam); got != want {
		t.Fatalf("visual cursor style after DECRC = %d, want %d", got, want)
	}
	if vt.mode.cursorBlinking {
		t.Fatal("DECRC restored cursor blinking mode")
	}
}

func TestSaveRestoreCursorRestoresSGRPenAndOriginMode(t *testing.T) {
	vt := New()
	vt.resize(8, 4)
	vt.margin.top = 1
	vt.margin.bottom = 3

	vt.update(testCSI('m', []uint32{1, 31, 44}))
	vt.update(testCSI('h', []uint32{6}, '?'))
	vt.update(testESC('7'))
	vt.update(testCSI('m', nil))
	vt.update(testCSI('l', []uint32{6}, '?'))
	vt.update(testESC('8'))

	if vt.cursor.Attribute&vaxis.AttrBold == 0 {
		t.Fatal("DECRC did not restore bold SGR pen")
	}
	if got, want := vt.cursor.Foreground, vaxis.IndexColor(1); got != want {
		t.Fatalf("DECRC foreground = %v, want %v", got, want)
	}
	if got, want := vt.cursor.Background, vaxis.IndexColor(4); got != want {
		t.Fatalf("DECRC background = %v, want %v", got, want)
	}
	if !vt.mode.decom {
		t.Fatal("DECRC did not restore origin mode")
	}
}

func TestSaveRestoreCursorDoesNotRestoreWraparoundMode(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('7'))
	vt.update(testCSI('l', []uint32{7}, '?'))
	vt.update(testESC('8'))

	if vt.mode.decawm {
		t.Fatal("DECRC restored wraparound mode")
	}
}

func TestOriginModeMovesCursorToHome(t *testing.T) {
	vt := New()
	vt.resize(8, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 4
	vt.cursor.col = 6

	vt.update(testCSI('h', []uint32{6}, '?'))
	if !vt.mode.decom {
		t.Fatal("DEC origin mode was not enabled")
	}
	if vt.cursor.row != vt.margin.top || vt.cursor.col != vt.margin.left {
		t.Fatalf("cursor after DECSET 6 = %d,%d, want %d,%d", vt.cursor.row, vt.cursor.col, vt.margin.top, vt.margin.left)
	}

	vt.cursor.row = 4
	vt.cursor.col = 6
	vt.update(testCSI('l', []uint32{6}, '?'))
	if vt.mode.decom {
		t.Fatal("DEC origin mode was not disabled")
	}
	if vt.cursor.row != 0 || vt.cursor.col != 0 {
		t.Fatalf("cursor after DECRST 6 = %d,%d, want 0,0", vt.cursor.row, vt.cursor.col)
	}
}
