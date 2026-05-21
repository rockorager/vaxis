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
	page   int
	count  int
	ticks  int
	name   string
	notes  string
	done   bool
	mode   string
	chat   string
	logs   []string
	anim   *ui.AnimationController
	stop   chan struct{}
	dialog bool
}

var demoPages = []string{"home", "text", "controls", "lists", "animation"}

func (s *DemoState) InitState() {
	rt := s.Context().Runtime()
	s.anim = s.NewAnimation(ui.AnimationOptions{Duration: 1200 * time.Millisecond, Curve: ui.EaseInOut})
	s.logs = []string{
		"Ada: welcome to the echo log",
		"Linus: submit a message below",
		"Grace: the viewport follows while you are at the end",
	}
	s.mode = "compact"
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
	body := ui.Padding(
		ui.All(1),
		ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
			s.header(),
			ui.SizedBox{Height: 1},
			s.pageBody(ctx),
			ui.SizedBox{Height: 1},
			s.footer(),
		}},
	)
	if s.dialog {
		body = ui.Stack{Alignment: ui.CenterAlign, Children: []ui.Widget{
			body,
			ui.Align{Alignment: ui.CenterAlign, Child: ui.Dialog{
				Title: "Demo dialog",
				Child: ui.Text{
					Value:    "Focus stays inside this dialog. Press Escape or choose Close to dismiss it.",
					SoftWrap: true,
				},
				Width: 48,
				Actions: []ui.Widget{
					ui.Button{Label: "Notify", OnPressed: func(ctx ui.EventContext) {
						ctx.Notify("Dialog", "Action from dialog")
					}},
					ui.Button{Label: "Close", OnPressed: func(ctx ui.EventContext) {
						s.SetState(func() { s.dialog = false })
					}},
				},
				OnDismiss: func(ctx ui.EventContext) {
					s.SetState(func() { s.dialog = false })
				},
			}},
		}}
	}
	return ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.ToggleProfileOverlayIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				ctx.ToggleProfileOverlay()
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{
				"Alt+p": ui.ToggleProfileOverlayIntent{},
			},
			Child: ui.Keymap{
				Bindings: map[string]ui.VoidCallback{
					"q": func(ctx ui.EventContext) { ctx.Quit() },
					"n": func(ctx ui.EventContext) { s.nextPage() },
					"p": func(ctx ui.EventContext) { s.previousPage() },
				},
				Child: body,
			},
		},
	}
}

func (s *DemoState) header() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Vaxis UI demo", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  —  n/p to switch pages, Tab to move focus, q to quit"},
		}},
		ui.Flex{Axis: ui.Horizontal, MainAxisAlignment: ui.MainAxisCenter, CrossAxisAlignment: ui.CrossAxisCenter, Children: []ui.Widget{
			ui.SegmentedControl[int]{
				Value: s.page,
				Segments: []ui.SegmentedItem[int]{
					{Value: 0, Label: "Home"},
					{Value: 1, Label: "Text"},
					{Value: 2, Label: "Controls"},
					{Value: 3, Label: "Lists"},
					{Value: 4, Label: "Animation"},
				},
				OnChanged: func(ctx ui.EventContext, page int) {
					s.setPage(page)
				},
			},
		}},
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
		return s.listsPage()
	case 4:
		return s.animationPage()
	default:
		return s.homePage()
	}
}

