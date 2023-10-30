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
	img, err := newImage()
	if err != nil {
		panic(err)
	}
	vImg, err := vx.NewImage(img)
	if err != nil {
		panic(err)
	}
	defer vImg.Destroy()
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			w, h := vx.Window().Size()
			vImg.Resize(w/2, h/2)
		case vaxis.Key:
			switch ev.String() {
			case "space":
				// Makes the image disappear
				vx.Window().Clear()
				vx.Render()
				continue
			case "Ctrl+c":
				return
			case "Ctrl+l":
				// Refreshes the entire screen
				win := vx.Window()
				win.Clear()
				w, h := vImg.CellSize()
				win = align.Center(win, w, h)
				vImg.Draw(win)
				vx.Refresh()
				continue
			}
		}
		win := vx.Window()
		win.Clear()
		w, h := vImg.CellSize()
		win = align.Center(win, w, h)
		vImg.Draw(win)
		vx.Render()
	}
}

func newImage() (image.Image, error) {
	f, err := os.Open("./_examples/graphics/vaxis.png")
	if err != nil {
		return nil, err
	}
	graphic, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return graphic, nil
}
