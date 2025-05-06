package main

import (
	"log"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/richtext"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
	"git.sr.ht/~rockorager/vaxis/vxfw/vxlayout"
)

type App struct {
	help   vxfw.Widget
	layout *vxlayout.FlexLayout
}

func (a *App) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
		if ev.Matches('l') || ev.Matches('L') {
			a.swapLayout()
			return vxfw.ConsumeAndRedraw(), nil
		}
	}
	return nil, nil
}

func (a *App) swapLayout() {
	if a.layout.Direction == vxlayout.FlexVertical {
		a.layout.Direction = vxlayout.FlexHorizontal
	} else {
		a.layout.Direction = vxlayout.FlexVertical
	}
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	root := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, a)
	layout, err := a.layout.Draw(vxfw.DrawContext{
		Min:        ctx.Min,
		Max:        vxfw.Size{Width: ctx.Max.Width, Height: ctx.Max.Height - 1},
		Characters: ctx.Characters,
	})
	if err != nil {
		return vxfw.Surface{}, err
	}

	help, err := a.help.Draw(vxfw.DrawContext{
		Max:        vxfw.Size{Width: ctx.Max.Width, Height: 1},
		Characters: ctx.Characters,
	})

	root.AddChild(0, 0, help)
	root.AddChild(0, 1, layout)
	return root, nil
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	widgets := []vxfw.Widget{
		richtext.New([]vaxis.Segment{
			{Text: "FIRST", Style: vaxis.Style{Background: vaxis.IndexColor(1)}},
		}),
		richtext.New([]vaxis.Segment{
			{Text: "MIDDLE", Style: vaxis.Style{Background: vaxis.IndexColor(2)}},
		}),
		richtext.New([]vaxis.Segment{
			{Text: "LAST", Style: vaxis.Style{Background: vaxis.IndexColor(3)}},
		}),
	}

	root := &App{
		help: text.New("Ctrl+C to quit, L (or l) to switch layouts."),
		layout: &vxlayout.FlexLayout{
			Children: []*vxlayout.FlexItem{
				{Widget: widgets[0], Flex: 0},
				{Widget: widgets[1], Flex: 1},
				{Widget: widgets[2], Flex: 0},
			},
			Direction: vxlayout.FlexHorizontal,
		},
	}

	app.Run(root)
}
