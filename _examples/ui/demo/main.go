package main

import (
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis/ui"
)

func main() {
	if err := ui.Run(Demo{}, ui.WithThemeSet(ui.DefaultThemeSet())); err != nil {
		panic(err)
	}
}

type Demo struct{}

func (Demo) CreateState() ui.State {
	return &DemoState{}
}

type demoIntent string

const (
	demoQuitIntent         demoIntent = "demo.quit"
	demoNextPageIntent     demoIntent = "demo.next-page"
	demoPreviousPageIntent demoIntent = "demo.previous-page"
	demoCommandPalette     demoIntent = "demo.command-palette"
)

func (i demoIntent) IntentType() ui.IntentType {
	return ui.IntentType(i)
}

type DemoState struct {
	ui.StateBase
	page    int
	count   int
	ticks   int
	name    string
	notes   string
	done    bool
	mode    string
	chat    string
	logs    []string
	anim    *ui.AnimationController
	stop    chan struct{}
	dialog  bool
	palette bool
}

var demoPages = []string{"home", "text", "controls", "lists", "table", "animation", "theme"}

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
	theme := ui.MustDepend[ui.Theme](ctx)
	body := ui.Padding(
		ui.All(1),
		ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
			s.header(theme),
			ui.SizedBox{Height: 1},
			s.pageBody(ctx),
			ui.SizedBox{Height: 1},
			s.footer(),
		}},
	)
	body = ui.DecoratedBox(ui.Decoration{Style: ui.Style{Foreground: theme.Foreground, Background: theme.Background}}, body)
	overlays := []ui.OverlayEntry{}
	if s.palette {
		overlays = append(overlays, ui.OverlayEntry{
			Modal: true,
			Child: ui.CommandPalette{
				Items: s.commandPaletteItems(),
				OnDismiss: func(ctx ui.EventContext) {
					s.SetState(func() { s.palette = false })
				},
			},
		})
	}
	if s.dialog {
		overlays = append(overlays, ui.OverlayEntry{
			Modal: true,
			Child: ui.Dialog{
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
			},
		})
	}
	body = ui.Overlay{Child: body, Entries: overlays}
	return ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			ui.ToggleProfileOverlayIntentType: func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				ctx.ToggleProfileOverlay()
				return ui.EventHandled
			},
			demoQuitIntent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				ctx.Quit()
				return ui.EventHandled
			},
			demoNextPageIntent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				s.nextPage()
				return ui.EventHandled
			},
			demoPreviousPageIntent.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				s.previousPage()
				return ui.EventHandled
			},
			demoCommandPalette.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				s.SetState(func() { s.palette = true })
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: s.shortcuts(),
			Child:    body,
		},
	}
}

func (s *DemoState) shortcuts() map[string]ui.Intent {
	bindings := map[string]ui.Intent{
		"Alt+p":   ui.ToggleProfileOverlayIntent{},
		"Super+k": demoCommandPalette,
		"Meta+k":  demoCommandPalette,
	}
	if !s.palette {
		bindings["q"] = demoQuitIntent
		bindings["n"] = demoNextPageIntent
		bindings["p"] = demoPreviousPageIntent
	}
	return bindings
}

func (s *DemoState) header(theme ui.Theme) ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Vaxis UI demo", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "  —  Cmd+K commands, n/p pages, Tab focus, q quit"},
		}},
		ui.Flex{
			Axis:               ui.Horizontal,
			MainAxisAlignment:  ui.MainAxisCenter,
			CrossAxisAlignment: ui.CrossAxisCenter,
			Children:           s.pageTabs(theme),
		},
	}}
}

func (s *DemoState) pageTabs(theme ui.Theme) []ui.Widget {
	labels := []string{"Home", "Text", "Controls", "Lists", "Table", "Animation", "Theme"}
	children := make([]ui.Widget, 0, len(labels)*2-1)
	for page, label := range labels {
		page := page
		if page > 0 {
			children = append(children, ui.SizedBox{Width: 1, Height: 1})
		}
		tabTheme := theme
		tab := ui.Widget(ui.Button{Label: label, OnPressed: func(ctx ui.EventContext) {
			s.setPage(page)
		}})
		if page == s.page {
			tabTheme.Surface = theme.Primary
			tabTheme.SurfaceHovered = theme.PrimaryHovered
		}
		children = append(children, ui.Provider[ui.Theme]{Value: tabTheme, Child: tab})
	}
	return children
}

