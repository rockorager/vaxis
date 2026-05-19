package ui

import "math"

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

type scrollOffsetController interface {
	scrollMetricsProvider
	ScrollByLines(int) bool
	ScrollByPages(int) bool
	ScrollToOffset(int) bool
	ScrollToStart() bool
	ScrollToEnd() bool
}

// Scrollbar paints and handles a vertical scrollbar over a scrollable child.
type Scrollbar struct {
	// Child is expected to expose ScrollMetrics from its render object.
	Child Widget
	// ThumbStyle overrides Theme.Scrollbar.Thumb when non-zero.
	ThumbStyle Style
	// TrackStyle overrides Theme.Scrollbar.Track when non-zero.
	TrackStyle Style
}

func (w Scrollbar) CreateState() State {
	return &scrollbarState{}
}

type scrollbarState struct {
	StateBase
	dragging bool
	grabRow  float64
}

func (s *scrollbarState) Build(BuildContext) Widget {
	w := s.Widget().(Scrollbar)
	return scrollbarView(w)
}

func (s *scrollbarState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	mouse, ok := ev.(Mouse)
	if !ok {
		return EventIgnored
	}
	r := s.renderObject()
	if r == nil {
		return EventIgnored
	}
	switch mouse.EventType {
	case EventPress:
		if mouse.Button != MouseLeftButton || !r.scrollbarColumn(mouse.Col) {
			return EventIgnored
		}
		metrics, ok := r.metrics()
		if !ok || metrics.MaxScrollOffset <= 0 {
			return EventIgnored
		}
		row := s.fractionalRow(ctx, mouse)
		thumb := scrollbarThumb(metrics)
		switch {
		case row < thumb.Top:
			return r.scrollByPages(-1)
		case row >= thumb.Top+thumb.Height:
			return r.scrollByPages(1)
		default:
			s.dragging = true
			s.grabRow = row - thumb.Top
			if ctx.app != nil {
				ctx.app.captureMouse(s.element)
			}
			return EventHandled
		}
	case EventMotion:
		if !s.dragging {
			return EventIgnored
		}
		if mouse.Button == MouseNoButton {
			s.stopDragging(ctx)
			return EventHandled
		}
		return r.scrollThumbTo(s.fractionalRow(ctx, mouse) - s.grabRow)
	case EventRelease:
		if s.dragging {
			s.stopDragging(ctx)
			return EventHandled
		}
	}
	return EventIgnored
}

func (s *scrollbarState) renderObject() *renderScrollbar {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderScrollbar); ok {
		return r
	}
	return nil
}

func (s *scrollbarState) fractionalRow(ctx EventContext, mouse Mouse) float64 {
	fractional := ctx.FractionalMousePoint(mouse)
	return float64(mouse.Row) + fractional.Row - math.Floor(fractional.Row)
}

func (s *scrollbarState) stopDragging(ctx EventContext) {
	s.dragging = false
	if ctx.app != nil {
		ctx.app.releaseMouseCapture(s.element)
	}
}

type scrollbarView struct {
	Child      Widget
	ThumbStyle Style
	TrackStyle Style
}

func (w scrollbarView) ChildWidget() Widget {
	return w.Child
}

func (w scrollbarView) CreateRenderObject(ctx BuildContext) RenderObject {
	theme := MustDepend[Theme](ctx)
	thumb, track := scrollbarStyles(theme, w.ThumbStyle, w.TrackStyle)
	return &renderScrollbar{ThumbStyle: thumb, TrackStyle: track}
}

