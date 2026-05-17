package main

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

func main() {
	theme := ui.Theme{
		Text: ui.Style{Foreground: vaxis.ColorWhite},
		Button: ui.ButtonTheme{
			Normal:  ui.Style{Foreground: vaxis.ColorSilver},
			Focused: ui.Style{Foreground: vaxis.ColorBlack, Background: vaxis.ColorAqua},
		},
	}
	if err := ui.Run(Counter{}, ui.WithTheme(theme)); err != nil {
		panic(err)
	}
}

type Counter struct{}

func (Counter) CreateState() ui.State { return &CounterState{} }

type CounterState struct {
	ui.StateBase
	count int
}

func (s *CounterState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Keymap(map[string]ui.VoidCallback{
		"q":      func(ctx ui.EventContext) { ctx.Quit() },
		"Ctrl+c": func(ctx ui.EventContext) { ctx.Quit() },
	}, ui.Center(
		ui.Column(
			ui.Text(fmt.Sprintf("count: %d", s.count)),
			ui.Row(
				ui.Button("-", func(ctx ui.EventContext) {
					s.SetState(func() { s.count-- })
				}),
				ui.Button("+", func(ctx ui.EventContext) {
					s.SetState(func() { s.count++ })
				}),
			),
		),
	))
}
