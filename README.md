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

import "go.rockorager.dev/vaxis"

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
		}
		win := vx.Window()
		win.Clear()
		win.Print(vaxis.Segment{Text: "Hello, World!"})
		vx.Render()
	}
}
```

### Primary-screen live region

By default Vaxis enters the alternate screen. For applications that should leave
output in the shell scrollback, create Vaxis with `PrimaryScreen`. In this mode
`Window` is the live region, and `Append`, `AppendString`, or `AppendWriter`
queue output to be written before that live region on the next `Render`.

```go
vx, err := vaxis.New(vaxis.Options{
	PrimaryScreen: &vaxis.PrimaryScreenOptions{RegionHeight: 1},
})
if err != nil {
	panic(err)
}
defer vx.Close()

vx.AppendString("command output\n")
win := vx.Window()
win.Clear()
win.Print(vaxis.Segment{Text: "status: running"})
vx.Render()
```

The primary screen is owned by the terminal, so existing shell scrollback may
reflow on resize. Vaxis repaints the live region after resize, but appended
content remains normal terminal output.

The `ui` package can use the same mode. `WithDynamicPrimaryScreen` measures the
root widget and resizes the live region to the widget's preferred height each
frame:

```go
err := ui.Run(root, ui.WithDynamicPrimaryScreen())
```

`EventContext.AppendText` and `AppendTextLn` append styled inline `TextSpan`
values without doing widget layout, so the terminal can still wrap and reflow
the text naturally.
`EventContext.AppendWidget` can append a one-time rendered widget snapshot. It
is useful for tables, status records, and other fixed layouts, but wrapped text
is converted to hard line breaks at the current terminal width. Use
`AppendString`, `AppendWriter`, `AppendText`, or `AppendTextLn` for prose or
logs that should reflow naturally in terminal scrollback.

## Support

Questions are welcome in #vaxis on libera.chat, or on the [mailing list](mailto:~rockorager/vaxis@lists.sr.ht).

Issues can be reported on the [tracker](https://todo.sr.ht/~rockorager/vaxis).
