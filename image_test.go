package vaxis_test

import (
	"image/png"
	"os"

	"git.sr.ht/~rockorager/vaxis"
)

func ExampleImage() {
	// Open our image
	f, err := os.Open("/home/rockorager/pic.png")
	if err != nil {
		panic(err)
	}
	// Decode into an image.Image
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	// Create a graphic with Vaxis. Depending on the terminal, this will
	// either send the graphic to the terminal or create a sixel encoded
	// version of the image
	vimg, err := vx.NewImage(img)
	if err != nil {
		panic(err)
	}
	// Resize to whatever size we want, in cell values
	w := 20
	h := 10
	vimg.Resize(w, h)
	// Create a window. The window should fully contain the image
	win := vx.Window().New(0, 0, w, h)
	// Draw the graphic in the window
	vimg.Draw(win)
	vx.Render()
}
