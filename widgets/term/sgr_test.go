package term

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestSGRDoubleUnderline(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{21}))

	if got, want := vt.cursor.UnderlineStyle, vaxis.UnderlineDouble; got != want {
		t.Fatalf("underline style = %d, want %d", got, want)
	}
}

func TestSGRRapidBlinkEnablesBlink(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{6}))

	if vt.cursor.Attribute&vaxis.AttrBlink == 0 {
		t.Fatal("rapid blink did not enable blink attribute")
	}
}

func TestSGRUnknownUnderlineStyleFallsBackToSingle(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{4, 9}))

	if got, want := vt.cursor.UnderlineStyle, vaxis.UnderlineSingle; got != want {
		t.Fatalf("underline style = %d, want %d", got, want)
	}
}

func TestSGRUnsupportedColonGroupIsIgnored(t *testing.T) {
	vt := New()
	vt.cursor.Attribute = vaxis.AttrBold
	vt.cursor.Foreground = vaxis.IndexColor(2)
	vt.cursor.Background = vaxis.IndexColor(3)

	seq := testCSI('m', []uint32{0, 4, 3})
	seq.ColonSeparators = 1 << 0
	vt.update(seq)

	if vt.cursor.Attribute&vaxis.AttrBold == 0 {
		t.Fatal("unsupported colon group reset bold attribute")
	}
	if vt.cursor.Foreground != vaxis.IndexColor(2) {
		t.Fatalf("unsupported colon group reset foreground to %v", vt.cursor.Foreground)
	}
	if vt.cursor.Background != vaxis.IndexColor(3) {
		t.Fatalf("unsupported colon group reset background to %v", vt.cursor.Background)
	}
	if vt.cursor.Attribute&vaxis.AttrItalic == 0 {
		t.Fatal("later SGR parameter was not applied")
	}
}

func TestSGRDirectColorExtraParameterContinues(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{38, 2, 1, 2, 3, 3}))

	if got, want := vt.cursor.Foreground, vaxis.RGBColor(1, 2, 3); got != want {
		t.Fatalf("foreground = %v, want %v", got, want)
	}
	if vt.cursor.Attribute&vaxis.AttrItalic == 0 {
		t.Fatal("later SGR parameter was not applied")
	}
}

func TestSGRDirectColorMissingColorIsIgnored(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
	}{
		{name: "foreground", params: []uint32{38, 5}},
		{name: "background", params: []uint32{48, 5}},
		{name: "underline", params: []uint32{58, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()
			vt.cursor.Foreground = vaxis.IndexColor(1)
			vt.cursor.Background = vaxis.IndexColor(2)
			vt.cursor.UnderlineColor = vaxis.IndexColor(3)

			vt.update(testCSI('m', tt.params))

			if got, want := vt.cursor.Foreground, vaxis.IndexColor(1); got != want {
				t.Fatalf("foreground = %v, want %v", got, want)
			}
			if got, want := vt.cursor.Background, vaxis.IndexColor(2); got != want {
				t.Fatalf("background = %v, want %v", got, want)
			}
			if got, want := vt.cursor.UnderlineColor, vaxis.IndexColor(3); got != want {
				t.Fatalf("underline color = %v, want %v", got, want)
			}
		})
	}
}

