package term

import "testing"

func TestInsertLinesZeroParameterDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1
	vt.lastCol = true

	vt.update(testCSI('L', []uint32{0}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("IL zero parameter reset pending wrap")
	}
}

func TestDeleteLinesZeroParameterDoesNothing(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1
	vt.lastCol = true

	vt.update(testCSI('M', []uint32{0}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.lastCol {
		t.Fatal("DL zero parameter reset pending wrap")
	}
}

func TestInsertDeleteLinesDefaultParameterActsOnce(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1

	vt.update(testCSI('L', nil))
	if got, want := vt.String(), "AAA\n   \nBBB"; got != want {
		t.Fatalf("screen after default IL = %q want %q", got, want)
	}

	vt.update(testCSI('M', nil))
	if got, want := vt.String(), "AAA\nBBB\n   "; got != want {
		t.Fatalf("screen after default DL = %q want %q", got, want)
	}
}

func TestInsertDeleteLinesIgnoreMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(3, 3)
	setScreenLine(vt.primaryScreen, 0, "AAA")
	setScreenLine(vt.primaryScreen, 1, "BBB")
	setScreenLine(vt.primaryScreen, 2, "CCC")
	vt.cursor.row = 1

	vt.update(testCSI('L', []uint32{1, 1}))
	vt.update(testCSI('M', []uint32{1, 1}))

	if got, want := vt.String(), "AAA\nBBB\nCCC"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestInsertDeleteLinesOutsideRegionDoNotResetPendingWrap(t *testing.T) {
	tests := []struct {
		name  string
		final rune
	}{
		{name: "insert", final: 'L'},
		{name: "delete", final: 'M'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(3, 3)
			vt.margin.top = 1
			vt.margin.bottom = 2
			vt.cursor.row = 0
			vt.lastCol = true

			vt.update(testCSI(tt.final, nil))

			if !vt.lastCol {
				t.Fatal("operation outside region reset pending wrap")
			}
		})
	}
}
