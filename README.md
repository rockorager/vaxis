# Vaxis

```
It begins with them, but ends with me. Their son, Vaxis
```

Vaxis is a Terminal User Interface (TUI) library for go. Vaxis supports modern
terminal features, such as styled underlines and graphics. A widgets package is
provided with some useful widgets.

Vaxis is _blazingly_ fast at rendering. It might not be as fast or efficient as
[notcurses](https://notcurses.com/), but significant profiling has been done to
reduce all render bottlenecks while still maintaining the feature-set.

All input parsing is done using a real terminal parser, based on the excellent
state machine by [Paul Flo Williams](https://vt100.net/emu/dec_ansi_parser).
Some modifications have been made to allow for proper SGR parsing (':' separated
sub-parameters)

Vaxis **does not use terminfo**. Support for features is detected through
terminal queries. Vaxis assumes xterm-style escape codes everywhere else.

Contributions are welcome.

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
			vaxis.Print(win, vaxis.Segment{Text: "Hello, World!"})
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

## Support

Questions are welcome in #vaxis on libera.chat, or on the [mailing list](mailto:~rockorager/vaxis-devel@lists.sr.ht)

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
| Synchronized Output (DEC 2026) |  ✅   |  ❌   |    ❌     |    ✅     |
| Unicode Core (DEC 2027)        |  ✅   |  ❌   |    ❌     |    ❌     |
| Color Mode Updates (DEC 2031)  |  ✅   |  ❌   |    ❌     |    ❌     |
| Images (full/space)            |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (half block)            |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (quadrant)              |  ❌   |  ❌   |    ❌     |    ✅     |
| Images (sextant)               |  ❌   |  ❌   |    ❌     |    ✅     |
| Images (sixel)                 |  ✅   |  ✅   |    ❌     |    ✅     |
| Images (kitty)                 |  ✅   |  ❌   |    ❌     |    ✅     |
| Images (iterm2)                |  ❌   |  ❌   |    ❌     |    ✅     |
| Video                          |  ❌   |  ❌   |    ❌     |    ✅     |
| Dank                           |  🆗   |  ❌   |    ❌     |    ✅     |
