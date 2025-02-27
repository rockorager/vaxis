package main

import (
	"log"
	"math"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/textfield"
)

type App struct {
	input *textfield.TextField
}

func (a *App) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
	}
	return nil, nil
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	switch ev.(type) {
	case vxfw.Init:
		return vxfw.FocusWidgetCmd(a.input), nil
	}
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	chCtx := vxfw.DrawContext{
		Max:        vxfw.Size{Width: 24, Height: math.MaxUint16},
		Characters: ctx.Characters,
	}
	s, err := a.input.Draw(chCtx)
	if err != nil {
		return vxfw.Surface{}, err
	}

	root := vxfw.NewSurface(s.Size.Width, s.Size.Height, a)
	root.AddChild(0, 0, s)

	return root, nil
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	root := &App{
		input: &textfield.TextField{},
	}

	app.Run(root)
}
