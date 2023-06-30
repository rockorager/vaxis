package main

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"
)

func main() {
	err := vaxis.Init(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	for {
		switch msg := vaxis.PollMsg().(type) {
		case vaxis.Resize:
			win := vaxis.Window{}
			vaxis.Clear(win)
			first := align.Center(win, 13, 1)
			second := align.Center(win, 18, 1)
			second.Row += 1
			vaxis.PrintSegments(first, vaxis.Segment{
				Text: "Hello, World!",
			})
			vaxis.PrintSegments(second, vaxis.Segment{
				Text:       "Press ESC to exit.",
				Attributes: vaxis.AttrDim,
			})
		case vaxis.Key:
			switch msg.String() {
			case "Escape":
				vaxis.Close()
				return
			}
		}
	}
}