func (s *DemoState) homePage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: []ui.Widget{
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
	return ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: []ui.Widget{
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
		ui.Text{Value: "Horizontal scroll"},
		ui.SizedBox{Width: 48, Height: 2, Child: ui.Scrollbar{
			Axis: ui.ScrollHorizontal,
			Child: ui.ScrollView{
				Axis:  ui.ScrollHorizontal,
				Child: ui.Text{Value: horizontalScrollDemoLine()},
			},
		}},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Scroll pane"},
		ui.SizedBox{Width: 48, Height: 6, Child: ui.Scrollbar{
			Axis: ui.ScrollHorizontal,
			Child: ui.Scrollbar{
				Child: ui.ScrollPane{Child: ui.Text{Value: scrollPaneDemoText()}},
			},
		}},
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
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Inline links can carry OSC 8 metadata and mouse callbacks: "},
			{Text: "rockorager.dev", Style: ui.Style{
				Foreground:      ui.RGB(120, 180, 255),
				UnderlineStyle:  ui.UnderlineSingle,
				Hyperlink:       "https://rockorager.dev",
				HyperlinkParams: "id=rockorager-demo-link",
			}, OnPressed: func(ctx ui.EventContext) {
				ctx.Notify("Link clicked", "rockorager.dev")
			}},
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
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
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
		ui.Stack{Alignment: ui.TopLeft, Children: []ui.Widget{
			ui.SizedBox{Width: 28, Height: 1, Child: ui.DecoratedBox(ui.Decoration{Style: ui.Style{Background: ui.RGB(36, 36, 36)}}, ui.Text{Value: "Stack base", Style: ui.Style{Attribute: ui.AttrBold}})},
			ui.Positioned{Left: 18, Top: 0, Child: ui.Text{Value: "new", Style: ui.Style{Attribute: ui.AttrBold, Foreground: ui.RGB(78, 201, 176)}}},
		}},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Notes"},
		ui.TextArea{Value: s.notes, Placeholder: "write a note", MinHeight: 2, SoftWrap: true, OnChanged: func(ctx ui.EventContext, value string) {
			s.SetState(func() { s.notes = value })
		}},
		ui.SizedBox{Height: 1},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, Children: []ui.Widget{
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
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Dialog", OnPressed: func(ctx ui.EventContext) { s.SetState(func() { s.dialog = true }) }},
		}},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, Children: []ui.Widget{
			ui.Checkbox{Checked: s.done, Label: "Use checkmark style", OnChanged: func(ctx ui.EventContext, checked bool) {
				s.SetState(func() { s.done = checked })
			}},
			ui.SizedBox{Width: 2, Height: 1},
			ui.RichText{Spans: []ui.TextSpan{
				{Text: "checkbox: "},
				{Text: checkboxStatus(s.done), Style: ui.Style{Attribute: ui.AttrBold}},
			}},
			ui.SizedBox{Width: 2, Height: 1},
			ui.Checkbox{Disabled: true, Label: "Disabled"},
		}},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, Children: []ui.Widget{
			ui.Radio[string]{Value: "compact", GroupValue: s.mode, Label: "Compact", OnChanged: func(ctx ui.EventContext, value string) {
				s.SetState(func() { s.mode = value })
			}},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Radio[string]{Value: "cozy", GroupValue: s.mode, Label: "Cozy", OnChanged: func(ctx ui.EventContext, value string) {
				s.SetState(func() { s.mode = value })
			}},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Radio[string]{Value: "spacious", GroupValue: s.mode, Label: "Spacious", OnChanged: func(ctx ui.EventContext, value string) {
				s.SetState(func() { s.mode = value })
			}},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Radio[string]{Value: "disabled", GroupValue: s.mode, Disabled: true, Label: "Disabled"},
			ui.SizedBox{Width: 2, Height: 1},
			ui.RichText{Spans: []ui.TextSpan{
				{Text: "radio: "},
				{Text: s.mode, Style: ui.Style{Attribute: ui.AttrBold}},
			}},
		}},
		ui.Align{Alignment: ui.CenterRight, Child: ui.Text{Value: "aligned right inside the page"}},
	}}
}

func (s *DemoState) listsPage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Lists", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  mixed slivers, lazy rows, and follow-output logs"},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.SizedBox{Width: 72, Height: 5, Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
			ui.ListTile{
				Leading:  ui.Text{Value: "•"},
				Title:    ui.Text{Value: "Deploy queue"},
				Subtitle: ui.Text{Value: "2 running, 4 queued"},
				Trailing: ui.Text{Value: "open"},
				Selected: true,
				OnPressed: func(ctx ui.EventContext) {
					ctx.Notify("List tile", "Deploy queue")
				},
			},
			ui.ListTile{
				Leading:  ui.Text{Value: "✓", Style: ui.Style{Foreground: ui.RGB(78, 201, 176)}},
				Title:    ui.Text{Value: "Checks passing"},
				Trailing: ui.Text{Value: "12/12"},
				OnPressed: func(ctx ui.EventContext) {
					ctx.Notify("List tile", "Checks passing")
				},
			},
			ui.ListTile{
				Leading:  ui.Text{Value: "·"},
				Title:    ui.Text{Value: "Archived target"},
				Subtitle: ui.Text{Value: "disabled row"},
				Disabled: true,
			},
		}}},
		ui.SizedBox{Height: 1},
		ui.SizedBox{Width: 72, Height: 5, Child: ui.Scrollbar{Child: ui.CustomScrollView{Slivers: []ui.Widget{
			ui.SliverToBox{Child: ui.Text{Value: "The intro sliver scrolls away before the pinned header takes over.", SoftWrap: true}},
			ui.SliverPinnedHeader{Child: ui.Text{
				Value: " #  target             status",
				Style: ui.Style{Attribute: ui.AttrBold, Background: ui.RGB(48, 48, 48)},
			}},
			ui.SliverListBuilder{
				Count:      2000,
				ItemExtent: 1,
				Overscan:   12,
				Builder: func(ctx ui.BuildContext, i int) ui.Widget {
					return listDemoRow(i + 1)
				},
			},
			ui.SliverToBox{Child: ui.RichText{Spans: []ui.TextSpan{
				{Text: "Footer sliver", Style: ui.Style{Attribute: ui.AttrBold}},
				{Text: "\nTrack clicks page the same viewport as Page Up and Page Down. Drag the thumb for proportional scrolling."},
			}, SoftWrap: true}},
		}}}},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Variable-height messages", Style: ui.Style{Attribute: ui.AttrBold}},
		ui.SizedBox{Width: 72, Height: 3, Child: ui.Scrollbar{Child: ui.CustomScrollView{FollowOutput: true, Slivers: []ui.Widget{
			ui.SliverListBuilder{
				Count:               len(s.logs),
				EstimatedItemExtent: 2,
				Overscan:            8,
				Builder: func(ctx ui.BuildContext, i int) ui.Widget {
					return chatDemoMessage(i, s.logs[i])
				},
			},
		}}}},
		ui.SizedBox{Height: 1},
		ui.TextField{
			Value:       s.chat,
			Placeholder: "echo a message",
			MinWidth:    72,
			OnChanged: func(ctx ui.EventContext, value string) {
				s.SetState(func() { s.chat = value })
			},
			OnSubmitted: func(ctx ui.EventContext, value string) {
				value = strings.TrimSpace(value)
				if value == "" {
					return
				}
				s.SetState(func() {
					s.logs = append(s.logs, "You: "+value, "Echo: "+value)
					s.chat = ""
				})
			},
		},
	}}
}

