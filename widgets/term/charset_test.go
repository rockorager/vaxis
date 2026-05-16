package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis/ansi"
)

func TestCharsetDesignationDoesNotInvokeG1G2G3(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(testESC('0', '*'))
	vt.update(testESC('0', '+'))
	vt.update(testPrint("`"))
	vt.update(testESC('0', '('))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "`◆  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCharsetSingleShiftRestoresPreviousSelection(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(ansi.C0(0x0E))
	vt.update(testPrint("`"))
	vt.update(testESC('B', '*'))
	vt.update(testESC('N'))
	vt.update(testPrint("`"))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "◆`◆ "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g1 {
		t.Fatalf("single shift restored %v, want G1", vt.charsets.selected)
	}
	if vt.charsets.singleShift {
		t.Fatal("single shift remained active after printing one graphic character")
	}
}

func TestCharsetNonASCIIMapsToSpaceInLegacyCharset(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testPrint("😀"))
	vt.update(testPrint("`"))

	if got, want := vt.String(), " ◆  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCharsetDECSpecialUnderscoreMatchesGhostty(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testPrint("_"))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "_◆  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCharsetUnsupportedDesignationSlotsAreIgnoredLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '-'))
	vt.update(testESC('0', '.'))
	vt.update(testESC('0', '/'))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "`   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.designations[g1] != ascii {
		t.Fatalf("G1 designation = %v, want ASCII", vt.charsets.designations[g1])
	}
	if vt.charsets.designations[g2] != ascii {
		t.Fatalf("G2 designation = %v, want ASCII", vt.charsets.designations[g2])
	}
	if vt.charsets.designations[g3] != ascii {
		t.Fatalf("G3 designation = %v, want ASCII", vt.charsets.designations[g3])
	}
}

func TestCharsetUnsupportedFinalIsIgnoredLikeGhostty(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', '('))
	vt.update(testESC('Z', '('))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "◆   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCharsetLockingShiftG2G3(t *testing.T) {
	vt := New()
	vt.resize(6, 1)

	vt.update(testESC('0', '*'))
	vt.update(testESC('0', '+'))
	vt.update(testPrint("`"))
	vt.update(testESC('n'))
	vt.update(testPrint("`"))
	vt.update(testPrint("`"))
	vt.update(testESC('o'))
	vt.update(testPrint("`"))
	vt.update(ansi.C0(0x0F))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "`◆◆◆` "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("SI selected %v, want G0", vt.charsets.selected)
	}
}

func TestCharsetRightLockingShiftsUpdateGR(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	if vt.charsets.gr != g2 {
		t.Fatalf("default GR = %v, want G2", vt.charsets.gr)
	}

	vt.update(testESC('~'))
	if vt.charsets.gr != g1 {
		t.Fatalf("LS1R selected GR %v, want G1", vt.charsets.gr)
	}

	vt.update(testESC('}'))
	if vt.charsets.gr != g2 {
		t.Fatalf("LS2R selected GR %v, want G2", vt.charsets.gr)
	}

	vt.update(testESC('|'))
	if vt.charsets.gr != g3 {
		t.Fatalf("LS3R selected GR %v, want G3", vt.charsets.gr)
	}
}

func TestCharsetSaveRestorePreservesGR(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('|'))
	vt.update(testESC('7'))
	vt.update(testESC('~'))
	vt.update(testESC('8'))

	if vt.charsets.gr != g3 {
		t.Fatalf("DECRC restored GR %v, want G3", vt.charsets.gr)
	}
}

func TestSaveRestoreCursorPreservesCharsetState(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(ansi.C0(0x0E))
	vt.update(testPrint(" "))
	vt.update(testESC('7'))
	vt.update(ansi.C0(0x0F))
	vt.update(testPrint("`"))
	vt.update(testESC('8'))
	vt.update(testPrint("`"))

	if got, want := vt.String(), " ◆  "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g1 {
		t.Fatalf("DECRC restored %v, want G1", vt.charsets.selected)
	}
}

func TestRISResetsCharsetState(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('0', ')'))
	vt.update(ansi.C0(0x0E))
	vt.update(testESC('c'))
	vt.update(testPrint("`"))

	if got, want := vt.String(), "`   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.charsets.selected != g0 {
		t.Fatalf("RIS selected charset = %v, want G0", vt.charsets.selected)
	}
	if vt.charsets.gr != g2 {
		t.Fatalf("RIS GR charset = %v, want G2", vt.charsets.gr)
	}
}
