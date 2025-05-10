package main

import (
	"fmt"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/log"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/list"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
	"git.sr.ht/~rockorager/vaxis/vxfw/vxlayout"
)

type App struct {
	index   int
	screens []vxfw.Widget
}

func (a *App) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
		if ev.Matches(' ') {
			a.changeScreen()
			return vxfw.ConsumeAndRedraw(), nil
		}
	}
	return nil, nil
}

func (a *App) changeScreen() {
	a.index += 1
	if a.index == len(a.screens) {
		a.index = 0
	}
}

func makeScreen1() vxfw.Widget {
	filler := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "·", Width: 1,
		},
		Style: vaxis.Style{
			Background: vaxis.ColorNavy,
		},
	}
	defaultopts := vxlayout.LayoutOptions{}

	header := func(s string) vxfw.Widget {
		t := text.New(s)
		t.Style.Background = vaxis.ColorGray
		return vxlayout.Row([]vxfw.Widget{t}, defaultopts)
	}

	return vxlayout.Column([]vxfw.Widget{
		// row 1
		header("Three widgets with space distributed evenly."),
		vxlayout.Constrained(vxlayout.Row([]vxfw.Widget{
			text.New("ONE"),
			text.New("TWO"),
			text.New("THREE"),
		}, vxlayout.LayoutOptions{
			MainAxis: vxlayout.MainAxisSpaceEvenly,
		}), nil, &vxfw.Size{Height: 1}),

		// row 2
		header("Three widgets center aligned with a 2 col gap."),
		vxlayout.Constrained(vxlayout.Row([]vxfw.Widget{
			text.New("ONE"),
			text.New("TWO"),
			text.New("THREE"),
		}, vxlayout.LayoutOptions{
			MainAxis: vxlayout.MainAxisCenter,
			Gap:      2,
		}), nil, &vxfw.Size{Height: 1}),

		// row 3
		header("Three widgets right aligned with a 2 col gap and a 20 column constrained filler."),
		vxlayout.Constrained(vxlayout.Row([]vxfw.Widget{
			text.New("ONE"),
			text.New("TWO"),
			text.New("THREE"),
			vxlayout.Flex(vxlayout.Sized(vxlayout.Fill(filler), vxfw.Size{Width: 20}), 1),
		}, vxlayout.LayoutOptions{
			MainAxis: vxlayout.MainAxisEnd,
			Gap:      2,
		}), nil, &vxfw.Size{Height: 1}),

		// row 4
		header("Three widgets with a 2 col gap and filler in between."),
		vxlayout.Constrained(vxlayout.Row([]vxfw.Widget{
			text.New("ONE"),
			vxlayout.Expanded(vxlayout.Fill(filler), 1),
			text.New("TWO"),
			vxlayout.Expanded(vxlayout.Fill(filler), 1),
			text.New("THREE"),
		}, vxlayout.LayoutOptions{
			Gap: 2,
		}), nil, &vxfw.Size{Height: 1}),
	}, vxlayout.LayoutOptions{
		Gap: 1,
	})
}

func makeScreen2() vxfw.Widget {
	filler := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "·", Width: 1,
		},
		Style: vaxis.Style{
			Background: vaxis.ColorNavy,
		},
	}
	defaultopts := vxlayout.LayoutOptions{}

	header := func(s string) vxfw.Widget {
		t := text.New(s)
		t.Style.Background = vaxis.ColorGray
		return vxlayout.Row([]vxfw.Widget{t}, defaultopts)
	}

	list := list.Dynamic{
		Builder: func(i, _ uint) vxfw.Widget {
			t := text.New(strings.Repeat(fmt.Sprintf("XX %d ", i), int(i)))
			t.Style.Foreground = vaxis.IndexColor(uint8(i) % 16)
			return t
		},
		Gap: 1,
	}

	return vxlayout.Column([]vxfw.Widget{
		header("Two spacers surrounding an infinite list widget, capped at 100 rows tall."),
		vxlayout.Row([]vxfw.Widget{
			vxlayout.Flex(vxlayout.Fill(filler), 1),
			vxlayout.Sized(&list, vxfw.Size{Height: 100, Width: 30}),
			vxlayout.Flex(vxlayout.Fill(filler), 1),
		}, defaultopts),
	}, vxlayout.LayoutOptions{Gap: 1})
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	root := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, a)

	left := text.New("Ctrl+C to quit, Space to change layouts.")
	right := text.New(fmt.Sprintf("Currently viewing layout %d of %d.", a.index+1, len(a.screens)))
	infobar, err := vxlayout.Row([]vxfw.Widget{
		left, vxlayout.Space(1), right,
	}, vxlayout.LayoutOptions{}).Draw(ctx.WithMax(vxfw.Size{Width: ctx.Max.Width, Height: 1}))
	if err != nil {
		return vxfw.Surface{}, err
	}

	screen, err := a.screens[a.index].Draw(ctx.WithMax(vxfw.Size{
		Width: ctx.Max.Width, Height: ctx.Max.Height - 1,
	}))
	if err != nil {
		return vxfw.Surface{}, err
	}

	root.AddChild(0, 0, infobar)
	root.AddChild(0, 1, screen)
	return root, nil
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Error("Couldn't create a new app: %v", err)
	}

	root := &App{
		screens: []vxfw.Widget{makeScreen1(), makeScreen2()},
	}

	app.Run(root)
}
