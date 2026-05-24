package main

import (
	"go.rockorager.dev/vaxis"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	vx.SetTitle("VAXIS")
	for ev := range vx.Events() {
		vx.Notify("Vaxis", "Can you hear us with your ears?")
		switch ev := ev.(type) {
		case vaxis.Resize:
			vx.Resize(ev)
		}
	}
}
