package vaxis

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestPrimaryFramePixelRecoveryAndReplacement(t *testing.T) {
	for _, mode := range []PrimaryScreenMode{PrimaryInline, PrimaryMain, PrimarySplit} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			vx, out := pixelTestVaxis(t)
			vx.winSize.Rows, vx.winSize.YPixel = 6, 120
			origin := 3
			if mode == PrimaryMain {
				origin = 0
			}
			vx.primaryScreen = &primaryScreen{mode: mode, regionHeight: 1, positioned: true, origin: origin}
			if !vx.SupportsKittyGraphics() {
				t.Fatal("RenderFrame must support native images before the first paint")
			}
			img, err := vx.NewPixelImage(2, 1, make([]byte, 8))
			if err != nil {
				t.Fatal(err)
			}
			a := placementForTest()
			b := a
			b.ID++
			b.Column = 1
			img.Draw(a)
			img.Draw(b)
			render := func() string {
				t.Helper()
				out.Reset()
				if err := vx.RenderFrame(); err != nil {
					t.Fatal(err)
				}
				return out.String()
			}
			if got := render(); strings.Count(got, "a=t") != 1 || strings.Count(got, "a=p") != 2 ||
				!strings.Contains(got, fmt.Sprintf("\x1b[%d;1H\x1b_Ga=t", origin+1)) {
				t.Fatalf("initial upload/physical origin incorrect: %q", got)
			}
			img.Invalidate()
			if got := render(); strings.Count(got, "a=t") != 1 || strings.Count(got, "a=p") != 2 {
				t.Fatalf("invalidation lost retained placements: %q", got)
			}
			// Replace geometry without clearing the queue: only this placement changes.
			b.SourceX, b.SourceWidth = 1, 1
			img.Draw(b)
			if got := render(); strings.Contains(got, "a=t") || strings.Count(got, "a=p") != 1 ||
				!strings.Contains(got, "p=4,x=1,y=0,w=1,h=1") {
				t.Fatalf("replacement did not preserve the sibling placement and upload: %q", got)
			}
			// Refresh while absent, as can happen across a screen re-reservation.
			vx.Window().Clear()
			render()
			vx.refresh = true
			if mode != PrimaryMain {
				vx.primaryScreen.origin = 1
			}
			render()
			img.Draw(a)
			if got := render(); strings.Count(got, "a=t") != 1 || strings.Count(got, "a=p") != 1 ||
				!strings.Contains(got, fmt.Sprintf("\x1b[%d;1H\x1b_Ga=t", vx.primaryScreen.origin+1)) {
				t.Fatalf("returning image did not recover at the new origin: %q", got)
			}
			if got := render(); strings.Contains(got, "\x1b_G") {
				t.Fatalf("unchanged recovered image emitted graphics: %q", got)
			}
			img.Destroy()
		})
	}
}

func TestPrimaryFrameNativeGraphicsAndMouseShape(t *testing.T) {
	vx, out := pixelTestVaxis(t)
	vx.winSize.Rows = 6
	vx.primaryScreen = &primaryScreen{regionHeight: 1, positioned: true, origin: 3}
	img, err := vx.NewPixelImage(2, 1, []byte{255, 0, 0, 255, 0, 255, 0, 255})
	if err != nil {
		t.Fatal(err)
	}
	img.Draw(placementForTest())
	vx.SetMouseShape(MouseShapeClickable)
	if err := vx.RenderFrame(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[4;1H\x1b_Ga=t") {
		t.Fatalf("image was not placed at physical row 4: %q", out.String())
	}
	if !strings.Contains(out.String(), tparm(mouseShape, MouseShapeClickable)) {
		t.Fatalf("missing mouse-shape update: %q", out.String())
	}
	out.Reset()
	vx.exitPrimaryScreen()
	if !strings.Contains(out.String(), "\x1b_Ga=d,d=i,") || len(vx.graphicsLast) != 0 {
		t.Fatalf("image placement was not released: %q", out.String())
	}
}

func TestPrimaryFrameSixelEraseUsesSurfaceOrigin(t *testing.T) {
	vx, out := pixelTestVaxis(t)
	vx.primaryScreen = &primaryScreen{checked: true, positioned: true, origin: 3}
	sixel := &Sixel{vx: vx, w: 1, h: 1, buf: bytes.NewBufferString("encoded sixel")}
	sixel.Draw(vx.Window())
	vx.graphicsNext[0].deleteFn(out)
	if got := out.String(); got != "\x1b[4;1H " {
		t.Fatalf("sixel erase=%q, want physical row 4", got)
	}
	out.Reset()
	vx.primaryScreen.positioned = false
	vx.graphicsNext[0].deleteFn(out)
	if out.Len() != 0 {
		t.Fatalf("unpositioned sixel erased terminal contents: %q", out.String())
	}
}