func (s *DemoState) animationPage() ui.Widget {
	value := s.anim.Value()
	status := animationStatus(s.anim.Status())
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: []ui.Widget{
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
		ui.ConstrainedBox{
			Constraints: ui.Constraints{MaxWidth: 120},
			Child:       ui.Text{Value: animationTrack(value, 120), Overflow: ui.TextOverflowClip},
		},
		ui.ConstrainedBox{
			Constraints: ui.Constraints{MaxWidth: 120},
			Child:       ui.ProgressBar{Value: value, GradientStart: ui.RGB(78, 201, 176), GradientEnd: ui.RGB(120, 180, 255)},
		},
		ui.SizedBox{Height: 1},
		ui.Flex{Axis: ui.Horizontal, CrossAxisAlignment: ui.CrossAxisCenter, Children: []ui.Widget{
			ui.Button{Label: "Replay", OnPressed: func(ctx ui.EventContext) { s.anim.Forward() }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Stop", OnPressed: func(ctx ui.EventContext) { s.anim.Stop() }},
			ui.SizedBox{Width: 1, Height: 1},
			ui.Button{Label: "Reset", OnPressed: func(ctx ui.EventContext) { s.anim.Reset() }},
		}},
	}}
}

func (s *DemoState) footer() ui.Widget {
	return ui.Flex{Axis: ui.Horizontal, MainAxisAlignment: ui.MainAxisEnd, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "page "},
			{Text: strconv.Itoa(s.page + 1), Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: " / " + strconv.Itoa(len(demoPages)) + "  " + demoPages[s.page]},
		}},
	}}
}

func (s *DemoState) setPage(page int) {
	s.SetState(func() { s.page = page })
	if page == 4 {
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

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func checkboxStatus(checked bool) string {
	if checked {
		return "checked"
	}
	return "unchecked"
}

func scrollDemoLines() ui.Widget {
	children := make([]ui.Widget, 0, 24)
	for i := 1; i <= 24; i++ {
		children = append(children, ui.RichText{Spans: []ui.TextSpan{
			{Text: "row " + strconv.Itoa(i), Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  scrollable content for the demo viewport"},
		}})
	}
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: children}
}

func horizontalScrollDemoLine() string {
	return strings.Join([]string{
		"col-001 alpha",
		"col-002 beta",
		"col-003 gamma",
		"col-004 delta",
		"col-005 epsilon",
		"col-006 zeta",
		"col-007 eta",
	}, "  |  ")
}

func scrollPaneDemoText() string {
	rows := make([]string, 0, 18)
	for row := 1; row <= 18; row++ {
		cols := make([]string, 0, 10)
		for col := 1; col <= 10; col++ {
			cols = append(cols, "r"+padLeft(strconv.Itoa(row), 2)+"c"+padLeft(strconv.Itoa(col), 2))
		}
		rows = append(rows, strings.Join(cols, "  "))
	}
	return strings.Join(rows, "\n")
}

func listDemoRow(i int) ui.Widget {
	status := "ready"
	style := ui.Style{Attribute: ui.AttrDim}
	switch {
	case i%9 == 0:
		status = "blocked"
		style = ui.Style{Foreground: ui.RGB(255, 160, 120)}
	case i%5 == 0:
		status = "running"
		style = ui.Style{Foreground: ui.RGB(78, 201, 176)}
	case i%4 == 0:
		status = "queued"
		style = ui.Style{Foreground: ui.RGB(120, 180, 255)}
	}
	return ui.RichText{Spans: []ui.TextSpan{
		{Text: padLeft(strconv.Itoa(i), 4), Style: ui.Style{Attribute: ui.AttrBold}},
		{Text: "  "},
		{Text: "deploy target " + strconv.Itoa(100+i)},
		{Text: "  "},
		{Text: status, Style: style},
	}}
}

func chatDemoMessage(i int, body string) ui.Widget {
	return ui.RichText{Spans: []ui.TextSpan{
		{Text: padLeft(strconv.Itoa(i+1), 3), Style: ui.Style{Attribute: ui.AttrDim}},
		{Text: " "},
		{Text: body},
	}, SoftWrap: true}
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
