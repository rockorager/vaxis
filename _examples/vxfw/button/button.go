package main

import (
	"log"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/button"
)

type App struct {
	b *button.Button
}

func (a *App) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
		if ev.Matches('l', vaxis.ModCtrl) {
			return []vxfw.Command{vxfw.DebugCmd{}, vxfw.RedrawCmd{}}, nil
		}
	}
	return nil, nil
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	switch ev.(type) {
	case vxfw.Init:
		return vxfw.FocusWidgetCmd(a.b), nil
	}
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	chCtx := ctx.WithMax(vxfw.Size{
		Width:  ctx.Max.Width / 2,
		Height: ctx.Max.Height / 2,
	})
	s, err := a.b.Draw(chCtx)
	if err != nil {
		return vxfw.Surface{}, err
	}

	root := vxfw.NewSurface(s.Size.Width, s.Size.Height, a)
	root.AddChild(0, 0, s)

	return root, nil
}

func (a *App) onClick() (vxfw.Command, error) {
	return vxfw.QuitCmd{}, nil
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	root := &App{}
	root.b = button.New("Click me!", root.onClick)

	app.Run(root)
}
