package ui

// ProgressBar paints a determinate horizontal progress indicator.
//
// Value is clamped to the range 0 through 1. The bar expands to the available
// width when bounded, otherwise it uses Width or a one-cell fallback.
type ProgressBar struct {
	// Value is the completed fraction, from 0 to 1.
	Value float64
	// Width is used when greater than zero or when layout is unbounded.
	Width int
	// FilledStyle overrides Theme.ProgressBar.Filled when non-zero.
	FilledStyle Style
	// EmptyStyle overrides Theme.ProgressBar.Empty when non-zero.
	EmptyStyle Style
	// GradientStart is the filled color at the start of the bar when non-zero.
	GradientStart Color
	// GradientEnd is the filled color at the end of the bar when non-zero.
	GradientEnd Color
}

func (w ProgressBar) CreateRenderObject(ctx BuildContext) RenderObject {
	filled, empty := progressBarStyles(MustDepend[Theme](ctx), w.FilledStyle, w.EmptyStyle)
	return &renderProgressBar{
		Value:         w.Value,
		Width:         w.Width,
		FilledStyle:   filled,
		EmptyStyle:    empty,
		GradientStart: w.GradientStart,
		GradientEnd:   w.GradientEnd,
	}
}

func (w ProgressBar) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	filled, empty := progressBarStyles(MustDepend[Theme](ctx), w.FilledStyle, w.EmptyStyle)
	r := ro.(*renderProgressBar)
	if r.Value != w.Value || r.Width != w.Width || r.FilledStyle != filled || r.EmptyStyle != empty || r.GradientStart != w.GradientStart || r.GradientEnd != w.GradientEnd {
		r.Value = w.Value
		r.Width = w.Width
		r.FilledStyle = filled
		r.EmptyStyle = empty
		r.GradientStart = w.GradientStart
		r.GradientEnd = w.GradientEnd
		r.MarkNeedsLayout()
	}
}

func progressBarStyles(theme Theme, filled, empty Style) (Style, Style) {
	if filled == (Style{}) {
		filled = theme.ProgressBar.Filled
	}
	if empty == (Style{}) {
		empty = theme.ProgressBar.Empty
	}
	return filled, empty
}

type renderProgressBar struct {
	LeafRenderObject
	Value         float64
	Width         int
	FilledStyle   Style
	EmptyStyle    Style
	GradientStart Color
	GradientEnd   Color
}

func (r *renderProgressBar) Layout(_ LayoutContext, c Constraints) {
	r.SetSize(r.size(c))
}

func (r *renderProgressBar) DryLayout(_ LayoutContext, c Constraints) Size {
	return r.size(c)
}

func (r *renderProgressBar) size(c Constraints) Size {
	width := r.Width
	if c.HasBoundedWidth() {
		width = c.MaxWidth
	} else if width <= 0 {
		width = 1
	}
	return c.Constrain(Size{Width: width, Height: 1})
}

func (r *renderProgressBar) Paint(p *Painter, off Offset) {
	size := r.Size()
	progress := clampFloat(r.Value, 0, 1) * float64(size.Width)
	for x := 0; x < size.Width; x++ {
		fraction := clampFloat(progress-float64(x), 0, 1)
		filledStyle := r.filledStyleAt(x)
		cell := progressBarCell(fraction, filledStyle, r.EmptyStyle)
		p.DrawCell(Point{X: off.X + x, Y: off.Y}, cell)
	}
}

func (r *renderProgressBar) filledStyleAt(x int) Style {
	style := r.FilledStyle
	if r.GradientStart == 0 || r.GradientEnd == 0 {
		return style
	}
	if color, ok := interpolateColor(r.GradientStart, r.GradientEnd, progressBarGradientPosition(x, r.Size().Width)); ok {
		style.Foreground = color
	}
	return style
}

func progressBarGradientPosition(x, width int) float64 {
	if width <= 1 {
		return 0
	}
	return float64(x) / float64(width-1)
}

func progressBarCell(fraction float64, filledStyle, emptyStyle Style) Cell {
	switch {
	case fraction <= 0:
		return Cell{Character: Character{Grapheme: " ", Width: 1}, Style: emptyStyle}
	case fraction >= 1:
		return Cell{Character: Character{Grapheme: "█", Width: 1}, Style: filledStyle}
	default:
		return Cell{
			Character: Character{Grapheme: horizontalBlock(fraction), Width: 1},
			Style: Style{
				Foreground:     progressBarColor(filledStyle),
				Background:     progressBarColor(emptyStyle),
				Attribute:      filledStyle.Attribute,
				UnderlineStyle: filledStyle.UnderlineStyle,
				UnderlineColor: filledStyle.UnderlineColor,
			},
		}
	}
}

func progressBarColor(style Style) Color {
	if style.Foreground != 0 {
		return style.Foreground
	}
	return style.Background
}

func interpolateColor(start, end Color, t float64) (Color, bool) {
	startParams := start.Params()
	endParams := end.Params()
	if len(startParams) != 3 || len(endParams) != 3 {
		return 0, false
	}
	t = clampFloat(t, 0, 1)
	return RGB(
		uint8(float64(startParams[0])+float64(int(endParams[0])-int(startParams[0]))*t),
		uint8(float64(startParams[1])+float64(int(endParams[1])-int(startParams[1]))*t),
		uint8(float64(startParams[2])+float64(int(endParams[2])-int(startParams[2]))*t),
	), true
}
