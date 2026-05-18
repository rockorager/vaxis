package image_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	vxfwimage "git.sr.ht/~rockorager/vaxis/vxfw/image"
)

type fakeImage struct {
	resizeW int
	resizeH int
	drawn   bool
	cellW   int
	cellH   int
}

func (f *fakeImage) Draw(vaxis.Window) {
	f.drawn = true
}

func (f *fakeImage) Destroy() {
}

func (f *fakeImage) Resize(w int, h int) {
	f.resizeW = w
	f.resizeH = h
	f.cellW = w
	f.cellH = h
}

func (f *fakeImage) CellSize() (int, int) {
	return f.cellW, f.cellH
}

func TestImageDraw(t *testing.T) {
	img := &fakeImage{cellW: 40, cellH: 20}
	widget := vxfwimage.New(img)

	s, err := widget.Draw(vxfw.DrawContext{
		Min: vxfw.Size{},
		Max: vxfw.Size{Width: 10, Height: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Size != (vxfw.Size{Width: 10, Height: 5}) {
		t.Fatalf("unexpected surface size: %#v", s.Size)
	}
	if s.Render == nil {
		t.Fatal("expected image surface to have a render hook")
	}

	s.Render(vaxis.Window{})
	if !img.drawn {
		t.Fatal("expected render hook to draw image")
	}
	if img.resizeW != 10 || img.resizeH != 5 {
		t.Fatalf("expected resize to 10x5, got %dx%d", img.resizeW, img.resizeH)
	}
}

func TestImageCanGrowAfterShrink(t *testing.T) {
	img := &fakeImage{cellW: 40, cellH: 20}
	widget := vxfwimage.New(img)

	s, err := widget.Draw(vxfw.DrawContext{
		Min: vxfw.Size{},
		Max: vxfw.Size{Width: 10, Height: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Render(vaxis.Window{})

	s, err = widget.Draw(vxfw.DrawContext{
		Min: vxfw.Size{},
		Max: vxfw.Size{Width: 30, Height: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Size != (vxfw.Size{Width: 30, Height: 15}) {
		t.Fatalf("unexpected surface size after growing: %#v", s.Size)
	}
	s.Render(vaxis.Window{})
	if img.resizeW != 30 || img.resizeH != 15 {
		t.Fatalf("expected resize to 30x15, got %dx%d", img.resizeW, img.resizeH)
	}
}
