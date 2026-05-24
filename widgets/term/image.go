package term

import (
	"image"

	"go.rockorager.dev/vaxis"
)

type Image struct {
	origin struct {
		row int
		col int
	}
	sourceRow int
	rows      int
	cols      int
	img       image.Image
	vaxii     []*vaxisImage
}

type positionedImage struct {
	img *Image
	row int
	col int
}

type vaxisImage struct {
	// A handle on the vaxis that created this image. This is in case
	// multiple vaxis instances are connected to the same term widget
	vx      *vaxis.Vaxis
	vxImage vaxis.Image
}

func (img *Image) destroy() {
	for _, cached := range img.vaxii {
		cached.vxImage.Destroy()
	}
	img.vaxii = nil
}
