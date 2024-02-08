package main

import "git.sr.ht/~rockorager/vaxis"

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
		win := vx.Window()
		win.Clear()
		win.Print(vaxis.Segment{Text: "Hello, World!"})
		vx.Render()
	}
}
