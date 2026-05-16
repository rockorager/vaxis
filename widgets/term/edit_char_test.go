package term

import "testing"

func TestInsertBlanks(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(2)

	if got, want := trimScreenString(vt.String()), "  ABC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksPushesOffEnd(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(2)

	if got, want := trimScreenString(vt.String()), "  A"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksMoreThanLineWidth(t *testing.T) {
	vt := New()
	vt.resize(3, 2)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 1}))

	vt.ich(5)

	if got, want := trimScreenString(vt.String()), ""; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksAtLastColumnBlanksCell(t *testing.T) {
	vt := New()
	vt.resize(3, 1)
	printText(vt, "ABC")
	vt.update(testCSI('H', []uint32{1, 3}))

	vt.ich(1)

	if got, want := vt.String(), "AB "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertBlanksOutsideLeftRightRegionDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 5
	vt.lastCol = true

	vt.ich(2)

	if got, want := vt.String(), "ABC123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.lastCol {
		t.Fatal("insert blanks outside region did not reset pending wrap")
	}
}

func TestInsertBlanksInsideLeftRightRegion(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 2

	vt.ich(2)

	if got, want := vt.String(), "AB  C3"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDeleteCharsOutsideLeftRightRegionDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 5
	vt.lastCol = true

	vt.dch(2)

	if got, want := vt.String(), "ABC123"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if vt.lastCol {
		t.Fatal("delete chars outside region did not reset pending wrap")
	}
}

func TestDeleteCharsInsideLeftRightRegion(t *testing.T) {
	vt := New()
	vt.resize(6, 1)
	printText(vt, "ABC123")
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.col = 3

	vt.dch(1)

	if got, want := vt.String(), "ABC2 3"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}
