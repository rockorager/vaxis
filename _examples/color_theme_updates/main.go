package main

import (
	"git.sr.ht/~rockorager/vaxis"
)

var colorTheme = "Color mode detection not supported"

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
		case vaxis.ColorThemeUpdate:
			switch ev.Mode {
			case vaxis.DarkMode:
				colorTheme = "Dark Mode"
			case vaxis.LightMode:
				colorTheme = "Light Mode"
			}
		}
		draw(vx.Window())
		vx.Render()
	}
}

func draw(win vaxis.Window) {
	win.Clear()
	win.Print(vaxis.Segment{
		Text: "Current Color Theme: " + colorTheme,
	})
}
