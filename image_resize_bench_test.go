package vaxis

import (
	"image"
	"image/color"
	"testing"
)

func BenchmarkScaleNearestRGBA(b *testing.B) {
	src := benchmarkRGBA(640, 360)
	dst := image.NewRGBA(image.Rect(0, 0, 160, 90))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scaleNearest(dst, src)
	}
}

func BenchmarkResizeImage(b *testing.B) {
	src := benchmarkRGBA(640, 360)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = resizeImage(src, 160, 100, 1, 1)
	}
}

func benchmarkRGBA(w int, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x),
				G: uint8(y),
				B: uint8(x + y),
				A: 0xff,
			})
		}
	}
	return img
}