func TestSGRSemicolonDirectColorDoesNotSkipColorSpace(t *testing.T) {
	tests := []struct {
		name   string
		params []uint32
		check  func(*testing.T, *Model)
	}{
		{
			name:   "foreground",
			params: []uint32{38, 2, 0, 1, 2, 3},
			check: func(t *testing.T, vt *Model) {
				if got, want := vt.cursor.Foreground, vaxis.RGBColor(0, 1, 2); got != want {
					t.Fatalf("foreground = %v, want %v", got, want)
				}
			},
		},
		{
			name:   "background",
			params: []uint32{48, 2, 0, 1, 2, 3},
			check: func(t *testing.T, vt *Model) {
				if got, want := vt.cursor.Background, vaxis.RGBColor(0, 1, 2); got != want {
					t.Fatalf("background = %v, want %v", got, want)
				}
			},
		},
		{
			name:   "underline",
			params: []uint32{58, 2, 0, 1, 2, 3},
			check: func(t *testing.T, vt *Model) {
				if got, want := vt.cursor.UnderlineColor, vaxis.RGBColor(0, 1, 2); got != want {
					t.Fatalf("underline color = %v, want %v", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New()

			vt.update(testCSI('m', tt.params))

			tt.check(t, vt)
			if vt.cursor.Attribute&vaxis.AttrItalic == 0 {
				t.Fatal("later SGR parameter was not applied")
			}
		})
	}
}

func TestSGRColonDirectColorSkipsColorSpace(t *testing.T) {
	vt := New()

	seq := testCSI('m', []uint32{38, 2, 0, 1, 2, 3, 3})
	for i := 0; i < 5; i++ {
		seq.ColonSeparators |= 1 << uint(i)
	}
	vt.update(seq)

	if got, want := vt.cursor.Foreground, vaxis.RGBColor(1, 2, 3); got != want {
		t.Fatalf("foreground = %v, want %v", got, want)
	}
	if vt.cursor.Attribute&vaxis.AttrItalic == 0 {
		t.Fatal("later SGR parameter was not applied")
	}
}

func TestSGRColonDirectColorTooManyColonsIsIgnored(t *testing.T) {
	vt := New()

	seq := testCSI('m', []uint32{38, 2, 0, 1, 2, 3, 4, 1})
	for i := 0; i < 6; i++ {
		seq.ColonSeparators |= 1 << uint(i)
	}
	vt.update(seq)

	if vt.cursor.Foreground != 0 {
		t.Fatalf("foreground = %v, want default", vt.cursor.Foreground)
	}
	if vt.cursor.Attribute&vaxis.AttrBold == 0 {
		t.Fatal("later SGR parameter was not applied")
	}
}

func TestSGRUnderlineColorTrailingColonShortSliceIgnored(t *testing.T) {
	vt := New()
	vt.cursor.UnderlineStyle = vaxis.UnderlineDouble
	vt.cursor.UnderlineColor = vaxis.IndexColor(3)

	seq := testCSI('m', []uint32{58, 4})
	seq.ColonSeparators = 1<<0 | 1<<1
	vt.update(seq)

	if got, want := vt.cursor.UnderlineStyle, vaxis.UnderlineDouble; got != want {
		t.Fatalf("underline style = %v, want %v", got, want)
	}
	if got, want := vt.cursor.UnderlineColor, vaxis.IndexColor(3); got != want {
		t.Fatalf("underline color = %v, want %v", got, want)
	}
}

func TestSGRKakouneUnderlineForegroundAndUnderlineColor(t *testing.T) {
	vt := New()

	seq := testCSI('m', []uint32{0, 4, 3, 38, 2, 175, 175, 215, 58, 2, 0, 190, 80, 70})
	for _, i := range []int{1, 8, 9, 10, 11, 12} {
		seq.ColonSeparators |= 1 << uint(i)
	}
	vt.update(seq)

	if got, want := vt.cursor.UnderlineStyle, vaxis.UnderlineCurly; got != want {
		t.Fatalf("underline style = %d, want %d", got, want)
	}
	if got, want := vt.cursor.Foreground, vaxis.RGBColor(175, 175, 215); got != want {
		t.Fatalf("foreground = %v, want %v", got, want)
	}
	if got, want := vt.cursor.UnderlineColor, vaxis.RGBColor(190, 80, 70); got != want {
		t.Fatalf("underline color = %v, want %v", got, want)
	}
}

func TestSGRKakouneUnderlineForegroundBackgroundAndUnderlineColor(t *testing.T) {
	vt := New()

	seq := testCSI('m', []uint32{4, 3, 38, 2, 51, 51, 51, 48, 2, 170, 170, 170, 58, 2, 255, 97, 136})
	seq.ColonSeparators = 1 << 0
	vt.update(seq)

	if got, want := vt.cursor.UnderlineStyle, vaxis.UnderlineCurly; got != want {
		t.Fatalf("underline style = %d, want %d", got, want)
	}
	if got, want := vt.cursor.Foreground, vaxis.RGBColor(51, 51, 51); got != want {
		t.Fatalf("foreground = %v, want %v", got, want)
	}
	if got, want := vt.cursor.Background, vaxis.RGBColor(170, 170, 170); got != want {
		t.Fatalf("background = %v, want %v", got, want)
	}
	if got, want := vt.cursor.UnderlineColor, vaxis.RGBColor(255, 97, 136); got != want {
		t.Fatalf("underline color = %v, want %v", got, want)
	}
}

func TestParsedSGRKakouneUnderlineForegroundBackgroundAndUnderlineColor(t *testing.T) {
	vt := New()

	parseAndApply(t, vt, "\x1b[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136;0m")

	if vt.cursor.Attribute != 0 || vt.cursor.Foreground != 0 || vt.cursor.Background != 0 ||
		vt.cursor.UnderlineColor != 0 || vt.cursor.UnderlineStyle != vaxis.UnderlineOff {
		t.Fatalf("cursor style was not reset by trailing SGR 0: %+v", vt.cursor.Style)
	}
}

func TestParsedSGRUnderlineColorBeyondInlineParams(t *testing.T) {
	vt := New()

	parseAndApply(t, vt, "\x1b[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136m")

	if got, want := vt.cursor.UnderlineColor, vaxis.RGBColor(255, 97, 136); got != want {
		t.Fatalf("underline color = %v, want %v", got, want)
	}
}

func TestSGRResetUnderlineColor(t *testing.T) {
	vt := New()
	vt.cursor.UnderlineColor = vaxis.IndexColor(4)

	vt.update(testCSI('m', []uint32{59}))

	if vt.cursor.UnderlineColor != 0 {
		t.Fatalf("underline color = %v, want default", vt.cursor.UnderlineColor)
	}
}

func TestSGROverline(t *testing.T) {
	vt := New()

	vt.update(testCSI('m', []uint32{53}))
	if vt.cursor.Attribute&vaxis.AttrOverline == 0 {
		t.Fatal("overline attribute was not set")
	}

	vt.update(testCSI('m', []uint32{55}))
	if vt.cursor.Attribute&vaxis.AttrOverline != 0 {
		t.Fatal("overline attribute was not reset")
	}
}
