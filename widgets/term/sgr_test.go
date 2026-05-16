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

func TestSGRResetUnderlineColor(t *testing.T) {
	vt := New()
	vt.cursor.UnderlineColor = vaxis.IndexColor(4)

	vt.update(testCSI('m', []uint32{59}))

	if vt.cursor.UnderlineColor != 0 {
		t.Fatalf("underline color = %v, want default", vt.cursor.UnderlineColor)
	}
}
