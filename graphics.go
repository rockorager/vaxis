package vaxis

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"

	"github.com/mattn/go-sixel"
)

var (
	nextID           uint64 = 1
	graphics                = map[Graphic]image.Image{}
	graphicsProtocol        = noGraphics
)

const (
	noGraphics = iota
	sixelGraphics
	kittyGraphics
)

type Graphic uint64

// NewGraphic registers a graphic with Vaxis. Vaxis will retain a copy of
// the graphic, returning only an ID for the graphic. Applications must
// refer to this ID for any subsequent operations
func NewGraphic(g image.Image) Graphic {
	nextID += 1
	graphics[Graphic(nextID)] = g
	return Graphic(nextID)
}

// Clears all graphics from the screen
func ClearGraphics() {
	for k := range nextGraphicPlacements {
		delete(nextGraphicPlacements, k)
	}
}

// columns x lines
func (g Graphic) CellSize() (columns int, lines int, err error) {
	img, ok := graphics[g]
	if !ok {
		return 0, 0, fmt.Errorf("image not found")
	}
	xPix := img.Bounds().Max.X
	yPix := img.Bounds().Max.Y

	// Looks complicated but we're just calculating the size of the
	// image in cells, and rounding up since we will always take
	// over any cell we bleed into.
	columns = int(math.Ceil(float64(xPix) * float64(winsize.Cols) / float64(winsize.XPixel)))
	lines = int(math.Ceil(float64(yPix) * float64(winsize.Rows) / float64(winsize.YPixel)))
	return columns, lines, nil
}

func (g Graphic) PixelSize(id uint64) (x int, y int, err error) {
	img, ok := graphics[g]
	if !ok {
		return 0, 0, fmt.Errorf("image not found")
	}
	x = img.Bounds().Max.X
	y = img.Bounds().Max.Y
	return x, y, nil
}

// Draw creates a new image placement. The image will fill the
// entire window, scaling as necessary. If the underlying graphic does
// not match the dimensions of the provided window, it will be resized
// in this call. Calling draw is a fast operation: it only queues the
// image to be drawn. Any new image to be drawn will be done so in the
// Render call. If the image doesn't require redrawing (the ID and
// geometry haven't changed), it will persist between renders
func (g Graphic) Draw(win Window) error {
	col, row := win.origin()
	w, h := win.Size()
	placementID := fmt.Sprintf("%d;%d;%d;%d;%d", g, col, row, win.Width, win.Height)
	if p, ok := lastGraphicPlacements[placementID]; ok {
		nextGraphicPlacements[placementID] = p
		return nil
	}

	// TODO we don't need to re-encode if only the location has
	// changed
	gCols, gRows, err := g.CellSize()
	if err != nil {
		return err
	}

	// from above, we already know this ID exists
	img := graphics[g]
	if gCols != win.Width || gRows != win.Height {
		// TODO resize graphic
	}

	placement := &placement{
		graphic: g,
		col:     col,
		row:     row,
		width:   w,
		height:  h,
	}

	switch graphicsProtocol {
	case sixelGraphics:
		buf := bytes.NewBuffer(nil)
		err := sixel.NewEncoder(buf).Encode(img)
		if err != nil {
			return err
		}
		placement.cache = buf.String()
	case kittyGraphics:
		buf := bytes.NewBuffer(nil)
		wc := base64.NewEncoder(base64.StdEncoding, buf)
		err := png.Encode(wc, img)
		if err != nil {
			return err
		}
		wc.Close()
		b := make([]byte, 4096)
		for buf.Len() > 0 {
			n, err := buf.Read(b)
			if err == io.EOF {
				break
			}
			if n == 0 {
				break
			}
			m := 1
			if buf.Len() == 0 {
				m = 0
			}
			tty.WriteString(fmt.Sprintf("\x1B_Gf=100,i=%d,m=%d;%s\x1B\\", g, m, string(b[:n])))
		}
		placement.cache = fmt.Sprintf("\x1B_Ga=p,i=%d\x1B\\", g)
	}

	// we key the placement by it's ID, drawing location and
	// size. If all of these are the same between subsequent draws,
	// it means the graphic hasn't changed and we won't end up
	// redrawing it on the terminal.
	nextGraphicPlacements[placementID] = placement
	return nil
}

// Deletes the graphic from Vaxis's store, and any pending renders of
// the graphic
func (g Graphic) Delete() {
	delete(graphics, g)
	for k, v := range nextGraphicPlacements {
		if v.graphic == g {
			delete(nextGraphicPlacements, k)
		}
	}
}

// placement is an image placement. If two placements are identical, the
// image will not be redrawn
type placement struct {
	graphic Graphic
	col     int
	row     int
	width   int // in cells
	height  int // in cells
	cache   string
}

// Delete clears the sixel flag on cells, or if kitty protocol is
// supported it deletes via that protocol
func (p *placement) delete() string {
	switch {
	case capabilities.kittyGraphics:
		return fmt.Sprintf("\x1B_Ga=d,d=i,i=%d\x1B\\", p.graphic)
	case capabilities.sixels:
		// For sixels we just have to loop over an remove the
		// sixel lock
		for row := p.row; row < (p.row + p.height); row += 1 {
			if row >= len(stdScreen.buf) {
				continue
			}
			for col := p.col; col < (p.col + p.width); col += 1 {
				if row >= len(stdScreen.buf[0]) {
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
		_, ok := graphics[p.graphic]
		if !ok {
			return
		}
		for row := p.row; row < (p.row + p.height); row += 1 {
			if row >= len(stdScreen.buf) {
				continue
			}
			for col := p.col; col < (p.col + p.width); col += 1 {
				if row >= len(stdScreen.buf[0]) {
					continue
				}
				stdScreen.buf[row][col].sixel = true
			}
		}
	}
}

// draw
func (p *placement) draw() string {
	return p.cache
}
