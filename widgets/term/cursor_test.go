package term

import "testing"

func TestCursorForwardOutsideRightMarginStaysOutside(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.right = 2
	vt.cursor.col = 4

	vt.cuf(100)

	if got, want := vt.cursor.col, column(4); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorForwardInsideRightMarginClampsToMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.right = 2

	vt.cuf(100)

	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorBackwardIgnoresLeftMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.left = 2
	vt.cursor.col = 3

	vt.cub(100)

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorDownOutsideBottomMarginStaysOutside(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.bottom = 2
	vt.cursor.row = 4

	vt.cud(100)

	if got, want := vt.cursor.row, row(4); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestCursorDownInsideBottomMarginClampsToMargin(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.bottom = 2

	vt.cud(100)

	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestBackspaceDoesNotReverseWrapByDefault(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.cursor.row = 1

	vt.bs()

	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
}

func TestCursorBackwardPendingWrapWithoutReverseWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)

	printText(vt, "ABCDE")
	vt.cub(1)
	printText(vt, "X")

	if got, want := vt.String(), "ABCXE\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardPendingWrapWithReverseWrap(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE")
	vt.cub(1)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDX\n     "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardReverseWrapUsesSoftWrappedRows(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE1")
	vt.cub(2)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDX\n1    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorBackwardReverseWrapStopsAtUnwrappedRow(t *testing.T) {
	vt := New()
	vt.resize(5, 2)
	vt.update(testCSI('h', []uint32{45}, '?'))

	printText(vt, "ABCDE")
	vt.cr()
	vt.lf()
	printText(vt, "1")
	vt.cub(2)
	printText(vt, "X")

	if got, want := vt.String(), "ABCDE\nX    "; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorNextLineDoesNotScroll(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	printText(vt, "ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "EFGH")

	vt.cnl(1)
	printText(vt, "X")

	if got, want := vt.String(), "ABCD\nXFGH"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorPreviousLineDoesNotScroll(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	printText(vt, "ABCD")
	vt.cr()
	vt.lf()
	printText(vt, "EFGH")
	vt.cursor.row = 0
	vt.cursor.col = 2

	vt.cpl(1)
	printText(vt, "X")

	if got, want := vt.String(), "XBCD\nEFGH"; got != want {
		t.Fatalf("screen mismatch: got %q want %q", got, want)
	}
}

func TestCursorNextPreviousLineIgnoreMultipleParameters(t *testing.T) {
	vt := New()
	vt.resize(4, 2)
	vt.cursor.row = 1
	vt.cursor.col = 2

	vt.update(testCSI('E', []uint32{1, 1}))
	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row after invalid CNL = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col after invalid CNL = %d, want %d", got, want)
	}

	vt.update(testCSI('F', []uint32{1, 1}))
	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row after invalid CPL = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col after invalid CPL = %d, want %d", got, want)
	}
}

func TestCursorMovementIgnoresMultipleParameters(t *testing.T) {
	tests := []struct {
		name   string
		final  rune
		params []uint32
	}{
		{name: "CUU", final: 'A', params: []uint32{1, 1}},
		{name: "CUU alias", final: 'k', params: []uint32{1, 1}},
		{name: "CUD", final: 'B', params: []uint32{1, 1}},
		{name: "CUF", final: 'C', params: []uint32{1, 1}},
		{name: "CUB", final: 'D', params: []uint32{1, 1}},
		{name: "CUB alias", final: 'j', params: []uint32{1, 1}},
		{name: "CHA", final: 'G', params: []uint32{1, 1}},
		{name: "HPA", final: '`', params: []uint32{1, 1}},
		{name: "HPR", final: 'a', params: []uint32{1, 1}},
		{name: "VPA", final: 'd', params: []uint32{1, 1}},
		{name: "VPR", final: 'e', params: []uint32{1, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.resize(4, 4)
			vt.cursor.row = 2
			vt.cursor.col = 2

			vt.update(testCSI(tt.final, tt.params))

			if got, want := vt.cursor.row, row(2); got != want {
				t.Fatalf("cursor row = %d, want %d", got, want)
			}
			if got, want := vt.cursor.col, column(2); got != want {
				t.Fatalf("cursor col = %d, want %d", got, want)
			}
		})
	}
}

func TestCursorPositionIgnoresTooManyParameters(t *testing.T) {
	vt := New()
	vt.resize(4, 4)
	vt.cursor.row = 2
	vt.cursor.col = 2

	vt.update(testCSI('H', []uint32{1, 1, 1}))
	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("cursor row after invalid CUP = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col after invalid CUP = %d, want %d", got, want)
	}

	vt.update(testCSI('f', []uint32{1, 1, 1}))
	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("cursor row after invalid HVP = %d, want %d", got, want)
	}
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("cursor col after invalid HVP = %d, want %d", got, want)
	}
}

func TestCursorMovementAliases(t *testing.T) {
	vt := New()
	vt.resize(4, 4)
	vt.cursor.row = 2
	vt.cursor.col = 2

	vt.update(testCSI('k', []uint32{1}))
	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row after CSI k = %d, want %d", got, want)
	}

	vt.update(testCSI('j', []uint32{1}))
	if got, want := vt.cursor.col, column(1); got != want {
		t.Fatalf("cursor col after CSI j = %d, want %d", got, want)
	}
}

func TestCursorCharacterAbsoluteIgnoresMarginsOutsideOriginMode(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.left = 2
	vt.margin.right = 4
	vt.cursor.row = 1
	vt.cursor.col = 3

	vt.cha(1)

	if got, want := vt.cursor.col, column(0); got != want {
		t.Fatalf("cursor col = %d, want %d", got, want)
	}
	if got, want := vt.cursor.row, row(1); got != want {
		t.Fatalf("cursor row = %d, want %d", got, want)
	}
}

func TestHorizontalPositionOriginModeUsesMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 3)
	vt.margin.left = 2
	vt.margin.right = 3
	vt.mode.decom = true

	vt.hpa(1)
	if got, want := vt.cursor.col, column(2); got != want {
		t.Fatalf("HPA origin col = %d, want %d", got, want)
	}

	vt.hpr(100)
	if got, want := vt.cursor.col, column(3); got != want {
		t.Fatalf("HPR origin col = %d, want %d", got, want)
	}
}

func TestVerticalPositionOriginModeUsesMargins(t *testing.T) {
	vt := New()
	vt.resize(5, 5)
	vt.margin.top = 2
	vt.margin.bottom = 3
	vt.mode.decom = true

	vt.vpa(1)
	if got, want := vt.cursor.row, row(2); got != want {
		t.Fatalf("VPA origin row = %d, want %d", got, want)
	}

	vt.vpr(100)
	if got, want := vt.cursor.row, row(3); got != want {
		t.Fatalf("VPR origin row = %d, want %d", got, want)
	}
}
