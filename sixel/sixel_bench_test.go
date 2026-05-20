package sixel

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func BenchmarkEncodePaletted(b *testing.B) {
	img := benchmarkImage(160, 96)
	var buf bytes.Buffer

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := NewEncoder(&buf).Encode(img); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	img := benchmarkImage(160, 96)
	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(img); err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var img image.Image
		if err := NewDecoder(bytes.NewReader(data)).Decode(&img); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkImage(w int, h int) *image.Paletted {
	palette := color.Palette{
		color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff},
		color.NRGBA{R: 0xe5, G: 0x3e, B: 0x3e, A: 0xff},
		color.NRGBA{R: 0x2e, G: 0xa0, B: 0x43, A: 0xff},
		color.NRGBA{R: 0x33, G: 0x7d, B: 0xe3, A: 0xff},
		color.NRGBA{R: 0xf0, G: 0xc4, B: 0x36, A: 0xff},
	}
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetColorIndex(x, y, uint8((x/8+y/6)%len(palette)))
		}
	}
	return img
}
