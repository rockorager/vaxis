//go:build ignore

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
			win := vaxis.Window{}
			vaxis.Clear(win)
			vaxis.Print(win, "Hello, World!")
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
