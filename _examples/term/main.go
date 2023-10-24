package main

import (
	"os"
	"os/exec"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	vt := term.New()
	vt.Attach(vx.PostEvent)
	vt.Focus()
	err = vt.Start(exec.Command(os.Getenv("EDITOR")))
	if err != nil {
		panic(err)
	}
	defer vt.Close()

	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		case term.EventClosed:
			return
		case vaxis.Redraw:
			vt.Draw(vx.Window())
			vx.Render()
			continue
		}
		vt.Update(ev)
	}
}
