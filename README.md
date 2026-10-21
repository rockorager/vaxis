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

### Checked primary-screen frames

`RenderFrame() error` adds cursor-position-checked rendering with the regular
cell and graphics renderer. Use it from the first frame for interactive normal-
buffer applications. It reports cursor-query and write failures instead of
assuming that the current line belongs to the application.

```go
vx, err := vaxis.New(vaxis.Options{
	PrimaryScreen: &vaxis.PrimaryScreenOptions{
		Mode: vaxis.PrimarySplit, RegionHeight: 6, ClearOnExit: true,
	},
})
// Handle err, draw into vx.Window(), then check vx.RenderFrame().
```

- `PrimaryInline` reserves a bounded region after existing output. Managed
  appended output replaces the old frame and moves the region down.
- `PrimarySplit` starts the same way, then keeps its footer at the bottom while
  output scrolls above it. A zero- or one-row output pane uses full-screen
  scrolling and repaint rather than invalid scroll margins.
- `PrimaryMain` reserves the physical terminal height in the normal buffer;
  `RegionHeight` is ignored. Nil `PrimaryScreen` keeps the alternate-screen default.

`ClearOnExit` clears owned rows on suspend/close. Its zero value preserves final
text; choose the policy explicitly. Kitty placements are released in either case.
Sixel erasure requires a valid origin; after an unrendered resize it is skipped
rather than risking shell output. Resume reserves a fresh region after intervening
output. Do not write directly to the terminal while Vaxis owns it; use the append
queue and render it before suspending or closing. Write errors can mean partial
output; Vaxis does not replay a possibly partially written block.

`Window` and `ShowCursor` use surface-local coordinates. Raw mouse events retain
terminal coordinates, including outside releases. `PrimaryScreenOrigin` returns
the physical origin, placement generation, and validity. Compare a mouse event's
`SurfaceGeneration` with that generation before translating its row; do not route
stale queued clicks into a newly positioned surface. Outside-release/capture
policy belongs to the application. Serialize rendering, resize, event dispatch,
and lifecycle operations.

The checked path requires cursor-position reports and normal-buffer saved-cursor
behavior. Terminal reflow and scrollback remain terminal-owned. Tests exercise a
responding terminal model and a real PTY; this is not a terminal-emulator
compatibility matrix. The older inline `Render` path remains available for
existing callers; after using `RenderFrame`, `Render` uses the checked path and
logs errors. Call `RenderFrame` directly when failures must reach the application.

## Support

Questions are welcome in #vaxis on libera.chat, or on the [mailing list](mailto:~rockorager/vaxis@lists.sr.ht).

Issues can be reported on the [tracker](https://todo.sr.ht/~rockorager/vaxis).