func (s *DemoState) commandPaletteItems() []ui.CommandPaletteItem {
	items := make([]ui.CommandPaletteItem, 0, len(demoPages)+4)
	for page, name := range demoPages {
		page := page
		label := demoPageLabel(name)
		items = append(items, ui.CommandPaletteItem{
			Title:       "Go to " + label,
			Description: "Open the " + name + " demo page",
			Aliases:     []string{name, "page"},
			OnSelected: func(ctx ui.EventContext) {
				s.SetState(func() {
					s.page = page
					s.palette = false
				})
			},
		})
	}
	items = append(
		items,
		ui.CommandPaletteItem{
			Title:       "Open dialog",
			Description: "Show the modal dialog example",
			Aliases:     []string{"modal"},
			OnSelected: func(ctx ui.EventContext) {
				s.SetState(func() {
					s.palette = false
					s.dialog = true
				})
			},
		},
		ui.CommandPaletteItem{
			Title:       "Replay animation",
			Description: "Start the animation controller",
			Aliases:     []string{"play"},
			OnSelected: func(ctx ui.EventContext) {
				s.anim.Forward()
				s.SetState(func() { s.palette = false })
			},
		},
		ui.CommandPaletteItem{
			Title:       "Send notification",
			Description: "Show a demo notification",
			Aliases:     []string{"notify"},
			OnSelected: func(ctx ui.EventContext) {
				ctx.Notify("Vaxis UI demo", "Command palette action")
				s.SetState(func() { s.palette = false })
			},
		},
		ui.CommandPaletteItem{
			Title:       "Quit",
			Description: "Exit the demo",
			Aliases:     []string{"exit"},
			OnSelected:  func(ctx ui.EventContext) { ctx.Quit() },
		},
	)
	return items
}

func demoPageLabel(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func (s *DemoState) pageBody(ctx ui.BuildContext) ui.Widget {
	theme := ui.MustDepend[ui.Theme](ctx)
	return ui.Padding(ui.All(1), s.pageContent(theme))
}

func (s *DemoState) pageContent(theme ui.Theme) ui.Widget {
	switch s.page {
	case 1:
		return s.textPage(theme)
	case 2:
		return s.controlsPage(theme)
	case 3:
		return s.listsPage(theme)
	case 4:
		return s.tablePage(theme)
	case 5:
		return s.animationPage(theme)
	case 6:
		return s.themePage(theme)
	default:
		return s.homePage()
	}
}

func (s *DemoState) homePage() ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Home", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nThis example is intentionally larger than the counter. It uses state, focus, buttons, shortcuts, rich text, wrapping text, alignment, themed borders, and runtime dispatch from a goroutine."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "The ticker below is updated through BuildContext.Runtime().Dispatch, then SetState marks this widget dirty.", SoftWrap: true},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "ticks: "},
			{Text: strconv.Itoa(s.ticks), Style: ui.Style{Attribute: ui.AttrBold}},
		}},
	}}
}

func (s *DemoState) tablePage(theme ui.Theme) ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStretch, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Table", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nColumns can size to content, take a fixed width, or share the remaining space proportionally."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.Table{
			Columns:   []ui.TableColumn{ui.IntrinsicColumn(), ui.FlexColumn(1), ui.FixedColumn(8)},
			ColumnGap: 2,
			RowGap:    1,
			Rows: []ui.TableRow{
				{Children: []ui.Widget{tableHeader("job"), tableHeader("status"), tableHeader("id")}},
				{Children: []ui.Widget{ui.Text{Value: "build"}, ui.Text{Value: "running tests"}, ui.Text{Value: "1024"}}},
				{Children: []ui.Widget{ui.Text{Value: "lint"}, ui.Text{Value: "waiting for worker"}, ui.Text{Value: "7"}}},
				{Children: []ui.Widget{ui.Text{Value: "deploy"}, ui.Text{Value: "blocked on approval"}, ui.Text{Value: "23"}}},
			},
		},
		ui.SizedBox{Height: 1},
		ui.Text{Value: "The first column is intrinsic, the middle column flexes, and the id column stays fixed at eight cells.", SoftWrap: true},
	}}
}

func (s *DemoState) textPage(theme ui.Theme) ui.Widget {
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
				Foreground:      theme.PrimaryHovered,
				UnderlineStyle:  ui.UnderlineSingle,
				Hyperlink:       "https://rockorager.dev",
				HyperlinkParams: "id=rockorager-demo-link",
			}, OnPressed: func(ctx ui.EventContext) {
				ctx.Notify("Link clicked", "rockorager.dev")
			}},
		}, SoftWrap: true},
		ui.Text{Value: "Second selectable line follows the first in copied text."},
		ui.SelectionContainer{Disabled: true, Child: ui.RichText{Spans: []ui.TextSpan{
			{Text: "This line opts out of SelectionArea.", Style: ui.Style{Foreground: theme.DisabledForeground}},
		}}},
		ui.ConstrainedBox{Constraints: ui.Constraints{MaxWidth: 48}, Child: ui.Text{
			Value:    "Ellipsis keeps hidden source text out of mouse selection.",
			Overflow: ui.TextOverflowEllipsis,
			MaxLines: 1,
		}},
		ui.ConstrainedBox{Constraints: ui.Constraints{MinWidth: 72, MaxWidth: 72}, Child: ui.Text{Value: "center aligned text", Align: ui.TextAlignCenter}},
	}}}
}

