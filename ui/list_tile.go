package ui

// ListTile is a focusable row with optional leading, subtitle, and trailing
// slots.
//
// ListTile calls OnPressed when activated by mouse, Enter, or Space. When
// Disabled is true the tile is not focusable and does not activate.
type ListTile struct {
	// Leading is painted before the title content when non-nil.
	Leading Widget
	// Title is the primary tile content.
	Title Widget
	// Subtitle is painted below Title when non-nil.
	Subtitle Widget
	// Trailing is painted at the end of the row when non-nil.
	Trailing Widget
	// Selected paints the tile with Theme.ListTile.Selected.
	Selected bool
	// Disabled prevents focus, hover, and activation when true.
	Disabled bool
	// OnPressed is called when the tile is activated.
	OnPressed VoidCallback
	// Padding overrides Theme.ListTile.Padding when non-zero.
	Padding Insets
	// Gap overrides Theme.ListTile.Gap when greater than zero.
	Gap int
	// MinHeight overrides Theme.ListTile.MinHeight when greater than zero.
	MinHeight int
}

func (w ListTile) CreateState() State {
	return &listTileState{}
}

type listTileState struct {
	StateBase
	node    FocusNode
	hovered bool
}

func (s *listTileState) Build(ctx BuildContext) Widget {
	w := s.Widget().(ListTile)
	theme := MustDepend[Theme](ctx)
	style := listTileStyle(w, s, theme.ListTile)
	child := listTileContent(w, theme.ListTile)
	childTheme := theme
	childTheme.Text = mergeStyle(childTheme.Text, style)
	child = Provider[Theme]{Value: childTheme, Child: child}
	tile := listTileSurface{
		Style:     style,
		MinHeight: listTileMinHeight(w, theme.ListTile),
		Child:     child,
	}
	if !w.Disabled && w.OnPressed != nil {
		s.node.onChange = s.MarkNeedsBuild
		return Focus(&s.node, tile)
	}
	return tile
}

func listTileStyle(w ListTile, s *listTileState, theme ListTileTheme) Style {
	style := theme.Normal
	if w.Selected {
		style = mergeStyle(style, theme.Selected)
	}
	if s.node.HasFocus() {
		style = mergeStyle(style, theme.Focused)
	}
	if s.hovered {
		style = mergeStyle(style, theme.Hovered)
	}
	if w.Disabled {
		style = mergeStyle(style, theme.Disabled)
	}
	return style
}

func listTileContent(w ListTile, theme ListTileTheme) Widget {
	padding := listTilePadding(w, theme)
	gap := listTileGap(w, theme)
	children := []Widget{}
	if w.Leading != nil {
		children = append(children, w.Leading, SizedBox{Width: gap})
	}
	children = append(children, Expanded(listTileMainContent(w)))
	if w.Trailing != nil {
		children = append(children, SizedBox{Width: gap}, w.Trailing)
	}
	return Padding(padding, Flex{Axis: Horizontal, CrossAxisAlignment: CrossAxisCenter, Children: children})
}

func listTileMainContent(w ListTile) Widget {
	switch {
	case w.Title != nil && w.Subtitle != nil:
		return Flex{
			Axis:               Vertical,
			MainAxisSize:       MainAxisSizeMin,
			CrossAxisAlignment: CrossAxisStart,
			Children:           []Widget{w.Title, w.Subtitle},
		}
	case w.Title != nil:
		return w.Title
	case w.Subtitle != nil:
		return w.Subtitle
	default:
		return SizedBox{Width: 0, Height: 1}
	}
}

func listTilePadding(w ListTile, theme ListTileTheme) Insets {
	if w.Padding != (Insets{}) {
		return w.Padding
	}
	if theme.Padding == (Insets{}) {
		return DefaultTheme().ListTile.Padding
	}
	return theme.Padding
}

func listTileGap(w ListTile, theme ListTileTheme) int {
	if w.Gap > 0 {
		return w.Gap
	}
	if theme.Gap < 0 {
		return DefaultTheme().ListTile.Gap
	}
	return theme.Gap
}

func listTileMinHeight(w ListTile, theme ListTileTheme) int {
	if w.MinHeight > 0 {
		return w.MinHeight
	}
	if theme.MinHeight <= 0 {
		return DefaultTheme().ListTile.MinHeight
	}
	return theme.MinHeight
}

func (s *listTileState) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	w := s.Widget().(ListTile)
	if w.Disabled || w.OnPressed == nil {
		return MouseShapeDefault
	}
	shape := MustDepend[Theme](s.Context()).ListTile.Mouse
	if shape == "" {
		return MouseShapeClickable
	}
	return shape
}

func (s *listTileState) HandleEvent(ctx EventContext, ev Event) EventResult {
	w := s.Widget().(ListTile)
	if w.Disabled || w.OnPressed == nil {
		return EventIgnored
	}
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	switch ev := ev.(type) {
	case Key:
		if keyIsRelease(ev) {
			return EventIgnored
		}
		if !ev.MatchString("Enter") && !ev.MatchString("Space") {
			return EventIgnored
		}
	case hoverExit:
		if s.hovered {
			s.SetState(func() { s.hovered = false })
		}
		return EventIgnored
	case Mouse:
		if ev.EventType == EventMotion {
			if !s.hovered {
				s.SetState(func() { s.hovered = true })
			}
			return EventIgnored
		}
		if ev.EventType != EventPress || ev.Button != MouseLeftButton {
			return EventIgnored
		}
	default:
		return EventIgnored
	}
	w.OnPressed(ctx)
	return EventHandled
}

type listTileSurface struct {
	Style     Style
	MinHeight int
	Child     Widget
}

func (w listTileSurface) WidgetChild() Widget {
	return w.Child
}

func (w listTileSurface) CreateRenderObject(BuildContext) RenderObject {
	return &renderListTileSurface{Style: w.Style, MinHeight: w.MinHeight}
}

func (w listTileSurface) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderListTileSurface)
	if r.Style != w.Style || r.MinHeight != w.MinHeight {
		r.Style = w.Style
		r.MinHeight = w.MinHeight
		r.MarkNeedsLayout()
	}
}

type renderListTileSurface struct {
	SingleChildRenderObject
	Style     Style
	MinHeight int
}

func (r *renderListTileSurface) Layout(ctx LayoutContext, c Constraints) {
	size := r.layout(ctx, c, false)
	r.SetSize(size)
}

func (r *renderListTileSurface) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *renderListTileSurface) layout(ctx LayoutContext, c Constraints, dry bool) Size {
	child := r.Child()
	size := Size{Height: r.MinHeight}
	if child != nil {
		if dry {
			size = DryLayout(ctx, child, c)
		} else {
			child.Layout(ctx, c)
			size = child.Base().Size()
		}
	}
	size.Height = max(size.Height, r.MinHeight)
	return c.Constrain(size)
}

func (r *renderListTileSurface) Paint(p *Painter, off Offset) {
	size := r.Size()
	if r.Style.Background != 0 {
		p.Fill(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, Cell{
			Character: Character{Grapheme: " ", Width: 1},
			Style:     r.Style,
		})
	}
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *renderListTileSurface) HitTest(*HitTestResult, Point) bool {
	return true
}
