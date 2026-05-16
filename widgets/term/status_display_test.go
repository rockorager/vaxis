package term

import "testing"

func TestStatusDisplayBlackholesPrintedOutput(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "abc")
	vt.cursor.col = 0

	vt.update(testCSI('}', []uint32{1}, '$'))
	printText(vt, "XYZ")

	if got, want := vt.String(), "abc  "; got != want {
		t.Fatalf("screen after status display print = %q, want %q", got, want)
	}

	vt.update(testCSI('}', []uint32{0}, '$'))
	printText(vt, "XYZ")

	if got, want := vt.String(), "XYZ  "; got != want {
		t.Fatalf("screen after main display print = %q, want %q", got, want)
	}
}

func TestRISResetsStatusDisplay(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('}', []uint32{1}, '$'))
	vt.update(testESC('c'))
	printText(vt, "abc")

	if got, want := vt.String(), "abc  "; got != want {
		t.Fatalf("screen after RIS = %q, want %q", got, want)
	}
}

func TestInvalidDECSASDDoesNotChangeStatusDisplay(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	vt.update(testCSI('}', []uint32{1}, '$'))

	vt.update(testCSI('}', nil, '$'))
	vt.update(testCSI('}', []uint32{2}, '$'))
	vt.update(testCSI('}', []uint32{0}, '?'))
	printText(vt, "abc")

	if got, want := vt.String(), "     "; got != want {
		t.Fatalf("screen after invalid DECSASD = %q, want %q", got, want)
	}
}