func (s *DemoState) controlsPage(theme ui.Theme) ui.Widget {
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
			ui.SizedBox{Width: 28, Height: 1, Child: ui.DecoratedBox(ui.Decoration{Style: ui.Style{Background: theme.Surface}}, ui.Text{Value: "Stack base", Style: ui.Style{Attribute: ui.AttrBold}})},
			ui.Positioned{Left: 18, Top: 0, Child: ui.Text{Value: "new", Style: ui.Style{Attribute: ui.AttrBold, Foreground: theme.AccentText}}},
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

func (s *DemoState) listsPage(theme ui.Theme) ui.Widget {
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
				Leading:  ui.Text{Value: "✓", Style: ui.Style{Foreground: theme.SuccessText}},
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
		ui.SizedBox{Width: 72, Height: 3, Child: ui.Scrollbar{Child: ui.CustomScrollView{Slivers: []ui.Widget{
			ui.SliverToBox{Child: ui.Text{Value: "The intro sliver scrolls away before the pinned header takes over.", SoftWrap: true}},
			ui.SliverPinnedHeader{Child: ui.Text{
				Value: " #  target             status",
				Style: ui.Style{Attribute: ui.AttrBold, Foreground: theme.Foreground, Background: theme.Surface},
			}},
			ui.SliverListBuilder{
				Count:      2000,
				ItemExtent: 1,
				Overscan:   12,
				Builder: func(ctx ui.BuildContext, i int) ui.Widget {
					return listDemoRow(theme, i+1)
				},
			},
			ui.SliverToBox{Child: ui.RichText{Spans: []ui.TextSpan{
				{Text: "Footer sliver", Style: ui.Style{Attribute: ui.AttrBold}},
				{Text: "\nTrack clicks page the same viewport as Page Up and Page Down. Drag the thumb for proportional scrolling."},
			}, SoftWrap: true}},
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
		ui.SizedBox{Height: 1},
		ui.Text{Value: "Variable-height messages", Style: ui.Style{Attribute: ui.AttrBold}},
		ui.SizedBox{Width: 72, Height: 2, Child: ui.Scrollbar{Child: ui.CustomScrollView{FollowOutput: true, Slivers: []ui.Widget{
			ui.SliverListBuilder{
				Count:               len(s.logs),
				EstimatedItemExtent: 2,
				Overscan:            8,
				Builder: func(ctx ui.BuildContext, i int) ui.Widget {
					return chatDemoMessage(theme, i, s.logs[i])
				},
			},
		}}}},
	}}
}

func (s *DemoState) animationPage(theme ui.Theme) ui.Widget {
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
			Child:       ui.ProgressBar{Value: value, GradientStart: theme.Accent, GradientEnd: theme.PrimaryHovered},
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

func (s *DemoState) themePage(theme ui.Theme) ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: []ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Theme", Style: ui.Style{Attribute: ui.AttrBold}},
			{Text: "\nSemantic colors are generated from the palette, then widgets derive their styles from these roles."},
		}, SoftWrap: true},
		ui.SizedBox{Height: 1},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Semantic colors", Style: ui.Style{Attribute: ui.AttrBold}},
		}},
		themeGroup("base", []ui.Widget{
			themeSwatchRow(
				themeSwatchSpan("Theme.Background", theme.Background, theme.Foreground),
				themeTextSpan("Theme.Foreground", theme.Foreground),
				themeSwatchSpan("Theme.Border", theme.Border, theme.Background),
			),
		}),
		themeGroup("surface", []ui.Widget{
			themeSwatchRow(
				themeSwatchSpan("Theme.Surface", theme.Surface, theme.Foreground),
				themeSwatchSpan("Theme.SurfaceHovered", theme.SurfaceHovered, theme.Foreground),
				themeSwatchSpan("Theme.SurfacePressed", theme.SurfacePressed, theme.Foreground),
			),
		}),
		themeGroup("primary and accent", []ui.Widget{
			themeSwatchRow(
				themeSwatchSpan("Theme.Primary", theme.Primary, theme.Foreground),
				themeTextSpan("Theme.PrimaryText", theme.PrimaryText),
			),
			themeSwatchRow(
				themeSwatchSpan("Theme.PrimaryHovered", theme.PrimaryHovered, theme.Foreground),
				themeSwatchSpan("Theme.PrimaryPressed", theme.PrimaryPressed, theme.Foreground),
			),
			themeSwatchRow(
				themeSwatchSpan("Theme.Accent", theme.Accent, theme.Foreground),
				themeTextSpan("Theme.AccentText", theme.AccentText),
			),
		}),
		themeGroup("state", []ui.Widget{
			themeSwatchRow(
				themeSwatchSpan("Theme.Success", theme.Success, theme.Foreground),
				themeTextSpan("Theme.SuccessText", theme.SuccessText),
			),
			themeSwatchRow(
				themeSwatchSpan("Theme.Warning", theme.Warning, theme.Foreground),
				themeTextSpan("Theme.WarningText", theme.WarningText),
			),
			themeSwatchRow(
				themeSwatchSpan("Theme.Danger", theme.Danger, theme.Foreground),
				themeTextSpan("Theme.DangerText", theme.DangerText),
			),
		}),
		themeGroup("text and selection", []ui.Widget{
			themeSwatchRow(
				themeTextSpan("Theme.MutedForeground", theme.MutedForeground),
				themeTextSpan("Theme.DisabledForeground", theme.DisabledForeground),
			),
			themeSwatchRow(
				themeSwatchSpan("Theme.Selection", theme.Selection, theme.Foreground),
			),
		}),
		ui.SizedBox{Height: 1},
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "Palette tones", Style: ui.Style{Attribute: ui.AttrBold}},
		}},
		paletteScale("neutral", theme.Palette.Neutral),
		paletteScale("red", theme.Palette.Red),
		paletteScale("green", theme.Palette.Green),
		paletteScale("yellow", theme.Palette.Yellow),
		paletteScale("blue", theme.Palette.Blue),
		paletteScale("magenta", theme.Palette.Magenta),
		paletteScale("cyan", theme.Palette.Cyan),
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

