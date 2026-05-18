package main

import (
	"strconv"

	"git.sr.ht/~rockorager/vaxis/ui"
)

func main() {
	if err := ui.Run(Counter{}); err != nil {
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
	return ui.Keymap{
		Bindings: map[string]ui.VoidCallback{
			"q":      func(ctx ui.EventContext) { ctx.Quit() },
			"Ctrl+c": func(ctx ui.EventContext) { ctx.Quit() },
		},
		Child: ui.Center(
			ui.Padding(ui.All(1),
				ui.Column(
					ui.RichText{Spans: []ui.TextSpan{
						{Text: "count: "},
						{Text: strconv.Itoa(s.count), Style: ui.Style{Attribute: ui.AttrBold}},
					}},
					ui.Row(
						ui.Button{Label: "-", OnPressed: func(ctx ui.EventContext) {
							s.SetState(func() { s.count-- })
						}},
						ui.Button{Label: "+", OnPressed: func(ctx ui.EventContext) {
							s.SetState(func() { s.count++ })
						}},
					),
				),
			),
		),
	}
}
