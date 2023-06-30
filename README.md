# Vaxis

```
Know now there is no time, space between the Well & Unknowing.
Our story starts there.
Well into our future, yet far beyond our past.
In a romance between a pair of Unheavenly Creatures.

[...]

It begins with them, but ends with me. Their son, Vaxis
```

## Usage

### Minimal example

```go
package main

import (
	"context"

	"git.sr.ht/~rockorager/vaxis"
)

type model struct{}

func (m *model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "C-c":
			vaxis.Quit()
		}
	}
}

func (m *model) Draw(win vaxis.Window) {
	vaxis.Print(win, "Hello, World!")
}

func main() {
	err := vaxis.Init(context.Background(), vaxis.Options{})
	if err != nil {
		panic(err)
	}
	m := &model{}
	if err := vaxis.Run(m); err != nil {
		panic(err)
	}
}
```

## TUI Library Roundup

Notcurses is included because it's the most advanced, most efficient,
most dank TUI library

| Feature                   | Vaxis | tcell | bubbletea | notcurses |
| ------------------------- | :---: | :---: | :-------: | :-------: |
| RGB                       |  ✅   |  ✅   |    ✅     |    ✅     |
| Hyperlinks                |  ✅   |  ✅   |    ❌     |    ❌     |
| Bracketed Paste           |  ✅   |  ✅   |    ❌     |    ❌     |
| Kitty Keyboard            |  ✅   |  ❌   |    ❌     |    ✅     |
| Styled Underlines         |  ✅   |  ❌   |    ❌     |    ✅     |
| System Clipboard (OSC 52) |  ✅   |  ❌   |    ❌     |    ❌     |
| Images (sixel)            |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (kitty)            |  ❌   |  ❌   |    ❌     |    ✅     |
| Images (iterm2)           |  ❌   |  ❌   |    ❌     |    ✅     |
| Video                     |  ❌   |  ❌   |    ❌     |    ✅     |
| Dank                      |  🆗   |  ❌   |    ❌     |    ✅     |
