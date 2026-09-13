package vaxis

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
)

// PixelPlacement describes an exact Kitty image placement. Source coordinates
// and dimensions are pixels; destination coordinates and dimensions are cells.
type PixelPlacement struct {
	ID                                             uint32
	Column, Row, Columns, Rows                     int
	SourceX, SourceY, SourceWidth, SourceHeight, Z int
}

// PixelImage is an immutable, full-resolution straight-alpha RGBA8 image.
type PixelImage struct {
	vx         *Vaxis
	id         uint64
	width      int
	height     int
	png        []byte
	uploaded   bool
	destroyed  bool
	generation uint64
}

// NewPixelImage synchronously PNG-encodes a static straight-alpha RGBA8 image.
func (vx *Vaxis) NewPixelImage(width int, height int, rgba []byte) (*PixelImage, error) {
	if width <= 0 || height <= 0 || width > int(^uint(0)>>1)/4/height || len(rgba) != width*height*4 {
		return nil, fmt.Errorf("invalid RGBA8 image dimensions or data length")
	}
	img := &image.NRGBA{Pix: append([]byte(nil), rgba...), Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return nil, fmt.Errorf("encode pixel image: %w", err)
	}
	id := vx.nextGraphicID()
	if id == 0 || id > uint64(^uint32(0)) {
		return nil, fmt.Errorf("kitty image ID space exhausted")
	}
	return &PixelImage{vx: vx, id: id, width: width, height: height, png: encoded.Bytes()}, nil
}

// Draw queues an exact placement. Invalid protocol rectangles panic.
func (p *PixelImage) Draw(at PixelPlacement) {
	if p == nil || p.destroyed {
		panic("vaxis: draw destroyed pixel image")
	}
	if at.ID == 0 || at.Column < 0 || at.Row < 0 || at.Columns <= 0 || at.Rows <= 0 ||
		at.SourceX < 0 || at.SourceY < 0 || at.SourceWidth <= 0 || at.SourceHeight <= 0 || at.Z <= 0 ||
		at.SourceX > p.width-at.SourceWidth || at.SourceY > p.height-at.SourceHeight {
		panic("vaxis: invalid pixel image placement")
	}
	variant := fmt.Sprintf("%d/%d/%d/%d/%d/%d/%d/%d", at.ID, at.SourceX, at.SourceY,
		at.SourceWidth, at.SourceHeight, at.Columns, at.Rows, at.Z)
	write := func(w io.Writer) {
		if !p.uploaded {
			p.writeUpload(w)
			p.uploaded = true
		}
		_, _ = fmt.Fprintf(w, "\x1b_Ga=p,i=%d,p=%d,x=%d,y=%d,w=%d,h=%d,c=%d,r=%d,z=%d,C=1,q=2\x1b\\",
			p.id, at.ID, at.SourceX, at.SourceY, at.SourceWidth, at.SourceHeight, at.Columns, at.Rows, at.Z)
	}
	deletePlacement := func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "\x1b_Ga=d,d=i,i=%d,p=%d,q=2\x1b\\", p.id, at.ID)
	}
	p.vx.graphicsNext = append(p.vx.graphicsNext, &placement{
		writeTo: write, deleteFn: deletePlacement, refreshFn: func() { p.uploaded = false },
		col: at.Column, row: at.Row, id: p.id, w: at.Columns, h: at.Rows,
		variant: variant, generation: p.generation,
	})
}

func (p *PixelImage) writeUpload(w io.Writer) {
	b64 := base64.StdEncoding.EncodeToString(p.png)
	for offset := 0; offset < len(b64); offset += 4096 {
		end := offset + 4096
		if end > len(b64) {
			end = len(b64)
		}
		more := 0
		if end < len(b64) {
			more = 1
		}
		if offset == 0 {
			_, _ = fmt.Fprintf(w, "\x1b_Ga=t,f=100,i=%d,m=%d,q=2;%s\x1b\\", p.id, more, b64[offset:end])
		} else {
			_, _ = fmt.Fprintf(w, "\x1b_Gm=%d,q=2;%s\x1b\\", more, b64[offset:end])
		}
	}
}

// Invalidate retains the PNG but forces upload and placement on next Draw.
func (p *PixelImage) Invalidate() {
	if p == nil || p.destroyed {
		return
	}
	p.uploaded = false
	p.generation++
}

// Destroy removes pending placements, deletes terminal data, and releases the
// cached PNG. It is safe to call repeatedly.
func (p *PixelImage) Destroy() {
	if p == nil || p.destroyed {
		return
	}
	p.destroyed = true
	p.vx.removeImagePlacement(p.id)
	p.vx.writeControlString(fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", p.id))
	p.png = nil
	p.uploaded = false
}
