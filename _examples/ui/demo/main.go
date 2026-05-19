package main

import (
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis/ui"
)

func main() {
	if err := ui.Run(Demo{}); err != nil {
		panic(err)
	}
}

type Demo struct{}

func (Demo) CreateState() ui.State {
	return &DemoState{}
}

type DemoState struct {
	ui.StateBase
	page  int
	count int
	ticks int
	name  string
	notes string
	anim  *ui.AnimationController
	stop  chan struct{}
}

var demoPages = []string{"home", "text", "controls", "animation"}

func (s *DemoState) InitState() {
	rt := s.Context().Runtime()
	s.anim = s.NewAnimation(ui.AnimationOptions{Duration: 1200 * time.Millisecond, Curve: ui.EaseInOut})
	s.stop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rt.Dispatch(func() {
					s.SetState(func() { s.ticks++ })
				})
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *DemoState) Dispose() {
	close(s.stop)
}

func (s *DemoState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Keymap{
		Bindings: map[string]ui.VoidCallback{
			"q": func(ctx ui.EventContext) { ctx.Quit() },
			"n": func(ctx ui.EventContext) { s.nextPage() },
			"p": func(ctx ui.EventContext) { s.previousPage() },
		},
		Child: ui.Padding(
			ui.All(1),
			ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, ChildrenWidget: []ui.Widget{
				s.header(),
				ui.SizedBox{Height: 1},
				s.pageBody(ctx),
				ui.SizedBox{Height: 1},
				s.footer(),
			}},
		),
	}
}

func (s *DemoState) header() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Vaxis UI demo", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  —  n/p to switch pages, Tab to move focus, q to quit"},
		}},
		ui.Flex{Axis: ui.Horizontal, MainAxisAlignment: ui.MainAxisCenter, CrossAxisAlignment: ui.CrossAxisCenter, ChildrenWidget: []ui.Widget{
			s.navButton(0, "Home"),
			ui.SizedBox{Width: 1, Height: 1},
			s.navButton(1, "Text"),
			ui.SizedBox{Width: 1, Height: 1},
			s.navButton(2, "Controls"),
			ui.SizedBox{Width: 1, Height: 1},
			s.navButton(3, "Animation"),
		}},
	}}
}

func (s *DemoState) navButton(page int, label string) ui.Widget {
	if s.page == page {
		label = "• " + label
	}
	return ui.Button{Label: label, OnPressed: func(ctx ui.EventContext) {
		s.setPage(page)
	}}
}

func (s *DemoState) pageBody(ctx ui.BuildContext) ui.Widget {
	theme := ui.MustDepend[ui.Theme](ctx)
	return ui.DecoratedBox(
		ui.Decoration{Border: ui.BorderAll(theme.Text)},
		ui.Padding(ui.All(1), s.pageContent()),
	)
}

func (s *DemoState) pageContent() ui.Widget {
	switch s.page {
	case 1:
		return s.textPage()
	case 2:
		return s.controlsPage()
	case 3:
		return s.animationPage()
	default:
		return s.homePage()
	}
}

func (s *DemoState) homePage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Home", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nThis example is intentionally larger than the counter. It uses state, focus, buttons, keymaps, rich text, wrapping text, alignment, themed borders, and runtime dispatch from a goroutine."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "The ticker below is updated through BuildContext.Runtime().Dispatch, then SetState marks this widget dirty.", SoftWrap: true},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "ticks: "},
			{Text: strconv.Itoa(s.ticks), Style: ui.Style{Attribute: ui.AttrBold}},
		}},
	}}
}

func (s *DemoState) textPage() ui.Widget {
	return ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Text layout", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nDrag to select, double-click words, triple-click lines, then press Ctrl+C to copy."},
			{Text: "\nRichText spans can mix "},
			{Text: "bold", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: ", "},
			{Text: "italic", Style: ui.Style{Attribute: ui.AttrItalic}},
			{Text: ", and normal text while sharing the same selection area."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.SizedBox{Width: 72, Height: 5, Child: ui.Scrollbar{Child: ui.ScrollView{Child: scrollDemoLines()}}},
		ui.SizedBox{Height: 1},
		ui.ConstrainedBox{Constraints: ui.Constraints{MaxWidth: 72}, Child: ui.Text{
			Value:    "This paragraph is constrained to seventy-two cells so resizing the terminal makes the surrounding layout obvious while the paragraph itself wraps inside a predictable measure.",
			SoftWrap: true,
		}},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Multiple selectable widgets: ", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "drag from this phrase "},
			{Text: "through the next line", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: " to test cross-widget copy."},
		}, SoftWrap: true},
		ui.Text{Value: "Second selectable line follows the first in copied text."},
		ui.SelectionContainer{Disabled: true, Child: ui.RichText{Spans: []ui.TextSpan{
			{Text: "This line opts out of SelectionArea.", Style: ui.Style{Attribute: ui.AttrDim}},
		}}},
		ui.ConstrainedBox{Constraints: ui.Constraints{MaxWidth: 48}, Child: ui.Text{
			Value:    "Ellipsis keeps hidden source text out of mouse selection.",
			Overflow: ui.TextOverflowEllipsis,
			MaxLines: 1,
		}},
		ui.ConstrainedBox{Constraints: ui.Constraints{MinWidth: 72, MaxWidth: 72}, Child: ui.Text{Value: "center aligned text", Align: ui.TextAlignCenter}},
	}}}
}

