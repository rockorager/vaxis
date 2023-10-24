package main

import (
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/spinner"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	spinner := spinner.New(vx, 100*time.Millisecond)
	spinner.Start()
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			vx.Resize(ev)
		case vaxis.SyncFunc:
			ev()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			case "space":
				spinner.Toggle()
			}
		}
		win := vx.Window()
		win.Clear()
		spinner.Draw(win)
		vx.Render()
	}
}
