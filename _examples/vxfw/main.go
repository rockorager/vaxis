package main

import (
	"log"
	"math"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/richtext"
)

type App struct {
	t *richtext.RichText
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
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	chCtx := vxfw.DrawContext{
		Max:        vxfw.Size{Width: 4, Height: math.MaxUint16},
		Characters: ctx.Characters,
	}
	s, err := a.t.Draw(chCtx)
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
		t: richtext.New([]vaxis.Segment{
			{Text: "Hello", Style: vaxis.Style{Foreground: vaxis.IndexColor(4)}},
			{Text: ", "},
			{Text: "World", Style: vaxis.Style{Foreground: vaxis.IndexColor(3)}},
		}),
	}

	app.Run(root)
}
