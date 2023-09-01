package main

import (
	"image"
	"os"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	img, err := newImage(vx)
	if err != nil {
		panic(err)
	}
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			win := vx.Window()
			vaxis.Clear(win)
			w, h := img.CellSize()
			img.Draw(align.Center(win, w, h))
			vx.Render()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
	}
}

func newImage(vx *vaxis.Vaxis) (*vaxis.Graphic, error) {
	f, err := os.Open("./_examples/graphics/vaxis.png")
	if err != nil {
		return nil, err
	}
	graphic, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return vx.NewGraphic(graphic)
}
