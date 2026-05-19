package ui

// ScrollMetrics describes the vertical scroll state of a render object.
type ScrollMetrics struct {
	// ScrollOffset is the current top visible row.
	ScrollOffset int
	// MaxScrollOffset is the largest valid scroll offset.
	MaxScrollOffset int
	// ViewportHeight is the visible row count.
	ViewportHeight int
	// ViewportWidth is the visible column count.
	ViewportWidth int
	// ContentHeight is the total scrollable row count.
	ContentHeight int
}

type scrollMetricsProvider interface {
	ScrollMetrics() ScrollMetrics
}

// Scrollbar paints a passive vertical scrollbar over a scrollable child.
type Scrollbar struct {
	// Child is expected to expose ScrollMetrics from its render object.
	Child Widget
	// ThumbStyle paints the scrollbar thumb. Theme.Text is used when empty.
	ThumbStyle Style
	// TrackStyle paints the scrollbar track. A dim Theme.Text is used when empty.
	TrackStyle Style
}

func (w Scrollbar) ChildWidget() Widget {
	return w.Child
}

func (w Scrollbar) CreateRenderObject(ctx BuildContext) RenderObject {
	theme := MustDepend[Theme](ctx)
	thumb, track := scrollbarStyles(theme, w.ThumbStyle, w.TrackStyle)
	return &renderScrollbar{ThumbStyle: thumb, TrackStyle: track}
}

func (w Scrollbar) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	theme := MustDepend[Theme](ctx)
	thumb, track := scrollbarStyles(theme, w.ThumbStyle, w.TrackStyle)
	r := ro.(*renderScrollbar)
	if r.ThumbStyle != thumb || r.TrackStyle != track {
		r.ThumbStyle = thumb
		r.TrackStyle = track
		r.MarkNeedsPaint()
	}
}

func scrollbarStyles(theme Theme, thumb, track Style) (Style, Style) {
	if thumb == (Style{}) {
		thumb = theme.Text
	}
	if track == (Style{}) {
		track = mergeStyle(theme.Text, Style{Attribute: AttrDim})
	}
	return thumb, track
}

type renderScrollbar struct {
	SingleChildRenderObject
	ThumbStyle Style
	TrackStyle Style
}

func (r *renderScrollbar) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.SetSize(c.Constrain(Size{}))
		return
	}
	child.Layout(ctx, c)
	r.SetSize(c.Constrain(child.Base().Size()))
}

func (r *renderScrollbar) DryLayout(ctx LayoutContext, c Constraints) Size {
	return DryLayout(ctx, r.Child(), c)
}

func (r *renderScrollbar) Paint(p *Painter, off Offset) {
	child := r.Child()
	if child != nil {
		child.Paint(p, off)
	}
	metrics, ok := r.metrics()
	if !ok || metrics.MaxScrollOffset <= 0 || r.Size().Width <= 0 || r.Size().Height <= 0 {
		return
	}
	thumbTop, thumbHeight := scrollbarThumb(metrics)
	x := off.X + r.Size().Width - 1
	for y := 0; y < metrics.ViewportHeight; y++ {
		style := r.TrackStyle
		grapheme := "│"
		if y >= thumbTop && y < thumbTop+thumbHeight {
			style = r.ThumbStyle
			grapheme = "█"
		}
		p.DrawCell(Point{X: x, Y: off.Y + y}, Cell{Character: Character{Grapheme: grapheme, Width: 1}, Style: style})
	}
}

func (r *renderScrollbar) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderScrollbar) metrics() (ScrollMetrics, bool) {
	child := r.Child()
	if child == nil {
		return ScrollMetrics{}, false
	}
	provider, ok := child.(scrollMetricsProvider)
	if !ok {
		return ScrollMetrics{}, false
	}
	metrics := provider.ScrollMetrics()
	if metrics.ViewportHeight > r.Size().Height {
		metrics.ViewportHeight = r.Size().Height
	}
	return metrics, true
}

func scrollbarThumb(metrics ScrollMetrics) (int, int) {
	if metrics.ViewportHeight <= 0 || metrics.ContentHeight <= 0 {
		return 0, 0
	}
	thumbHeight := max(1, metrics.ViewportHeight*metrics.ViewportHeight/metrics.ContentHeight)
	thumbHeight = min(metrics.ViewportHeight, thumbHeight)
	trackRange := metrics.ViewportHeight - thumbHeight
	if trackRange <= 0 || metrics.MaxScrollOffset <= 0 {
		return 0, thumbHeight
	}
	offset := clampInt(metrics.ScrollOffset, 0, metrics.MaxScrollOffset)
	return offset * trackRange / metrics.MaxScrollOffset, thumbHeight
}
