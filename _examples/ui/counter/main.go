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

func (Counter) CreateState() ui.State {
	return &CounterState{}
}

type quitIntent struct{}

func (quitIntent) IntentType() ui.IntentType {
	return "counter.quit"
}

type CounterState struct {
	ui.StateBase
	count int
}

func (s *CounterState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			quitIntent{}.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				ctx.Quit()
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{
				"q":      quitIntent{},
				"Ctrl+c": quitIntent{},
			},
			Child: ui.Center(
				ui.Padding(
					ui.All(1),
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
		},
	}
}