func (w scrollbarView) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
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
		thumb = theme.Scrollbar.Thumb
	}
	if track == (Style{}) {
		track = theme.Scrollbar.Track
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
	thumb := scrollbarThumb(metrics)
	x := off.X + r.Size().Width - 1
	for y := 0; y < metrics.ViewportHeight; y++ {
		p.DrawCell(Point{X: x, Y: off.Y + y}, scrollbarCell(float64(y), thumb, r.ThumbStyle, r.TrackStyle))
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

func (r *renderScrollbar) controller() (scrollOffsetController, bool) {
	child := r.Child()
	if child == nil {
		return nil, false
	}
	controller, ok := child.(scrollOffsetController)
	return controller, ok
}

func (r *renderScrollbar) scrollbarColumn(col int) bool {
	return r.Size().Width > 0 && col == r.Size().Width-1
}

func (r *renderScrollbar) scrollByPages(pages int) EventResult {
	controller, ok := r.controller()
	if !ok {
		return EventIgnored
	}
	controller.ScrollByPages(pages)
	return EventHandled
}

func (r *renderScrollbar) scrollThumbTo(top float64) EventResult {
	controller, ok := r.controller()
	if !ok {
		return EventIgnored
	}
	metrics := controller.ScrollMetrics()
	thumb := scrollbarThumb(metrics)
	trackRange := float64(metrics.ViewportHeight) - thumb.Height
	if metrics.MaxScrollOffset <= 0 || trackRange <= 0 {
		return EventHandled
	}
	top = clampFloat(top, 0, trackRange)
	offset := int(math.Round(top * float64(metrics.MaxScrollOffset) / trackRange))
	controller.ScrollToOffset(offset)
	return EventHandled
}

type scrollbarThumbGeometry struct {
	Top    float64
	Height float64
}

func scrollbarThumb(metrics ScrollMetrics) scrollbarThumbGeometry {
	if metrics.ViewportHeight <= 0 || metrics.ContentHeight <= 0 {
		return scrollbarThumbGeometry{}
	}
	viewport := float64(metrics.ViewportHeight)
	thumbHeight := viewport * viewport / float64(metrics.ContentHeight)
	thumbHeight = clampFloat(thumbHeight, 1, viewport)
	trackRange := viewport - thumbHeight
	if trackRange <= 0 || metrics.MaxScrollOffset <= 0 {
		return scrollbarThumbGeometry{Height: thumbHeight}
	}
	offset := clampInt(metrics.ScrollOffset, 0, metrics.MaxScrollOffset)
	top := float64(offset) * trackRange / float64(metrics.MaxScrollOffset)
	return scrollbarThumbGeometry{Top: top, Height: thumbHeight}
}

func scrollbarCell(row float64, thumb scrollbarThumbGeometry, thumbStyle, trackStyle Style) Cell {
	coverageStart := maxFloat(row, thumb.Top)
	coverageEnd := minFloat(row+1, thumb.Top+thumb.Height)
	coverage := clampFloat(coverageEnd-coverageStart, 0, 1)
	thumbColor := scrollbarColor(thumbStyle)
	trackColor := scrollbarColor(trackStyle)
	switch {
	case coverage <= 0:
		return scrollbarFillCell(trackStyle, trackColor)
	case coverage >= 1:
		return scrollbarFillCell(thumbStyle, thumbColor)
	case coverageStart > row:
		return Cell{
			Character: Character{Grapheme: lowerBlock(coverage), Width: 1},
			Style:     Style{Foreground: thumbColor, Background: trackColor, Attribute: thumbStyle.Attribute},
		}
	default:
		return Cell{
			Character: Character{Grapheme: lowerBlock(1 - coverage), Width: 1},
			Style:     Style{Foreground: trackColor, Background: thumbColor, Attribute: thumbStyle.Attribute},
		}
	}
}

func scrollbarFillCell(style Style, fill Color) Cell {
	style.Background = fill
	return Cell{Character: Character{Grapheme: " ", Width: 1}, Style: style}
}

func scrollbarColor(style Style) Color {
	if style.Background != 0 {
		return style.Background
	}
	return style.Foreground
}

func lowerBlock(fraction float64) string {
	switch int(math.Round(clampFloat(fraction, 0, 1) * 8)) {
	case 1:
		return "▁"
	case 2:
		return "▂"
	case 3:
		return "▃"
	case 4:
		return "▄"
	case 5:
		return "▅"
	case 6:
		return "▆"
	case 7:
		return "▇"
	case 8:
		return "█"
	default:
		return " "
	}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
