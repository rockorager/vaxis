package term

import "testing"

func TestDECSCADECSELProtectsCells(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))
	vt.update(testPrint("d"))
	vt.update(testPrint("e"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}, '?'))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
	if !vt.activeScreen.cell(0, 1).protected {
		t.Fatal("protected cell lost protection")
	}
}

func TestDECSCADefaultAndTwoDisableProtection(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testCSI('q', nil, '"'))
	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testCSI('q', []uint32{2}, '"'))
	vt.update(testPrint("b"))

	if vt.activeScreen.cell(0, 0).protected {
		t.Fatal("default DECSCA did not disable protection")
	}
	if vt.activeScreen.cell(0, 1).protected {
		t.Fatal("DECSCA 2 did not disable protection")
	}
}

func TestDECProtectedCellsIgnoredByNormalEraseLine(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}))

	if got, want := vt.String(), "     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestISOProtectedCellsRespectedByNormalEraseLine(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testESC('V'))
	vt.update(testPrint("b"))
	vt.update(testESC('W'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('K', []uint32{2}))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestISOProtectedCellsRespectedByEraseCharacters(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testESC('V'))
	vt.update(testPrint("b"))
	vt.update(testESC('W'))
	vt.update(testPrint("c"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('X', []uint32{3}))

	if got, want := vt.String(), " b   "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDECMostRecentMakesNormalEraseCharactersIgnoreProtectedCells(t *testing.T) {
	vt := New()
	vt.resize(5, 1)

	vt.update(testPrint("a"))
	vt.update(testESC('V'))
	vt.update(testPrint("b"))
	vt.update(testESC('W'))
	vt.update(testPrint("c"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testCSI('q', []uint32{0}, '"'))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('X', []uint32{3}))

	if got, want := vt.String(), "     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDECSEDProtectsCells(t *testing.T) {
	vt := New()
	vt.resize(4, 2)

	vt.update(testPrint("a"))
	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testPrint("b"))
	vt.update(testCSI('q', []uint32{0}, '"'))
	vt.update(testPrint("c"))
	vt.update(testPrint("d"))
	vt.update(testCSI('H', []uint32{2, 1}))
	vt.update(testPrint("e"))
	vt.update(testPrint("f"))
	vt.update(testPrint("g"))
	vt.update(testPrint("h"))

	vt.update(testCSI('H', []uint32{1, 1}))
	vt.update(testCSI('J', []uint32{0}, '?'))

	if got, want := vt.String(), " b  \n    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestDECSEDProtectedRouting(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
		want   string
	}{
		{name: "default", params: nil, want: " b  \n    "},
		{name: "below", params: []uint32{0}, want: " b  \n    "},
		{name: "above", params: []uint32{1}, want: " bcd\nefgh"},
		{name: "complete", params: []uint32{2}, want: " b  \n    "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(4, 2)

			vt.update(testPrint("a"))
			vt.update(testCSI('q', []uint32{1}, '"'))
			vt.update(testPrint("b"))
			vt.update(testCSI('q', []uint32{0}, '"'))
			vt.update(testPrint("c"))
			vt.update(testPrint("d"))
			vt.update(testCSI('H', []uint32{2, 1}))
			vt.update(testPrint("e"))
			vt.update(testPrint("f"))
			vt.update(testPrint("g"))
			vt.update(testPrint("h"))
			vt.update(testCSI('H', []uint32{1, 1}))

			vt.update(testCSI('J', tt.params, '?'))

			if got := vt.String(); got != tt.want {
				t.Fatalf("screen mismatch: got %q want %q", got, tt.want)
			}
			if !vt.activeScreen.cell(0, 1).protected {
				t.Fatal("protected cell lost protection")
			}
		})
	}
}

func TestDECSELProtectedRouting(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
		want   string
	}{
		{name: "default", params: nil, want: "ab   "},
		{name: "right", params: []uint32{0}, want: "ab   "},
		{name: "left", params: []uint32{1}, want: " b d "},
		{name: "complete", params: []uint32{2}, want: " b   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(5, 1)

			vt.update(testPrint("a"))
			vt.update(testCSI('q', []uint32{1}, '"'))
			vt.update(testPrint("b"))
			vt.update(testCSI('q', []uint32{0}, '"'))
			vt.update(testPrint("c"))
			vt.update(testPrint("d"))
			vt.update(testCSI('H', []uint32{1, 3}))

			vt.update(testCSI('K', tt.params, '?'))

			if got := vt.String(); got != tt.want {
				t.Fatalf("screen mismatch: got %q want %q", got, tt.want)
			}
			if !vt.activeScreen.cell(0, 1).protected {
				t.Fatal("protected cell lost protection")
			}
		})
	}
}

func TestProtectedEraseIgnoresInvalidIntermediates(t *testing.T) {
	vt := New()
	vt.resize(5, 1)
	printText(vt, "abcde")

	vt.update(testCSI('H', []uint32{1, 3}))
	vt.update(testCSI('J', []uint32{0}, '>'))
	vt.update(testCSI('K', []uint32{1}, '<'))

	if got, want := vt.String(), "abcde"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestSaveCursorRestoresProtectedPen(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testESC('V'))
	vt.update(testESC('7'))
	vt.update(testESC('W'))
	vt.update(testESC('8'))
	vt.update(testPrint("x"))

	if !vt.activeScreen.cell(0, 0).protected {
		t.Fatal("restored cursor did not restore protected pen")
	}
}

func TestDECSCAIgnoresInvalidParameterLists(t *testing.T) {
	vt := New()
	vt.resize(4, 1)

	vt.update(testCSI('q', []uint32{1}, '"'))
	vt.update(testCSI('q', []uint32{0, 1}, '"'))
	vt.update(testPrint("x"))

	if !vt.activeScreen.cell(0, 0).protected {
		t.Fatal("multi-parameter DECSCA changed protected pen")
	}

	vt.update(testCSI('q', []uint32{3}, '"'))
	vt.update(testPrint("y"))

	if !vt.activeScreen.cell(0, 1).protected {
		t.Fatal("invalid DECSCA value changed protected pen")
	}
}
