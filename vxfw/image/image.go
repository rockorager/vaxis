// Package image provides a vxfw widget for rendering vaxis images.
package image

import (
	"math"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// Image displays a vaxis.Image in a vxfw widget tree.
type Image struct {
	Image vaxis.Image

	preferredW  int
	preferredH  int
	lastResizeW int
	lastResizeH int
}

func New(img vaxis.Image) *Image {
	return &Image{Image: img}
}

func (i *Image) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	width, height := i.size(ctx)
	s := vxfw.NewSurface(width, height, i)
	s.Render = func(win vaxis.Window) {
		if i.Image == nil {
			return
		}
		if i.lastResizeW != int(width) || i.lastResizeH != int(height) {
			i.Image.Resize(int(width), int(height))
			i.lastResizeW = int(width)
			i.lastResizeH = int(height)
		}
		i.Image.Draw(win)
	}
	return s, nil
}

func (i *Image) size(ctx vxfw.DrawContext) (uint16, uint16) {
	var width, height uint16
	if i.Image != nil {
		w, h := i.Image.CellSize()
		if w > i.preferredW {
			i.preferredW = w
		}
		if h > i.preferredH {
			i.preferredH = h
		}
		width = clampInt(i.preferredW, ctx.Min.Width, ctx.Max.Width)
		height = clampInt(i.preferredH, ctx.Min.Height, ctx.Max.Height)
	}
	if width == 0 {
		width = fallbackSize(ctx.Min.Width, ctx.Max.Width)
	}
	if height == 0 {
		height = fallbackSize(ctx.Min.Height, ctx.Max.Height)
	}
	return width, height
}

func fallbackSize(minimum, maximum uint16) uint16 {
	if maximum != math.MaxUint16 {
		return max(minimum, maximum)
	}
	return minimum
}

func clampInt(v int, minimum, maximum uint16) uint16 {
	if v < 0 {
		v = 0
	}
	if v < int(minimum) {
		v = int(minimum)
	}
	if maximum != math.MaxUint16 && v > int(maximum) {
		v = int(maximum)
	}
	return uint16(v)
}
