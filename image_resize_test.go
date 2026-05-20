package vaxis

import (
	"image"
	"image/color"
	"testing"
)

func TestResizeImageReturnsOriginalWhenItFits(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))

	got := resizeImage(img, 2, 2, 1, 1)
	if got != img {
		t.Fatal("resizeImage should return original image when it already fits")
	}
}

func TestResizeImageNearestNeighbor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	colors := []color.RGBA{
		{R: 0xff, A: 0xff},
		{G: 0xff, A: 0xff},
		{B: 0xff, A: 0xff},
		{R: 0xff, G: 0xff, A: 0xff},
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, colors[(y/2)*2+x/2])
		}
	}

	got := resizeImage(img, 2, 3, 1, 1)
	if bounds := got.Bounds(); bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Fatalf("bounds = %v, want 2x2", bounds)
	}

	tests := []struct {
		x    int
		y    int
		want color.RGBA
	}{
		{x: 0, y: 0, want: colors[0]},
		{x: 1, y: 0, want: colors[1]},
		{x: 0, y: 1, want: colors[2]},
		{x: 1, y: 1, want: colors[3]},
	}
	for _, test := range tests {
		got := color.RGBAModel.Convert(got.At(test.x, test.y))
		if got != test.want {
			t.Fatalf("pixel %d,%d = %#v, want %#v", test.x, test.y, got, test.want)
		}
	}
}

func TestResizeImagePreservesAspectRatio(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))

	got := resizeImage(img, 10, 10, 1, 1)
	if bounds := got.Bounds(); bounds.Dx() != 10 || bounds.Dy() != 5 {
		t.Fatalf("bounds = %v, want 10x5", bounds)
	}
}
