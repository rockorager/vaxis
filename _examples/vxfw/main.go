package main

import (
	"log"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
)

type App struct {
	t *text.Text
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vxfw.Init:
		return vxfw.RedrawCmd{}, nil
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
	}
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	s, err := a.t.Draw(ctx)
	if err != nil {
		return vxfw.Surface{}, err
	}

	root := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, a)
	root.AddChild(0, 0, s)

	return root, nil
}

func main() {
	app, err := vxfw.NewApp()
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	root := &App{
		t: text.New("Hello, world!"),
	}

	app.Run(root)
}
