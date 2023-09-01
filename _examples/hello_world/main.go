package main

import (
	"git.sr.ht/~rockorager/vaxis"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			win := vx.Window()
			vaxis.Clear(win)
			vaxis.Print(win, vaxis.Text{Content: "Hello, World!"})
			truncWin := win.New(0, 1, 10, -1)
			vaxis.PrintLine(truncWin, 0, "…", vaxis.Text{Content: "This line should be truncated at 6 characters"})
			vx.Render()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
	}
}
