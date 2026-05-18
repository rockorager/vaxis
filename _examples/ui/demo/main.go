package main

import (
	"strconv"
	"time"

	"git.sr.ht/~rockorager/vaxis/ui"
)

func main() {
	if err := ui.Run(Demo{}); err != nil {
		panic(err)
	}
}

type Demo struct{}

func (Demo) CreateState() ui.State { return &DemoState{} }

type DemoState struct {
	ui.StateBase
	page  int
	count int
	ticks int
	stop  chan struct{}
}

var demoPages = []string{"home", "text", "controls"}

func (s *DemoState) InitState() {
	rt := s.Context().Runtime()
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

func (s *DemoState) Dispose() { close(s.stop) }

func (s *DemoState) Build(ctx ui.BuildContext) ui.Widget {
	return ui.Keymap{
		Bindings: map[string]ui.VoidCallback{
			"q":      func(ctx ui.EventContext) { ctx.Quit() },
			"Ctrl+c": func(ctx ui.EventContext) { ctx.Quit() },
			"n":      func(ctx ui.EventContext) { s.nextPage() },
			"Right":  func(ctx ui.EventContext) { s.nextPage() },
			"p":      func(ctx ui.EventContext) { s.previousPage() },
			"Left":   func(ctx ui.EventContext) { s.previousPage() },
		},
		Child: ui.Padding(ui.All(1),
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
			{Text: "  —  n/p or ←/→ to switch pages, Tab to move focus, q to quit"},
		}},
		ui.Flex{Axis: ui.Horizontal, MainAxisAlignment: ui.MainAxisCenter, CrossAxisAlignment: ui.CrossAxisCenter, ChildrenWidget: []ui.Widget{
			s.navButton(0, "Home"),
			ui.SizedBox{Width: 1, Height: 1},
			s.navButton(1, "Text"),
			ui.SizedBox{Width: 1, Height: 1},
			s.navButton(2, "Controls"),
		}},
	}}
}

func (s *DemoState) navButton(page int, label string) ui.Widget {
	if s.page == page {
		label = "• " + label
	}
	return ui.Button{Label: label, OnPressed: func(ctx ui.EventContext) {
		s.SetState(func() { s.page = page })
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
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Text layout", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nRichText spans can mix emphasis while sharing the same wrapping engine as Text. "},
			{Text: "Bold spans", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: " stay attached to their content as lines wrap across the available width."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.SizedBox{Width: 72, Height: 5, Child: ui.Text{
			Value:    "This paragraph is constrained to seventy-two cells so resizing the terminal makes the surrounding layout obvious while the paragraph itself wraps inside a predictable measure.",
			SoftWrap: true,
		}},
		ui.SizedBox{Width: 72, Height: 1, Child: ui.Text{Value: "center aligned text", Align: ui.TextAlignCenter}},
	}}
}

func (s *DemoState) controlsPage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, ChildrenWidget: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Controls", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nButtons are focusable. Use Tab/Shift+Tab and Enter, or click them with the mouse."},
		}, SoftWrap: true},
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
		}},
		ui.Align{Alignment: ui.CenterRight, Child: ui.Text{Value: "aligned right inside the page"}},
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

func (s *DemoState) nextPage() {
	s.SetState(func() { s.page = (s.page + 1) % len(demoPages) })
}

func (s *DemoState) previousPage() {
	s.SetState(func() { s.page = (s.page + len(demoPages) - 1) % len(demoPages) })
}
