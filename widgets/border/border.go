package border

import "git.sr.ht/~rockorager/vaxis"

func All(win vaxis.Window, fg, bg vaxis.Color) vaxis.Window {
	w, h := win.Size()
	win.SetCell(0, 0, vaxis.Text{
		Content:    "╭",
		Foreground: fg,
		Background: bg,
	})
	win.SetCell(0, h-1, vaxis.Text{
		Content:    "╰",
		Foreground: fg,
		Background: bg,
	})
	win.SetCell(w-1, 0, vaxis.Text{
		Content:    "╮",
		Foreground: fg,
		Background: bg,
	})
	win.SetCell(w-1, h-1, vaxis.Text{
		Content:    "╯",
		Foreground: fg,
		Background: bg,
	})
	for i := 1; i < (w - 1); i += 1 {
		win.SetCell(i, 0, vaxis.Text{
			Content:    "─",
			Foreground: fg,
			Background: bg,
		})
		win.SetCell(i, h-1, vaxis.Text{
			Content:    "─",
			Foreground: fg,
			Background: bg,
		})
	}
	for i := 1; i < (h - 1); i += 1 {
		win.SetCell(0, i, vaxis.Text{
			Content:    "│",
			Foreground: fg,
			Background: bg,
		})
		win.SetCell(w-1, i, vaxis.Text{
			Content:    "│",
			Foreground: fg,
			Background: bg,
		})
	}
	return win.New(1, 1, w-2, h-2)
}

func Left(win vaxis.Window, fg, bg vaxis.Color) vaxis.Window {
	_, h := win.Size()
	for i := 0; i < h; i += 1 {
		win.SetCell(0, i, vaxis.Text{
			Content:    "│",
			Foreground: fg,
			Background: bg,
		})
	}
	return win.New(1, 0, -1, -1)
}

func Right(win vaxis.Window, fg, bg vaxis.Color) vaxis.Window {
	w, h := win.Size()
	for i := 0; i < h; i += 1 {
		win.SetCell(w-1, i, vaxis.Text{
			Content:    "│",
			Foreground: fg,
			Background: bg,
		})
	}
	return win.New(0, 0, w-1, -1)
}

func Bottom(win vaxis.Window, fg, bg vaxis.Color) vaxis.Window {
	w, h := win.Size()
	for i := 0; i < (w - 1); i += 1 {
		win.SetCell(i, h-1, vaxis.Text{
			Content:    "─",
			Foreground: fg,
			Background: bg,
		})
	}
	return win.New(0, 0, -1, h-1)
}
