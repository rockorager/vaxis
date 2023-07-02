package main

import (
	"image"
	_ "image/png"
	"os"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"
)

type img struct {
	g *vaxis.Graphic
}

func (i *img) Update(msg vaxis.Msg) {}

func (i *img) Draw(win vaxis.Window) {
	cols, rows := i.g.CellSize()
	i.g.Draw(align.Center(win, cols, rows))
}

func newImage() *img {
	f, err := os.Open("./cmd/demo/vaxis.png")
	if err != nil {
		panic(err)
	}
	graphic, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	id, err := vaxis.NewGraphic(graphic)
	if err != nil {
		panic(err)
	}
	i := &img{id}
	return i
}
