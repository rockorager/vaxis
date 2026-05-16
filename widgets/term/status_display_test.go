package term

import "testing"

func TestStatusDisplaySuppressesPrintableOutput(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testCSI('}', []uint32{1}, '$'))
	vt.update(testPrint("b"))
	vt.update(testCSI('}', []uint32{0}, '$'))
	vt.update(testPrint("c"))

	if got, want := vt.String(), "ac   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestFullResetRestoresMainStatusDisplay(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('}', []uint32{1}, '$'))
	vt.update(testESC('c'))
	vt.update(testPrint("x"))

	if got, want := vt.String(), "x    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInvalidStatusDisplaySelectionIgnored(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('}', []uint32{2}, '$'))
	vt.update(testPrint("x"))

	if got, want := vt.String(), "x    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
