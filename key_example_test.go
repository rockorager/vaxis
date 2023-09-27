package vaxis_test

import (
	"git.sr.ht/~rockorager/vaxis"
)

func ExampleKey() {
	vx, _ := vaxis.New(vaxis.Options{})
	msg := vx.PollEvent()
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "Ctrl+c":
			vx.Close()
		case "Ctrl+l":
			vx.Refresh()
		case "j":
			// Down?
		default:
			// handle the key
		}
	}
}
