package main

import (
	"log"
	"math"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/richtext"
)

const lorem = `Lorem ipsum odor amet, consectetuer adipiscing elit. Nulla viverra ipsum id curae dui etiam massa? Sagittis non morbi ornare penatibus pharetra inceptos dolor posuere. Placerat netus nascetur tellus nec magnis magna. Convallis accumsan sollicitudin dui sem natoque; tristique nam! Condimentum tristique risus diam nisl cursus suscipit mauris. Penatibus viverra mattis nunc maximus curabitur. Aenean mi tempus vivamus amet vitae urna. Orci at senectus ullamcorper suspendisse augue proin.
`

var segments = []vaxis.Segment{
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(1)}},
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(2)}},
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(3)}},
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(4)}},
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(5)}},
	{Text: lorem, Style: vaxis.Style{Foreground: vaxis.IndexColor(6)}},
}

type App struct {
	t      *richtext.RichText
	scroll int
}

func redrawAndConsume() vxfw.BatchCmd {
	return []vxfw.Command{
		vxfw.RedrawCmd{},
		vxfw.ConsumeEventCmd{},
	}
}

func (a *App) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('c', vaxis.ModCtrl) {
			return vxfw.QuitCmd{}, nil
		}
		if ev.Matches('j') {
			a.scroll -= 1
			return redrawAndConsume(), nil
		}
		if ev.Matches('k') {
			a.scroll += 1
			return redrawAndConsume(), nil
		}
	}
	return nil, nil
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	chCtx := ctx.WitMax(vxfw.Size{Width: 24, Height: math.MaxUint16})
	s, err := a.t.Draw(chCtx)
	if err != nil {
		return vxfw.Surface{}, err
	}

	root := vxfw.NewSurface(s.Size.Width, s.Size.Height, a)
	root.AddChild(0, a.scroll, s)

	return root, nil
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	root := &App{t: richtext.New(segments)}

	app.Run(root)
}
