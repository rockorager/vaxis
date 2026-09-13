package vaxis

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"math/rand"
	"strings"
	"testing"
)

func pixelTestVaxis(t *testing.T) (*Vaxis, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	vx := newWriterTestVaxis(out)
	vx.graphicsProtocol = kitty
	vx.winSize = Resize{Cols: 2, Rows: 1, XPixel: 20, YPixel: 20}
	return vx, out
}

func renderPixelTest(t *testing.T, vx *Vaxis, out *bytes.Buffer) string {
	t.Helper()
	vx.render()
	if _, err := vx.tw.Flush(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	out.Reset()
	return s
}

func placementForTest() PixelPlacement {
	return PixelPlacement{ID: 3, Column: 0, Row: 0, Columns: 1, Rows: 1, SourceWidth: 2, SourceHeight: 1, Z: 1}
}

func TestPixelImageUploadPreservesRGBAAndIsReused(t *testing.T) {
	vx, out := pixelTestVaxis(t)
	want := []byte{255, 0, 0, 17, 0, 255, 0, 203}
	img, err := vx.NewPixelImage(2, 1, want)
	if err != nil {
		t.Fatal(err)
	}
	img.Draw(placementForTest())
	first := renderPixelTest(t, vx, out)
	upload := strings.Index(first, "\x1b_Ga=t")
	start := upload + strings.Index(first[upload:], ";") + 1
	end := strings.Index(first[start:], "\x1b\\")
	data, err := base64.StdEncoding.DecodeString(first[start : start+end])
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 2 || decoded.Bounds().Dy() != 1 {
		t.Fatalf("PNG bounds = %v", decoded.Bounds())
	}
	for x, alpha := range []uint32{17, 203} {
		_, _, _, a := decoded.At(x, 0).RGBA()
		if a>>8 != alpha {
			t.Fatalf("alpha[%d] = %d, want %d", x, a>>8, alpha)
		}
	}
	vx.graphicsNext = nil
	img.Draw(placementForTest())
	if second := renderPixelTest(t, vx, out); strings.Contains(second, "a=t") || strings.Contains(second, "a=p") {
		t.Fatalf("unchanged placement emitted graphics: %q", second)
	}
}

func TestPixelImageUploadChunksReconstructOriginalPNG(t *testing.T) {
	vx, _ := pixelTestVaxis(t)
	rgba := make([]byte, 97*83*4)
	_, _ = rand.New(rand.NewSource(7)).Read(rgba)
	img, err := vx.NewPixelImage(97, 83, rgba)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	img.writeUpload(&output)
	chunks := strings.Split(strings.TrimSuffix(output.String(), "\x1b\\"), "\x1b\\")
	if len(chunks) < 3 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	var payload strings.Builder
	for i, chunk := range chunks {
		header, data, ok := strings.Cut(chunk, ";")
		if !ok || len(data) > 4096 || len(data)%4 != 0 {
			t.Fatalf("invalid chunk %d: payload length %d", i, len(data))
		}
		more := "m=1"
		if i == len(chunks)-1 {
			more = "m=0"
		}
		if !strings.Contains(header, more) {
			t.Fatalf("chunk %d header = %q, want %s", i, header, more)
		}
		payload.WriteString(data)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil || !bytes.Equal(decoded, img.png) {
		t.Fatalf("chunk reconstruction differs from original PNG: %v", err)
	}
}

func TestPixelPlacementChangesReconcileAndIDsDoNotCollide(t *testing.T) {
	vx, out := pixelTestVaxis(t)
	img, _ := vx.NewPixelImage(2, 1, make([]byte, 8))
	a := placementForTest()
	img.Draw(a)
	renderPixelTest(t, vx, out)
	for _, mutate := range []func(*PixelPlacement){
		func(p *PixelPlacement) { p.ID++ }, func(p *PixelPlacement) { p.SourceX++; p.SourceWidth-- },
		func(p *PixelPlacement) { p.Columns++ }, func(p *PixelPlacement) { p.Z++ },
	} {
		vx.graphicsNext = nil
		b := a
		mutate(&b)
		img.Draw(b)
		got := renderPixelTest(t, vx, out)
		if !strings.Contains(got, "a=d") || !strings.Contains(got, "a=p") {
			t.Fatalf("changed placement was not reconciled: %q", got)
		}
		a = b
	}
	other, _ := vx.NewPixelImage(2, 1, make([]byte, 8))
	if img.id == other.id {
		t.Fatal("generated image IDs collided")
	}
	vx.graphicsNext = nil
	a.ID, a.SourceX, a.SourceWidth = 8, 0, 2
	b := a
	b.ID = 9
	img.Draw(a)
	img.Draw(b)
	if got := renderPixelTest(t, vx, out); strings.Count(got, "a=p") != 2 {
		t.Fatalf("multiple placements output = %q", got)
	}
}

func TestPixelImageClearInvalidateRefreshAndDestroy(t *testing.T) {
	vx, out := pixelTestVaxis(t)
	img, _ := vx.NewPixelImage(2, 1, make([]byte, 8))
	img.Draw(placementForTest())
	renderPixelTest(t, vx, out)
	vx.graphicsNext = nil
	if got := renderPixelTest(t, vx, out); !strings.Contains(got, "a=d") {
		t.Fatalf("omitted placement not deleted: %q", got)
	}
	img.Invalidate()
	img.Draw(placementForTest())
	if got := renderPixelTest(t, vx, out); !strings.Contains(got, "a=t") || !strings.Contains(got, "a=p") {
		t.Fatalf("invalidate did not reupload: %q", got)
	}
	vx.graphicsNext = nil
	img.Draw(placementForTest())
	vx.refresh = true
	if got := renderPixelTest(t, vx, out); !strings.Contains(got, "a=t") {
		t.Fatalf("refresh did not reupload: %q", got)
	}
	img.Destroy()
	img.Destroy()
	if img.png != nil || len(vx.graphicsNext) != 0 {
		t.Fatal("destroy did not release bytes and pending placements")
	}
}

func TestPixelImageRejectsInvalidInputsAndUnsupportedCapability(t *testing.T) {
	vx, _ := pixelTestVaxis(t)
	if !vx.SupportsKittyGraphics() {
		t.Fatal("valid selected Kitty protocol reported unsupported")
	}
	vx.graphicsProtocol = sixelGraphics
	if vx.SupportsKittyGraphics() {
		t.Fatal("non-Kitty selected protocol reported supported")
	}
	vx.graphicsProtocol = kitty
	vx.winSize.XPixel = 0
	if vx.SupportsKittyGraphics() {
		t.Fatal("invalid pixel geometry reported supported")
	}
	for _, tc := range []struct {
		w, h int
		rgba []byte
	}{{0, 1, nil}, {1, 1, nil}, {-1, 1, nil}} {
		if _, err := vx.NewPixelImage(tc.w, tc.h, tc.rgba); err == nil {
			t.Fatalf("NewPixelImage(%d,%d,%d bytes) succeeded", tc.w, tc.h, len(tc.rgba))
		}
	}
	img, _ := vx.NewPixelImage(1, 1, make([]byte, 4))
	defer func() {
		if recover() == nil {
			t.Fatal("invalid placement did not panic")
		}
	}()
	img.Draw(PixelPlacement{})
}
