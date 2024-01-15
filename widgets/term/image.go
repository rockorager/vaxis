package term

import (
	"image"

	"git.sr.ht/~rockorager/vaxis"
)

type Image struct {
	origin struct {
		row int
		col int
	}
	img   image.Image
	vaxii []*vaxisImage
}

type vaxisImage struct {
	// A handle on the vaxis that created this image. This is in case
	// multiple vaxis instances are connected to the same term widget
	vx      *vaxis.Vaxis
	vxImage vaxis.Image
}
