package main

import "git.sr.ht/~rockorager/vaxis"

func main() {
	err := vaxis.Init(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	for {
		switch msg := vaxis.PollMsg().(type) {
		case vaxis.Resize:
			win := vaxis.Window{Width: -1, Height: -1}
			vaxis.Clear(win)
			vaxis.Print(win, vaxis.Text{Content: "Hello, World!"})
			truncWin := vaxis.NewWindow(&win, 0, 1, 10, -1)
			vaxis.PrintLine(truncWin, 0, "…", vaxis.Text{Content: "This line should be truncated at 6 characters"})
			vaxis.Render()
		case vaxis.Key:
			switch msg.String() {
			case "Ctrl+c":
				vaxis.Close()
				return
			}
		}
	}
}
