package vaxis

import (
	"image"
	"testing"
)

func TestImage(t *testing.T) {
	upperLeft := image.Point{0, 0}
	bottomRight := image.Point{100, 100}

	goImg := image.NewRGBA(image.Rectangle{upperLeft, bottomRight})
	id := NewGraphic(goImg)

	// Set our global...ick
	winsize = Resize{
		Cols:   10,
		Rows:   5,
		XPixel: 100,
		YPixel: 100,
	}

	cols, rows, _ := id.CellSize()
	if cols != 10 {
		t.Fatalf("cols expected 10, got %d", cols)
	}
	if rows != 5 {
		t.Fatalf("rows expect 5, got %d", rows)
	}

	winsize = Resize{
		Cols:   2,
		Rows:   1,
		XPixel: 1000,
		YPixel: 1000,
	}
	cols, rows, _ = id.CellSize()
	if cols != 1 {
		t.Fatalf("cols expected 1, got %d", cols)
	}
	if rows != 1 {
		t.Fatalf("rows expect 5, got %d", rows)
	}
}
