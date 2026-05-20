package sixel

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestDecodeSinglePixel(t *testing.T) {
	img := decodeString(t, "\x1bP0;0;8q#0;2;100;0;0#0@\x1b\\")

	if got := img.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("bounds = %v, want 1x1", got)
	}
	if got := color.NRGBAModel.Convert(img.At(0, 0)); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("pixel = %#v, want red", got)
	}
}

func TestDecodeRepeat(t *testing.T) {
	img := decodeString(t, "\x1bP0;0;8q#0;2;0;100;0#0!3@\x1b\\")

	if got := img.Bounds(); got.Dx() != 3 || got.Dy() != 1 {
		t.Fatalf("bounds = %v, want 3x1", got)
	}
	for x := 0; x < 3; x++ {
		if got := color.NRGBAModel.Convert(img.At(x, 0)); got != (color.NRGBA{G: 255, A: 255}) {
			t.Fatalf("pixel %d = %#v, want green", x, got)
		}
	}
}

func TestDecodeCarriageReturnOverlaysColor(t *testing.T) {
	img := decodeString(t, "\x1bP0;0;8q#1;2;100;0;0#2;2;0;0;100#1@$#2@\x1b\\")

	if got := img.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("bounds = %v, want 1x1", got)
	}
	if got := color.NRGBAModel.Convert(img.At(0, 0)); got != (color.NRGBA{B: 255, A: 255}) {
		t.Fatalf("pixel = %#v, want blue overlay", got)
	}
}

func TestDecodeNextLine(t *testing.T) {
	img := decodeString(t, "\x1bP0;0;8q#0;2;100;0;0#0@-@\x1b\\")

	if got := img.Bounds(); got.Dx() != 1 || got.Dy() != 7 {
		t.Fatalf("bounds = %v, want 1x7", got)
	}
}

func TestEncodePalettedGolden(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{
		color.NRGBA{R: 255, A: 255},
	})
	img.SetColorIndex(0, 0, 0)

	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(img); err != nil {
		t.Fatal(err)
	}

	const want = "\x1bP0;0;8q\"1;1#1;2;100;0;0#1@\x1b\\"
	if got := buf.String(); got != want {
		t.Fatalf("encoded sixel = %q, want %q", got, want)
	}
}

func TestEncodeDecodeTransparentPixel(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 2, 1), color.Palette{
		color.NRGBA{R: 255, A: 255},
		color.NRGBA{},
	})
	img.SetColorIndex(0, 0, 0)
	img.SetColorIndex(1, 0, 1)

	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(img); err != nil {
		t.Fatal(err)
	}
	decoded := decodeBytes(t, buf.Bytes())

	if got := decoded.Bounds(); got.Dx() != 1 || got.Dy() != 1 {
		t.Fatalf("bounds = %v, want 1x1", got)
	}
}

func TestEncodeRejectsTooManyColors(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{
		color.NRGBA{R: 255, A: 255},
		color.NRGBA{G: 255, A: 255},
	})

	var buf bytes.Buffer
	err := (&Encoder{w: &buf, Colors: 2}).Encode(img)
	if err == nil {
		t.Fatal("expected too many colors error")
	}
}

func decodeString(t *testing.T, s string) image.Image {
	t.Helper()
	return decodeBytes(t, []byte(s))
}

func decodeBytes(t *testing.T, data []byte) image.Image {
	t.Helper()
	var img image.Image
	if err := NewDecoder(bytes.NewReader(data)).Decode(&img); err != nil {
		t.Fatal(err)
	}
	return img
}
