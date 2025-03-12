package main

import (
	"fmt"
	"log"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/list"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
)

type App struct {
	list list.Dynamic
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
		if ev.Matches('l', vaxis.ModCtrl) {
			return []vxfw.Command{vxfw.DebugCmd{}, vxfw.RedrawCmd{}}, nil
		}
	}
	return nil, nil
}

func (a *App) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	switch ev.(type) {
	case vxfw.Init:
		return vxfw.FocusWidgetCmd(&a.list), nil
	}
	return nil, nil
}

func (a *App) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	s, err := a.list.Draw(ctx)
	if err != nil {
		return vxfw.Surface{}, err
	}

	root := vxfw.NewSurface(s.Size.Width, s.Size.Height, a)
	root.AddChild(0, 0, s)

	return root, nil
}

func getWidget(i uint, cursor uint) vxfw.Widget {
	var style vaxis.Style
	if i == cursor {
		style.Attribute = vaxis.AttrReverse
	}
	content := fmt.Sprintf("Row %d", i)
	for n := uint(0); n < i; n += 1 {
		content += "\n Multiline"
	}
	return &text.Text{
		Content: content,
		Style:   style,
	}
}

func main() {
	app, err := vxfw.NewApp(vaxis.Options{})
	if err != nil {
		log.Fatalf("Couldn't create a new app: %v", err)
	}

	root := &App{
		list: list.Dynamic{
			Builder:    getWidget,
			DrawCursor: true,
			Gap: 1,
		},
	}

	app.Run(root)
}
