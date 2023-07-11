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
)

var (
	nextID           uint64 = 0
	graphics                = make(map[uint64]Graphic)
	graphicsProtocol        = noGraphics
)

const (
	noGraphics = iota
	sixelGraphics
	kittyGraphics
)

type Graphic struct {
	id          uint64
	pixelWidth  int
	pixelHeight int
	placement   string
}

// NewGraphic loads a graphic into memory. Depending on the terminal
// capabilities, this can mean that vaxis will retain a sixel-encoded string or
// it could mean that vaxis loads the graphic into the terminals memory (kitty)
func NewGraphic(img image.Image) (*Graphic, error) {
	nextID += 1

	g := &Graphic{
		id:          nextID,
		pixelWidth:  img.Bounds().Max.X,
		pixelHeight: img.Bounds().Max.Y,
	}

	switch graphicsProtocol {
	case sixelGraphics:
		buf := bytes.NewBuffer(nil)
		err := sixel.NewEncoder(buf).Encode(img)
		if err != nil {
			return nil, err
		}
		g.placement = buf.String()
	case kittyGraphics:
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
			stdout.WriteString(fmt.Sprintf("\x1B_Gf=100,i=%d,m=%d;%s\x1B\\", g.id, m, string(b[:n])))
		}
		g.placement = fmt.Sprintf("\x1B_Ga=p,i=%d\x1B\\", g.id)
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
	columns = int(math.Ceil(float64(g.pixelWidth) * float64(winsize.Cols) / float64(winsize.XPixel)))
	lines = int(math.Ceil(float64(g.pixelHeight) * float64(winsize.Rows) / float64(winsize.YPixel)))
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
	nextGraphicPlacements[id] = placement
	return
}

// Delete removes the graphic from memory
func (g Graphic) Delete() {
	// TODO
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
	switch graphicsProtocol {
	case kittyGraphics:
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
			if row >= len(stdScreen.buf) {
				continue
			}
			for col := p.col; col < (p.col + w); col += 1 {
				if col >= len(stdScreen.buf[0]) {
					continue
				}
				stdScreen.buf[row][col].sixel = false
			}
		}
	}
	return ""
}

func (p *placement) lockRegion() {
	switch graphicsProtocol {
	case sixelGraphics:
		w, h := p.graphic.CellSize()
		for row := p.row; row < (p.row + h); row += 1 {
			if row >= len(stdScreen.buf) {
				continue
			}
			for col := p.col; col < (p.col + w); col += 1 {
				if col >= len(stdScreen.buf[0]) {
					continue
				}
				stdScreen.buf[row][col].sixel = true
			}
		}
	}
}

// draw
func (p *placement) draw() string {
	switch graphicsProtocol {
	case kittyGraphics:
		id, err := p.id()
		if err != nil {
			return p.graphic.placement
		}
		return fmt.Sprintf("\x1B_Ga=p,i=%d,p=%d\x1B\\", p.graphic.id, id)
	}
	return p.graphic.placement
}
