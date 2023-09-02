# Vaxis

```
It begins with them, but ends with me. Their son, Vaxis
```

## Usage

### Minimal example

```go
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
		case vaxis.Resize:
			win := vx.Window()
			vaxis.Clear(win)
			vaxis.Print(win, vaxis.Text{Content: "Hello, World!"})
			vx.Render()
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c":
				return
			}
		}
	}
}
```

## TUI Library Roundup

Notcurses is included because it's the most advanced, most efficient,
most dank TUI library

| Feature                        | Vaxis | tcell | bubbletea | notcurses |
| ------------------------------ | :---: | :---: | :-------: | :-------: |
| RGB                            |  ✅   |  ✅   |    ✅     |    ✅     |
| Hyperlinks                     |  ✅   |  ✅   |    ❌     |    ❌     |
| Bracketed Paste                |  ✅   |  ✅   |    ❌     |    ❌     |
| Kitty Keyboard                 |  ✅   |  ❌   |    ❌     |    ✅     |
| Styled Underlines              |  ✅   |  ❌   |    ❌     |    ✅     |
| Mouse Shapes (OSC 22)          |  ✅   |  ❌   |    ❌     |    ❌     |
| System Clipboard (OSC 52)      |  ✅   |  ❌   |    ❌     |    ❌     |
| System Notifications (OSC 9)   |  ✅   |  ❌   |    ❌     |    ❌     |
| System Notifications (OSC 777) |  ✅   |  ❌   |    ❌     |    ❌     |
| Synchronized Output            |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (sixel)                 |  ✅   |  ✅   |    ❌     |    ✅     |
| Images (kitty)                 |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (iterm2)                |  ❌   |  ❌   |    ❌     |    ✅     |
| Video                          |  ❌   |  ❌   |    ❌     |    ✅     |
| Dank                           |  🆗   |  ❌   |    ❌     |    ✅     |
