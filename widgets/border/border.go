package border

import "git.sr.ht/~rockorager/vaxis"

var (
	horizontal  = vaxis.Character{Grapheme: "─", Width: 1}
	vertical    = vaxis.Character{Grapheme: "│", Width: 1}
	topLeft     = vaxis.Character{Grapheme: "╭", Width: 1}
	topRight    = vaxis.Character{Grapheme: "╮", Width: 1}
	bottomRight = vaxis.Character{Grapheme: "╯", Width: 1}
	bottomLeft  = vaxis.Character{Grapheme: "╰", Width: 1}
)

func All(win vaxis.Window, style vaxis.Style) vaxis.Window {
	w, h := win.Size()
	win.SetCell(0, 0, vaxis.Cell{
		Character: topLeft,
		Style:     style,
	})
	win.SetCell(0, h-1, vaxis.Cell{
		Character: bottomLeft,
		Style:     style,
	})
	win.SetCell(w-1, 0, vaxis.Cell{
		Character: topRight,
		Style:     style,
	})
	win.SetCell(w-1, h-1, vaxis.Cell{
		Character: bottomRight,
		Style:     style,
	})
	for i := 1; i < (w - 1); i += 1 {
		win.SetCell(i, 0, vaxis.Cell{
			Character: horizontal,
			Style:     style,
		})
		win.SetCell(i, h-1, vaxis.Cell{
			Character: horizontal,
			Style:     style,
		})
	}
	for i := 1; i < (h - 1); i += 1 {
		win.SetCell(0, i, vaxis.Cell{
			Character: vertical,
			Style:     style,
		})
		win.SetCell(w-1, i, vaxis.Cell{
			Character: vertical,
			Style:     style,
		})
	}
	return win.New(1, 1, w-2, h-2)
}

func Left(win vaxis.Window, style vaxis.Style) vaxis.Window {
	_, h := win.Size()
	for i := 0; i < h; i += 1 {
		win.SetCell(0, i, vaxis.Cell{
			Character: vertical,
			Style:     style,
		})
	}
	return win.New(1, 0, -1, -1)
}

func Right(win vaxis.Window, style vaxis.Style) vaxis.Window {
	w, h := win.Size()
	for i := 0; i < h; i += 1 {
		win.SetCell(w-1, i, vaxis.Cell{
			Character: vertical,
			Style:     style,
		})
	}
	return win.New(0, 0, w-1, -1)
}

func Bottom(win vaxis.Window, style vaxis.Style) vaxis.Window {
	w, h := win.Size()
	for i := 0; i < w; i += 1 {
		win.SetCell(i, h-1, vaxis.Cell{
			Character: horizontal,
			Style:     style,
		})
	}
	return win.New(0, 0, -1, h-1)
}

func Top(win vaxis.Window, style vaxis.Style) vaxis.Window {
	w, _ := win.Size()
	for i := 0; i < w; i += 1 {
		win.SetCell(i, 0, vaxis.Cell{
			Character: horizontal,
			Style:     style,
		})
	}
	return win.New(0, 1, -1, -1)
}
