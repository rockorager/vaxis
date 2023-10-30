package vaxis

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"sync/atomic"

	"github.com/mattn/go-sixel"
	"golang.org/x/image/draw"
)

const (
	noGraphics = iota
	sixelGraphics
	kitty
)

// Image is a static image on the screen
type Image interface {
	// Draw draws the [Image] to the [Window]. The image will not be drawn
	// if it is larger than the window
	Draw(Window)
	// Destroy removes an image from memory. Call when done with this image
	Destroy()
	// Resizes the image to fit within the provided area. The image will not
	// be upscaled, nor will it's aspect ratio be changed
	Resize(w int, h int)
	// CellSize is the current cell size of the encoded image
	CellSize() (w int, h int)
}

// NewImage creates a new image using the highest quality renderer the terminal
// is capable of
func (vx *Vaxis) NewImage(img image.Image) (Image, error) {
	switch vx.graphicsProtocol {
	case sixelGraphics:
		return vx.NewSixel(img), nil
	case kitty:
		return vx.NewKittyGraphic(img), nil
	default:
		return nil, fmt.Errorf("no supported image protocol")
	}
}

type KittyImage struct {
	vx       *Vaxis
	img      image.Image
	id       uint64
	w        int
	h        int
	uploaded atomic.Bool
	encoding atomic.Bool
	buf      *bytes.Buffer
}

func (vx *Vaxis) NewKittyGraphic(img image.Image) *KittyImage {
	k := &KittyImage{
		vx:  vx,
		img: img,
		id:  vx.nextGraphicID(),
		buf: bytes.NewBuffer(nil),
	}
	return k
}

// Draw draws the [Image] to the [Window].
func (k *KittyImage) Draw(win Window) {
	if k.encoding.Load() {
		return
	}
	col, row := win.Origin()
	// the pid is a 32 bit number where the high 16bits are the width and
	// the low 16 are the height
	pid := uint(col)<<16 | uint(row)
	writeFunc := func(w io.Writer) {
		if !k.uploaded.Load() {
			w.Write(k.buf.Bytes())
			k.uploaded.Store(true)
			k.buf.Reset()
		}
		fmt.Fprintf(w, "\x1B_Ga=p,i=%d,p=%d\x1B\\", k.id, pid)
	}
	deleteFunc := func(w io.Writer) {
		fmt.Fprintf(w, "\x1B_Ga=d,d=i,i=%d,p=%d\x1B\\", k.id, pid)
	}
	placement := &placement{
		col:      col,
		row:      row,
		writeTo:  writeFunc,
		deleteFn: deleteFunc,
		pid:      pid,
	}
	k.vx.graphicsNext[int(pid)] = placement
}

// Destroy deletes this image from memory
func (k *KittyImage) Destroy() {
	fmt.Fprintf(k.vx.tty, "\x1B_Ga=d,d=I,i=%d\x1B\\", k.id)
}

func (k *KittyImage) CellSize() (w int, h int) {
	return k.w, k.h
}

// Resizes the image to fit within the wxh area. The image will not be
// upscaled, nor will it's aspect ratio be changed. Resizing will be done in a
// separate goroutine. A [Redraw] event will be posted when complete
func (k *KittyImage) Resize(w int, h int) {
	// Resize the image
	cellPixW := k.vx.winSize.XPixel / k.vx.winSize.Cols
	cellPixH := k.vx.winSize.YPixel / k.vx.winSize.Rows
	img := resizeImage(k.img, w, h, cellPixW, cellPixH)

	// Reupload the image
	k.w = img.Bounds().Max.X / cellPixW
	k.h = img.Bounds().Max.Y / cellPixH

	k.encoding.Store(true)
	go func() {
		defer k.encoding.Store(false)
		// Encode it to base64
		buf := bytes.NewBuffer(nil)
		wc := base64.NewEncoder(base64.StdEncoding, buf)
		err := png.Encode(wc, img)
		if err != nil {
			log.Error("couldn't encode kitty image", "error", err)
			return
		}
		wc.Close()
		b := make([]byte, 4096)
		k.uploaded.Store(false)
		for buf.Len() > 0 {
			n, err := buf.Read(b)
			if err == io.EOF {
				break
			}
			m := 1
			if buf.Len() == 0 {
				m = 0
			}
			fmt.Fprintf(k.buf, "\x1B_Gf=100,i=%d,m=%d;%s\x1B\\", k.id, m, string(b[:n]))
		}
		k.vx.PostEvent(Redraw{})
	}()
}

