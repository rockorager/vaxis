package main

import (
	"fmt"
	"time"

	"go.rockorager.dev/vaxis"
)

func main() {
	vx, err := vaxis.New(vaxis.Options{
		DisableMouse: true,
		PrimaryScreen: &vaxis.PrimaryScreenOptions{
			RegionHeight: 3,
		},
	})
	if err != nil {
		panic(err)
	}
	defer vx.Close()

	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-ticker.C:
				vx.SyncFunc(func() {})
			case <-done:
				return
			}
		}
	}()

	count := 0
	manual := 0
	for ev := range vx.Events() {
		switch ev := ev.(type) {
		case vaxis.Resize:
			vx.Resize(ev)
		case vaxis.SyncFunc:
			ev()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c", "q":
				return
			case "space":
				manual++
				vx.AppendString(fmt.Sprintf("manual append %03d at %s\n", manual, time.Now().Format(time.Kitchen)))
			}
		}

		count++
		vx.AppendString(fmt.Sprintf("tick %03d: this line is appended above the live region\n", count))
		win := vx.Window()
		win.Clear()
		win.Print(vaxis.Segment{Text: "primary screen demo\n"})
		win.Print(vaxis.Segment{Text: fmt.Sprintf("live tick: %03d  manual appends: %03d\n", count, manual)})
		win.Print(vaxis.Segment{Text: "press space to append, q or Ctrl+C to quit"})
		vx.Render()
	}
}
