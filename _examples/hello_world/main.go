package main

import (
	"os"

	"git.sr.ht/~rockorager/vaxis"
	"golang.org/x/exp/slog"
)

func main() {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	vx, err := vaxis.New(vaxis.Options{
		Logger: slog.Default(),
	})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	events := vx.Events()
	for ev := range events {
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
			vx.Render()
		case vaxis.Key:
			slog.Warn("Key", "is", ev)

			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
	}
}