type Sixel struct {
	vx       *Vaxis
	img      image.Image
	buf      *bytes.Buffer
	id       uint64
	w        int
	h        int
	encoding atomic.Bool
}

// Draw draws the [Image] to the [Window]. The image will not be drawn
// if it is larger than the window
func (s *Sixel) Draw(win Window) {
	if s.buf.Len() == 0 {
		return
	}
	if s.encoding.Load() {
		return
	}
	for y := 0; y < s.h; y += 1 {
		for x := 0; x < s.w; x += 1 {
			win.SetCell(x, y, Cell{
				sixel: true,
			})
		}
	}
	writeFunc := func(w io.Writer) {
		w.Write(s.buf.Bytes())
	}
	// loop over the locked cells and unlock them
	deleteFunc := func(_ io.Writer) {
		for y := 0; y < s.h; y += 1 {
			for x := 0; x < s.w; x += 1 {
				win.SetCell(x, y, Cell{
					sixel: false,
				})
			}
		}
	}
	col, row := win.Origin()
	// the pid is a 32 bit number where the high 16bits are the width and
	// the low 16 are the height
	pid := uint(col)<<16 | uint(row)
	placement := &placement{
		col:      col,
		row:      row,
		writeTo:  writeFunc,
		deleteFn: deleteFunc,
		pid:      pid,
	}
	s.vx.graphicsNext[int(pid)] = placement
}

// Destroy removes an image from memory. Call when done with this image
func (s *Sixel) Destroy() {
	s.buf.Reset()
}

// Resizes the image to fit within the wxh area. The image will not be
// upscaled, nor will it's aspect ratio be changed. Resize will be done in a
// separate gorotuine. A Redraw event will be posted when complete
func (s *Sixel) Resize(w int, h int) {
	s.encoding.Store(true)
	go func() {
		defer s.encoding.Store(false)
		// Resize the image
		cellPixW := s.vx.winSize.XPixel / s.vx.winSize.Cols
		cellPixH := s.vx.winSize.YPixel / s.vx.winSize.Rows
		img := resizeImage(s.img, w, h, cellPixW, cellPixH)
		s.w = img.Bounds().Max.X / cellPixW
		s.h = img.Bounds().Max.Y / cellPixH
		// Re-encode the image
		s.buf.Reset()
		err := sixel.NewEncoder(s.buf).Encode(img)
		if err != nil {
			log.Error("couldn't encode sixel: %v", err)
			return
		}
		s.vx.PostEvent(Redraw{})
	}()
}

// CellSize is the current cell size of the encoded image
func (s *Sixel) CellSize() (w int, h int) {
	return s.w, s.h
}

func (vx *Vaxis) NewSixel(img image.Image) *Sixel {
	s := &Sixel{
		vx:  vx,
		img: img,
		id:  vx.nextGraphicID(),
		buf: bytes.NewBuffer(nil),
	}
	return s
}

// placement is an image placement. If two placements are identical, the
// image will not be redrawn
type placement struct {
	writeTo  func(w io.Writer)
	deleteFn func(w io.Writer)
	col      int
	row      int
	pid      uint
}

// Resizes an image to fit within the provided rectangle (as cells). If the
// image already fits, it won't be resized
func resizeImage(img image.Image, w int, h int, cellPixW int, cellPixH int) image.Image {
	wPix := img.Bounds().Max.X
	hPix := img.Bounds().Max.Y
	// Looks complicated but we're just calculating the size of the
	// image in cells, and rounding up since we will always take
	// over any cell we bleed into.
	columns := wPix / cellPixW
	lines := hPix / cellPixH
	if columns <= w && lines <= h {
		return img
	}
	// calculate scale factors
	sfX := float64(w) / float64(columns)
	sfY := float64(h) / float64(lines)
	newPixelWidth := wPix
	newPixelHeight := hPix
	switch {
	case sfX == sfY:
		// no-op
	case sfX < sfY:
		// Width is farther off, so set our new width to w and scale h
		// appropriately
		newPixelWidth = int(sfX * float64(wPix))
		newPixelHeight = int(sfX * float64(hPix))
	case sfX > sfY:
		newPixelWidth = int(sfY * float64(wPix))
		newPixelHeight = int(sfY * float64(hPix))
	}
	dst := image.NewRGBA(image.Rect(0, 0, newPixelWidth, newPixelHeight))
	draw.NearestNeighbor.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)
	return dst
}
