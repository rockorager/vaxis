package main

import (
	"os"
	"os/exec"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/widgets/term"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{
		CSIuBitMask: vaxis.CSIuDisambiguate |
			vaxis.CSIuReportEvents |
			vaxis.CSIuAlternateKeys |
			vaxis.CSIuAllKeys |
			vaxis.CSIuAssociatedText,
	})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	vt := term.New(term.WithVaxis(vx), term.WithKittyKeyboard(true))
	vt.Attach(vx.PostEvent)
	vt.Focus()
	err = vt.Start(exec.Command(os.Getenv("SHELL")))
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
			vx.HideCursor()
			vt.Draw(vx.Window())
			vx.Render()
			continue
		case term.EventNotify:
			vx.Notify(ev.Title, ev.Body)
		}
		vt.Update(ev)
	}
}
