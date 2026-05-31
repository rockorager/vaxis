package main

import (
	"fmt"
	"time"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ui"
)

func main() {
	if err := ui.Run(logDemo{}, ui.WithDynamicPrimaryScreen()); err != nil {
		panic(err)
	}
}

type logDemo struct{}

func (logDemo) CreateState() ui.State {
	return &logDemoState{}
}

type logDemoState struct {
	ui.StateBase
	count int
}

func (s *logDemoState) Build(ui.BuildContext) ui.Widget {
	progress := float64(s.count%20) / 20
	return ui.Padding(ui.Symmetric(1, 0), ui.Column(
		ui.Text{Value: "primary-screen ui demo"},
		ui.ProgressBar{Value: progress},
		ui.Row(
			ui.Button{Label: "append log", OnPressed: s.appendLog},
			ui.Text{Value: "  space: append  q/Ctrl+C: quit"},
		),
		ui.Text{Value: fmt.Sprintf("log lines appended: %d", s.count)},
	))
}

func (s *logDemoState) HandleEvent(ctx ui.EventContext, ev ui.Event) ui.EventResult {
	key, ok := ev.(vaxis.Key)
	if !ok {
		return ui.EventIgnored
	}
	switch key.String() {
	case "Ctrl+c", "q":
		ctx.Quit()
		return ui.EventHandled
	case "space":
		s.appendLog(ctx)
		return ui.EventHandled
	}
	return ui.EventIgnored
}

func (s *logDemoState) appendLog(ctx ui.EventContext) {
	s.SetState(func() { s.count++ })
	color := logColor(s.count)
	ctx.AppendTextLn([]ui.TextSpan{
		{Text: "demo ", Style: ui.Style{Foreground: color, Attribute: ui.AttrBold}},
		{Text: time.Now().Format("15:04:05") + " ", Style: ui.Style{Foreground: ui.RGB(120, 120, 120)}},
		{Text: fmt.Sprintf("line=%03d ", s.count), Style: ui.Style{Foreground: color}},
		{Text: "this styled TextSpan log is appended without widget layout, so long text remains normal terminal output and can reflow in scrollback"},
	})
}

func logColor(index int) ui.Color {
	colors := []ui.Color{
		ui.RGB(244, 114, 182),
		ui.RGB(96, 165, 250),
		ui.RGB(52, 211, 153),
		ui.RGB(251, 191, 36),
		ui.RGB(167, 139, 250),
	}
	return colors[index%len(colors)]
}