func (s *DemoState) controlsPage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Controls", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nButtons are focusable. Use Tab/Shift+Tab and Enter, or click them with the mouse."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Name"},
		ui.TextField{Value: s.name, Placeholder: "type here", OnChanged: func(ctx ui.EventContext, value string) {
			s.SetState(func() { s.name = value })
		}},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Notes"},
		ui.TextArea{Value: s.notes, Placeholder: "write a note", MinHeight: 3, SoftWrap: true, OnChanged: func(ctx ui.EventContext, value string) {
			s.SetState(func() { s.notes = value })
		}},
		ui.SizedBox{Height: 1},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, ChildrenWidget: []ui.Widget{
			ui.Button{Label: "-", OnPressed: func(ctx ui.EventContext) { s.SetState(func() { s.count-- }) }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.RichText{Spans: []ui.TextSpan{
				{Text: "count: "},
				{Text: strconv.Itoa(s.count), Style: ui.Style{Attribute: ui.AttrBold}},
			}},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "+", OnPressed: func(ctx ui.EventContext) { s.SetState(func() { s.count++ }) }},
			ui.SizedBox{Width: 2, Height: 1},
			ui.Button{Label: "Title", OnPressed: func(ctx ui.EventContext) { ctx.SetTitle("Vaxis UI demo") }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Copy notes", OnPressed: func(ctx ui.EventContext) { ctx.Copy(s.notes) }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Notify", OnPressed: func(ctx ui.EventContext) { ctx.Notify("Vaxis UI demo", "Notification from the controls page") }},
		}},
		ui.Align{Alignment: ui.CenterRight, Child: ui.Text{Value: "aligned right inside the page"}},
	}}
}

func (s *DemoState) animationPage() ui.Widget {
	value := s.anim.Value()
	status := animationStatus(s.anim.Status())
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Animation", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nThis page uses a StateBase-owned AnimationController. The runner ticks it before each frame and the state is rebuilt while it is active."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "value: "},
			{Text: formatFloat(value), Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "   status: "},
			{Text: status, Style: ui.Style{Attribute: ui.AttrBold}},
		}},
		ui.Text{Value: animationTrack(value, 48)},
		ui.RichText{Spans: animationBar(value, 48)},
		ui.SizedBox{Height: 1},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, ChildrenWidget: []ui.Widget{
			ui.Button{Label: "Replay", OnPressed: func(ctx ui.EventContext) { s.anim.Forward() }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Stop", OnPressed: func(ctx ui.EventContext) { s.anim.Stop() }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Reset", OnPressed: func(ctx ui.EventContext) { s.anim.Reset() }},
		}},
	}}
}

func (s *DemoState) footer() ui.Widget {
	return ui.Flex{Axis: ui.Horizontal, MainAxisAlignment: ui.MainAxisEnd, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "page "},
			{Text: strconv.Itoa(s.page + 1), Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: " / " + strconv.Itoa(len(demoPages)) + "  " + demoPages[s.page]},
		}},
	}}
}

func (s *DemoState) setPage(page int) {
	s.SetState(func() { s.page = page })
	if page == 3 {
		s.anim.Forward()
	}
}

func (s *DemoState) nextPage() {
	s.setPage((s.page + 1) % len(demoPages))
}

func (s *DemoState) previousPage() {
	s.setPage((s.page + len(demoPages) - 1) % len(demoPages))
}

func animationStatus(status ui.AnimationStatus) string {
	switch status {
	case ui.AnimationForward:
		return "running"
	case ui.AnimationCompleted:
		return "complete"
	default:
		return "idle"
	}
}

func animationTrack(value float64, width int) string {
	if width <= 1 {
		return "◆"
	}
	pos := int(value * float64(width-1))
	if pos < 0 {
		pos = 0
	}
	if pos >= width {
		pos = width - 1
	}
	return "|" + strings.Repeat(" ", pos) + "◆" + strings.Repeat(" ", width-pos-1) + "|"
}

func animationBar(value float64, width int) []ui.TextSpan {
	if width <= 0 {
		return nil
	}
	filled := int(value * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return []ui.TextSpan{
		{Text: strings.Repeat("█", filled), Style: ui.Style{Foreground: ui.RGB(78, 201, 176)}},
		{Text: strings.Repeat("░", width-filled), Style: ui.Style{Attribute: ui.AttrDim}},
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func scrollDemoLines() ui.Widget {
	children := make([]ui.Widget, 0, 24)
	for i := 1; i <= 24; i++ {
		children = append(children, ui.RichText{Spans: []ui.TextSpan{
			{Text: "row " + strconv.Itoa(i), Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  scrollable content for the demo viewport"},
		}})
	}
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: children}
}
