package vaxis

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"strconv"

	"github.com/mattn/go-sixel"
	"golang.org/x/image/draw"
)

const (
	noGraphics = iota
	sixelGraphics
	kitty
)

// Graphic is an image which will be displayed in the terminal
type Graphic struct {
	vx          *Vaxis
	placement   string
	pixelWidth  int
	pixelHeight int
	id          uint64
}

// NewGraphic loads a graphic into memory. Depending on the terminal
// capabilities, this can mean that vaxis will retain a sixel-encoded string or
// it could mean that vaxis loads the graphic into the terminals memory (kitty)
func (vx *Vaxis) NewGraphic(img image.Image) (*Graphic, error) {
	vx.graphicsIDNext += 1

	g := &Graphic{
		id:          vx.graphicsIDNext,
		pixelWidth:  img.Bounds().Max.X,
		pixelHeight: img.Bounds().Max.Y,
		vx:          vx,
	}

	switch vx.graphicsProtocol {
	case sixelGraphics:
		buf := bytes.NewBuffer(nil)
		err := sixel.NewEncoder(buf).Encode(img)
		if err != nil {
			return nil, err
		}
		g.placement = buf.String()
	case kitty:
		buf := bytes.NewBuffer(nil)
		wc := base64.NewEncoder(base64.StdEncoding, buf)
		err := png.Encode(wc, img)
		if err != nil {
			return nil, err
		}
		wc.Close()
		b := make([]byte, 4096)
		for buf.Len() > 0 {
			n, err := buf.Read(b)
			if err == io.EOF {
				break
			}
			m := 1
			if buf.Len() == 0 {
				m = 0
			}
			fmt.Fprintf(vx.tty, "\x1B_Gf=100,i=%d,m=%d;%s\x1B\\", g.id, m, string(b[:n]))
		}
		g.placement = fmt.Sprintf("\x1B_GC=1,a=p,i=%d\x1B\\", g.id)
	default:
		return nil, fmt.Errorf("no graphics protocol supported")
	}
	return g, nil
}

// columns x lines
func (g Graphic) CellSize() (columns int, lines int) {
	// Looks complicated but we're just calculating the size of the
	// image in cells, and rounding up since we will always take
	// over any cell we bleed into.
	columns = int(math.Ceil(float64(g.pixelWidth) * float64(g.vx.winSize.Cols) / float64(g.vx.winSize.XPixel)))
	lines = int(math.Ceil(float64(g.pixelHeight) * float64(g.vx.winSize.Rows) / float64(g.vx.winSize.YPixel)))
	return columns, lines
}

func (g Graphic) PixelSize(id uint64) (x int, y int) {
	return g.pixelWidth, g.pixelHeight
}

// Draw creates a new image placement. The image will fill the
// entire window, scaling as necessary. If the underlying graphic does
// not match the dimensions of the provided window, it will be resized
// in this call. Calling draw is a fast operation: it only queues the
// image to be drawn. Any new image to be drawn will be done so in the
// Render call. If the image doesn't require redrawing (the ID and
// geometry haven't changed), it will persist between renders
func (g Graphic) Draw(win Window) {
	col, row := win.origin()
	placement := &placement{
		graphic: &g,
		col:     col,
		row:     row,
	}
	id, err := placement.id()
	if err != nil {
		return
	}
	g.vx.graphicsNext[id] = placement
}

// Delete removes the graphic from memory
func (g *Graphic) Delete() {
	switch g.vx.graphicsProtocol {
	case sixelGraphics:
		g.placement = ""
	case kitty:
		fmt.Fprintf(g.vx.tty, "\x1B_Ga=d,d=I,i=%d\x1B\\", g.id)
	}
}

// placement is an image placement. If two placements are identical, the
// image will not be redrawn
type placement struct {
	graphic *Graphic
	col     int
	row     int
}

func (p placement) id() (int, error) {
	idStr := fmt.Sprintf("%d%d%d", p.graphic.id, p.col, p.row)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// This will probably never happen?? Not sure how it could
		return 0, err
	}
	return id, nil
}

// delete clears the sixel flag on cells, or if kitty protocol is
// supported it deletes via that protocol
func (p *placement) delete() string {
	switch p.graphic.vx.graphicsProtocol {
	case kitty:
		id, err := p.id()
		if err != nil {
			// fallback to deleting all placements for this graphic
			return fmt.Sprintf("\x1B_Ga=d,d=i,i=%d\x1B\\", p.graphic.id)
		}
		return fmt.Sprintf("\x1B_Ga=d,d=i,i=%d,p=%d\x1B\\", p.graphic.id, id)
	case sixelGraphics:
		// For sixels we just have to loop over an remove the
		// sixel lock
		w, h := p.graphic.CellSize()
		for row := p.row; row < (p.row + h); row += 1 {
			if row >= len(p.graphic.vx.screenNext.buf) {
				continue
			}
			for col := p.col; col < (p.col + w); col += 1 {
				if col >= len(p.graphic.vx.screenNext.buf[0]) {
					continue
				}
				p.graphic.vx.screenNext.buf[row][col].sixel = false
			}
		}
	}
	return ""
}

func (p *placement) lockRegion() {
	switch p.graphic.vx.graphicsProtocol {
	case sixelGraphics:
		w, h := p.graphic.CellSize()
		for row := p.row; row < (p.row + h); row += 1 {
			if row >= len(p.graphic.vx.screenNext.buf) {
				continue
			}
			for col := p.col; col < (p.col + w); col += 1 {
				if col >= len(p.graphic.vx.screenNext.buf[0]) {
					continue
				}
				p.graphic.vx.screenNext.buf[row][col].sixel = true
			}
		}
	}
}

// draw
func (p *placement) draw() string {
	switch p.graphic.vx.graphicsProtocol {
	case kitty:
		id, err := p.id()
		if err != nil {
			return p.graphic.placement
		}
		return fmt.Sprintf("\x1B_Ga=p,i=%d,p=%d\x1B\\", p.graphic.id, id)
	}
	return p.graphic.placement
}

// Resizes an image to fit within the provided rectangle (as cells). If the
// image already fits, it won't be resized
func (vx *Vaxis) ResizeGraphic(img image.Image, w int, h int) image.Image {
	pixelWidth := img.Bounds().Max.X
	pixelHeight := img.Bounds().Max.Y
	// Looks complicated but we're just calculating the size of the
	// image in cells, and rounding up since we will always take
	// over any cell we bleed into.
	columns := float64(pixelWidth) * float64(vx.winSize.Cols) / float64(vx.winSize.XPixel)
	lines := float64(pixelHeight) * float64(vx.winSize.Rows) / float64(vx.winSize.YPixel)
	if columns <= float64(w) && lines <= float64(h) {
		return img
	}
	sfX := float64(w) / columns
	sfY := float64(h) / lines
	newPixelWidth := pixelWidth
	newPixelHeight := pixelHeight
	switch {
	case sfX == sfY:
		// no-op
	case sfX < sfY:
		// Width is farther off, so set our new width to w and scale h
		// appropriately
		newPixelWidth = int(sfX * float64(pixelWidth))
		newPixelHeight = int(sfX * float64(pixelHeight))
	case sfX > sfY:
		newPixelWidth = int(sfY * float64(pixelWidth))
		newPixelHeight = int(sfY * float64(pixelHeight))
	}
	dst := image.NewRGBA(image.Rect(0, 0, newPixelWidth, newPixelHeight))
	draw.NearestNeighbor.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)
	return dst
}