func themeGroup(name string, children []ui.Widget) ui.Widget {
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, Children: append([]ui.Widget{
		ui.RichText{Spans: []ui.TextSpan{
			{Text: "  "},
			{Text: name, Style: ui.Style{Attribute: ui.AttrBold}},
		}},
	}, children...)}
}

func tableHeader(value string) ui.Widget {
	return ui.Text{Value: value, Style: ui.Style{Attribute: ui.AttrBold}}
}

func themeSwatchRow(spans ...ui.TextSpan) ui.Widget {
	row := []ui.TextSpan{{Text: "    "}}
	for i, span := range spans {
		if i > 0 {
			row = append(row, ui.TextSpan{Text: "  "})
		}
		row = append(row, span)
	}
	return ui.RichText{Spans: row}
}

func themeSwatchSpan(name string, background, foreground ui.Color) ui.TextSpan {
	return ui.TextSpan{
		Text:  " " + padRight(name, 24) + " ",
		Style: ui.Style{Foreground: foreground, Background: background},
	}
}

func themeTextSpan(name string, foreground ui.Color) ui.TextSpan {
	return ui.TextSpan{
		Text:  " " + padRight(name, 24) + " ",
		Style: ui.Style{Foreground: foreground},
	}
}

func paletteScale(name string, scale ui.ColorScale) ui.Widget {
	tones := []ui.Color{
		scale.Tone50,
		scale.Tone100,
		scale.Tone200,
		scale.Tone300,
		scale.Tone400,
		scale.Tone500,
		scale.Tone600,
		scale.Tone700,
		scale.Tone800,
		scale.Tone900,
		scale.Tone950,
	}
	spans := []ui.TextSpan{{Text: padRight(name, 9)}}
	for _, tone := range tones {
		spans = append(spans, ui.TextSpan{Text: "  ", Style: ui.Style{Background: tone}})
	}
	return ui.RichText{Spans: spans}
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

func listDemoRow(theme ui.Theme, i int) ui.Widget {
	status := "ready"
	style := ui.Style{Foreground: theme.MutedForeground}
	switch {
	case i%9 == 0:
		status = "blocked"
		style = ui.Style{Foreground: theme.DangerText}
	case i%5 == 0:
		status = "running"
		style = ui.Style{Foreground: theme.SuccessText}
	case i%4 == 0:
		status = "queued"
		style = ui.Style{Foreground: theme.WarningText}
	}
	return ui.RichText{Spans: []ui.TextSpan{
		{Text: padLeft(strconv.Itoa(i), 4), Style: ui.Style{Attribute: ui.AttrBold}},
		{Text: "  "},
		{Text: "deploy target " + strconv.Itoa(100+i)},
		{Text: "  "},
		{Text: status, Style: style},
	}}
}

func chatDemoMessage(theme ui.Theme, i int, body string) ui.Widget {
	return ui.RichText{Spans: []ui.TextSpan{
		{Text: padLeft(strconv.Itoa(i+1), 3), Style: ui.Style{Foreground: theme.MutedForeground}},
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

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
