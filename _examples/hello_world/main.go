package main

import (
	"os"
	"os/exec"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
	})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			win := vx.Window()
			win.Clear()
			win.Print(vaxis.Segment{
				Text: "Hello, World!",
			},
			)
			truncWin := win.New(0, 1, 10, -1)
			truncWin.PrintTruncate(0, vaxis.Segment{
				Text: "This line should be truncated at 6 characters",
			},
			)
			vx.Refresh()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			case "space":
				vx.Suspend()
				cmd := exec.Command("ls", "-al")
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
				cmd.Run()
				time.Sleep(2 * time.Second)
				vx.Resume()
				vx.Render()
			}
		}
	}
}
