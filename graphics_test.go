package vaxis_test

import (
	"image/png"
	"os"

	"git.sr.ht/~rockorager/vaxis"
)

func ExampleGraphic() {
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
	// Resize to whatever size we want, in cell values
	w := 20
	h := 10
	resized := vaxis.ResizeGraphic(img, w, h)
	// Create a graphic with Vaxis. Depending on the terminal, this will
	// either send the graphic to the terminal or create a sixel encoded
	// version of the image
	g, err := vaxis.NewGraphic(resized)
	if err != nil {
		panic(err)
	}
	// Create an instance of the fullscreen window
	fullscreen := vaxis.Window{}
	// Create a window the proper size
	win := vaxis.NewWindow(&fullscreen, 0, 0, w, h)
	// Draw the graphic in the window
	g.Draw(win)
}
