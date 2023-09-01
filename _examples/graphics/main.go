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
			img.Draw(win)
			vx.Render()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
	}
}

type img struct {
	g *vaxis.Graphic
}

func (i *img) Draw(win vaxis.Window) {
	cols, rows := i.g.CellSize()
	i.g.Draw(align.Center(win, cols, rows))
}

func newImage(vx *vaxis.Vaxis) (*img, error) {
	f, err := os.Open("./_examples/graphics/vaxis.png")
	if err != nil {
		return nil, err
	}
	graphic, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	g, err := vx.NewGraphic(graphic)
	if err != nil {
		return nil, err
	}
	i := &img{g}
	return i, nil
}
