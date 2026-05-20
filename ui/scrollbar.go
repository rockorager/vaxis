package ui

import "math"

// ScrollMetrics describes the scroll state of a render object.
type ScrollMetrics struct {
	// ScrollOffset is the current visible row or column on the active axis.
	ScrollOffset int
	// MaxScrollOffset is the largest valid scroll offset.
	MaxScrollOffset int
	// ViewportHeight is the visible row count.
	ViewportHeight int
	// ViewportWidth is the visible column count.
	ViewportWidth int
	// ContentHeight is the total scrollable row count.
	ContentHeight int
	// ContentWidth is the total scrollable column count.
	ContentWidth int
}

type scrollMetricsProvider interface {
	ScrollMetrics() ScrollMetrics
}

type scrollAxisMetricsProvider interface {
	ScrollMetricsForAxis(ScrollAxis) ScrollMetrics
}

type scrollOffsetController interface {
	scrollMetricsProvider
	ScrollByLines(int) bool
	ScrollByPages(int) bool
	ScrollToOffset(int) bool
	ScrollToStart() bool
	ScrollToEnd() bool
}

type scrollAxisOffsetController interface {
	scrollAxisMetricsProvider
	ScrollByLinesAxis(ScrollAxis, int) bool
	ScrollByPagesAxis(ScrollAxis, int) bool
	ScrollToOffsetAxis(ScrollAxis, int) bool
	ScrollToStartAxis(ScrollAxis) bool
	ScrollToEndAxis(ScrollAxis) bool
}

// Scrollbar paints and handles a scrollbar over a scrollable child.
type Scrollbar struct {
	// Axis controls which edge the scrollbar occupies. The zero value is
	// vertical.
	Axis ScrollAxis
	// Child is expected to expose ScrollMetrics from its render object.
	Child Widget
	// ThumbStyle overrides Theme.Scrollbar.Thumb when non-zero.
	ThumbStyle Style
	// TrackStyle overrides Theme.Scrollbar.Track when non-zero.
	TrackStyle Style
	// FocusedThumbStyle overrides Theme.Scrollbar.FocusedThumb when non-zero.
	FocusedThumbStyle Style
	// FocusedTrackStyle overrides Theme.Scrollbar.FocusedTrack when non-zero.
	FocusedTrackStyle Style
}

func (w Scrollbar) CreateState() State {
	return &scrollbarState{}
}

type scrollbarState struct {
	StateBase
	dragging bool
	grabPos  float64
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
		if mouse.Button != MouseLeftButton || !r.scrollbarHit(mouse) {
			return EventIgnored
		}
		metrics, ok := r.metrics()
		if !ok || metrics.MaxScrollOffset <= 0 {
			return EventIgnored
		}
		pos := s.fractionalPosition(ctx, mouse, r.Axis)
		thumb := scrollbarThumb(r.Axis, metrics)
		switch {
		case pos < thumb.Top:
			return r.scrollByPages(-1)
		case pos >= thumb.Top+thumb.Height:
			return r.scrollByPages(1)
		default:
			s.dragging = true
			s.grabPos = pos - thumb.Top
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
		return r.scrollThumbTo(s.fractionalPosition(ctx, mouse, r.Axis) - s.grabPos)
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

func (s *scrollbarState) fractionalPosition(ctx EventContext, mouse Mouse, axis ScrollAxis) float64 {
	fractional := ctx.FractionalMousePoint(mouse)
	if axis == ScrollHorizontal {
		return float64(mouse.Col) + fractional.Col - math.Floor(fractional.Col)
	}
	return float64(mouse.Row) + fractional.Row - math.Floor(fractional.Row)
}

func (s *scrollbarState) stopDragging(ctx EventContext) {
	s.dragging = false
	if ctx.app != nil {
		ctx.app.releaseMouseCapture(s.element)
	}
}

type scrollbarView struct {
	Axis              ScrollAxis
	Child             Widget
	ThumbStyle        Style
	TrackStyle        Style
	FocusedThumbStyle Style
	FocusedTrackStyle Style
}

func (w scrollbarView) WidgetChild() Widget {
	return w.Child
}

func (w scrollbarView) CreateRenderObject(ctx BuildContext) RenderObject {
	theme := MustDepend[Theme](ctx)
	thumb, track, focusedThumb, focusedTrack := scrollbarStyles(theme, w.ThumbStyle, w.TrackStyle, w.FocusedThumbStyle, w.FocusedTrackStyle)
	return &renderScrollbar{Axis: w.Axis, ThumbStyle: thumb, TrackStyle: track, FocusedThumbStyle: focusedThumb, FocusedTrackStyle: focusedTrack}
}

func (w scrollbarView) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	theme := MustDepend[Theme](ctx)
	thumb, track, focusedThumb, focusedTrack := scrollbarStyles(theme, w.ThumbStyle, w.TrackStyle, w.FocusedThumbStyle, w.FocusedTrackStyle)
	r := ro.(*renderScrollbar)
	if r.Axis != w.Axis || r.ThumbStyle != thumb || r.TrackStyle != track || r.FocusedThumbStyle != focusedThumb || r.FocusedTrackStyle != focusedTrack {
		r.Axis = w.Axis
		r.ThumbStyle = thumb
		r.TrackStyle = track
		r.FocusedThumbStyle = focusedThumb
		r.FocusedTrackStyle = focusedTrack
		r.MarkNeedsPaint()
	}
}

func scrollbarStyles(theme Theme, thumb, track, focusedThumb, focusedTrack Style) (Style, Style, Style, Style) {
	thumbOverride := thumb != (Style{})
	trackOverride := track != (Style{})
	if thumb == (Style{}) {
		thumb = theme.Scrollbar.Thumb
	}
	if track == (Style{}) {
		track = theme.Scrollbar.Track
	}
	if focusedThumb == (Style{}) {
		if thumbOverride {
			focusedThumb = thumb
		} else {
			focusedThumb = theme.Scrollbar.FocusedThumb
		}
	}
	if focusedTrack == (Style{}) {
		if trackOverride {
			focusedTrack = track
		} else {
			focusedTrack = theme.Scrollbar.FocusedTrack
		}
	}
	if focusedThumb == (Style{}) {
		focusedThumb = thumb
	}
	if focusedTrack == (Style{}) {
		focusedTrack = track
	}
	return thumb, track, focusedThumb, focusedTrack
}

type renderScrollbar struct {
	SingleChildRenderObject
	Axis              ScrollAxis
	ThumbStyle        Style
	TrackStyle        Style
	FocusedThumbStyle Style
	FocusedTrackStyle Style
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
	thumb := scrollbarThumb(r.Axis, metrics)
	thumbStyle, trackStyle := r.ThumbStyle, r.TrackStyle
	if r.childHasFocus() {
		thumbStyle, trackStyle = r.FocusedThumbStyle, r.FocusedTrackStyle
	}
	if r.Axis == ScrollHorizontal {
		y := off.Y + r.Size().Height - 1
		for x := 0; x < metrics.ViewportWidth; x++ {
			pt := Point{X: off.X + x, Y: y}
			p.DrawCell(pt, horizontalScrollbarCell(p.Cell(pt.X, pt.Y), float64(x), thumb, thumbStyle, trackStyle))
		}
		return
	}
	x := off.X + r.Size().Width - 1
	for y := 0; y < metrics.ViewportHeight; y++ {
		p.DrawCell(Point{X: x, Y: off.Y + y}, scrollbarCell(r.Axis, float64(y), thumb, thumbStyle, trackStyle))
	}
}

func (r *renderScrollbar) childHasFocus() bool {
	if r.owner == nil || r.owner.focused.element == nil || r.Child() == nil {
		return false
	}
	focused := findRenderObject(r.owner.focused.element)
	for focused != nil {
		if focused == r.Child() {
			return true
		}
		focused = focused.Base().parent
	}
	return false
}

func (r *renderScrollbar) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderScrollbar) ScrollMetrics() ScrollMetrics {
	metrics, _ := r.metrics()
	return metrics
}

func (r *renderScrollbar) ScrollMetricsForAxis(axis ScrollAxis) ScrollMetrics {
	child := r.Child()
	if child == nil {
		return ScrollMetrics{}
	}
	if provider, ok := child.(scrollAxisMetricsProvider); ok {
		return r.constrainMetrics(provider.ScrollMetricsForAxis(axis))
	}
	if provider, ok := child.(scrollMetricsProvider); ok {
		return r.constrainMetrics(provider.ScrollMetrics())
	}
	return ScrollMetrics{}
}

func (r *renderScrollbar) ScrollByLinesAxis(axis ScrollAxis, lines int) bool {
	if controller, ok := r.axisController(); ok {
		return controller.ScrollByLinesAxis(axis, lines)
	}
	if controller, ok := r.controller(); ok {
		return controller.ScrollByLines(lines)
	}
	return false
}

func (r *renderScrollbar) ScrollByPagesAxis(axis ScrollAxis, pages int) bool {
	if controller, ok := r.axisController(); ok {
		return controller.ScrollByPagesAxis(axis, pages)
	}
	if controller, ok := r.controller(); ok {
		return controller.ScrollByPages(pages)
	}
	return false
}

func (r *renderScrollbar) ScrollToOffsetAxis(axis ScrollAxis, offset int) bool {
	if controller, ok := r.axisController(); ok {
		return controller.ScrollToOffsetAxis(axis, offset)
	}
	if controller, ok := r.controller(); ok {
		return controller.ScrollToOffset(offset)
	}
	return false
}

func (r *renderScrollbar) ScrollToStartAxis(axis ScrollAxis) bool {
	if controller, ok := r.axisController(); ok {
		return controller.ScrollToStartAxis(axis)
	}
	if controller, ok := r.controller(); ok {
		return controller.ScrollToStart()
	}
	return false
}

func (r *renderScrollbar) ScrollToEndAxis(axis ScrollAxis) bool {
	if controller, ok := r.axisController(); ok {
		return controller.ScrollToEndAxis(axis)
	}
	if controller, ok := r.controller(); ok {
		return controller.ScrollToEnd()
	}
	return false
}

func (r *renderScrollbar) metrics() (ScrollMetrics, bool) {
	child := r.Child()
	if child == nil {
		return ScrollMetrics{}, false
	}
	if provider, ok := child.(scrollAxisMetricsProvider); ok {
		return r.constrainMetrics(provider.ScrollMetricsForAxis(r.Axis)), true
	}
	provider, ok := child.(scrollMetricsProvider)
	if !ok {
		return ScrollMetrics{}, false
	}
	return r.constrainMetrics(provider.ScrollMetrics()), true
}

func (r *renderScrollbar) constrainMetrics(metrics ScrollMetrics) ScrollMetrics {
	if metrics.ViewportHeight > r.Size().Height {
		metrics.ViewportHeight = r.Size().Height
	}
	if metrics.ViewportWidth > r.Size().Width {
		metrics.ViewportWidth = r.Size().Width
	}
	return metrics
}

func (r *renderScrollbar) controller() (scrollOffsetController, bool) {
	child := r.Child()
	if child == nil {
		return nil, false
	}
	controller, ok := child.(scrollOffsetController)
	return controller, ok
}

func (r *renderScrollbar) axisController() (scrollAxisOffsetController, bool) {
	child := r.Child()
	if child == nil {
		return nil, false
	}
	controller, ok := child.(scrollAxisOffsetController)
	return controller, ok
}

func (r *renderScrollbar) scrollbarHit(mouse Mouse) bool {
	if r.Axis == ScrollHorizontal {
		return r.Size().Height > 0 && mouse.Row == r.Size().Height-1
	}
	return r.scrollbarColumn(mouse.Col)
}

func (r *renderScrollbar) scrollbarColumn(col int) bool {
	return r.Size().Width > 0 && col == r.Size().Width-1
}

func (r *renderScrollbar) scrollByPages(pages int) EventResult {
	if child := r.Child(); child != nil {
		if controller, ok := child.(scrollAxisOffsetController); ok {
			controller.ScrollByPagesAxis(r.Axis, pages)
			return EventHandled
		}
	}
	controller, ok := r.controller()
	if !ok {
		return EventIgnored
	}
	controller.ScrollByPages(pages)
	return EventHandled
}

func (r *renderScrollbar) scrollThumbTo(top float64) EventResult {
	if child := r.Child(); child != nil {
		if controller, ok := child.(scrollAxisOffsetController); ok {
			metrics := controller.ScrollMetricsForAxis(r.Axis)
			thumb := scrollbarThumb(r.Axis, metrics)
			trackRange := float64(scrollbarViewportLength(r.Axis, metrics)) - thumb.Height
			if metrics.MaxScrollOffset <= 0 || trackRange <= 0 {
				return EventHandled
			}
			top = clampFloat(top, 0, trackRange)
			offset := int(math.Round(top * float64(metrics.MaxScrollOffset) / trackRange))
			controller.ScrollToOffsetAxis(r.Axis, offset)
			return EventHandled
		}
	}
	controller, ok := r.controller()
	if !ok {
		return EventIgnored
	}
	metrics := controller.ScrollMetrics()
	thumb := scrollbarThumb(r.Axis, metrics)
	trackRange := float64(scrollbarViewportLength(r.Axis, metrics)) - thumb.Height
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

func scrollbarThumb(axis ScrollAxis, metrics ScrollMetrics) scrollbarThumbGeometry {
	viewport := scrollbarViewportLength(axis, metrics)
	content := scrollbarContentLength(axis, metrics)
	if viewport <= 0 || content <= 0 {
		return scrollbarThumbGeometry{}
	}
	viewportFloat := float64(viewport)
	thumbHeight := viewportFloat * viewportFloat / float64(content)
	thumbHeight = clampFloat(thumbHeight, 1, viewportFloat)
	trackRange := viewportFloat - thumbHeight
	if trackRange <= 0 || metrics.MaxScrollOffset <= 0 {
		return scrollbarThumbGeometry{Height: thumbHeight}
	}
	offset := clampInt(metrics.ScrollOffset, 0, metrics.MaxScrollOffset)
	top := float64(offset) * trackRange / float64(metrics.MaxScrollOffset)
	return scrollbarThumbGeometry{Top: top, Height: thumbHeight}
}

func scrollbarViewportLength(axis ScrollAxis, metrics ScrollMetrics) int {
	if axis == ScrollHorizontal {
		return metrics.ViewportWidth
	}
	return metrics.ViewportHeight
}

func scrollbarContentLength(axis ScrollAxis, metrics ScrollMetrics) int {
	if axis == ScrollHorizontal {
		return metrics.ContentWidth
	}
	return metrics.ContentHeight
}

func scrollbarCell(axis ScrollAxis, pos float64, thumb scrollbarThumbGeometry, thumbStyle, trackStyle Style) Cell {
	coverageStart := maxFloat(pos, thumb.Top)
	coverageEnd := minFloat(pos+1, thumb.Top+thumb.Height)
	coverage := clampFloat(coverageEnd-coverageStart, 0, 1)
	thumbColor := scrollbarColor(thumbStyle)
	trackColor := scrollbarColor(trackStyle)
	switch {
	case coverage <= 0:
		return scrollbarFillCell(trackStyle, trackColor)
	case coverage >= 1:
		return scrollbarFillCell(thumbStyle, thumbColor)
	case axis == ScrollHorizontal && coverageStart > pos:
		return Cell{
			Character: Character{Grapheme: horizontalBlock(1 - coverage), Width: 1},
			Style:     Style{Foreground: trackColor, Background: thumbColor, Attribute: thumbStyle.Attribute},
		}
	case axis == ScrollHorizontal:
		return Cell{
			Character: Character{Grapheme: horizontalBlock(coverage), Width: 1},
			Style:     Style{Foreground: thumbColor, Background: trackColor, Attribute: thumbStyle.Attribute},
		}
	case coverageStart > pos:
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

func horizontalScrollbarCell(base Cell, pos float64, thumb scrollbarThumbGeometry, thumbStyle, trackStyle Style) Cell {
	coverageStart := maxFloat(pos, thumb.Top)
	coverageEnd := minFloat(pos+1, thumb.Top+thumb.Height)
	coverage := clampFloat(coverageEnd-coverageStart, 0, 1)
	style := trackStyle
	fill := scrollbarColor(trackStyle)
	if coverage > 0 {
		style = thumbStyle
		fill = scrollbarColor(thumbStyle)
	}
	style.Foreground = fill
	style.Background = base.Background
	return Cell{Character: Character{Grapheme: "▄", Width: 1}, Style: style}
}

func horizontalBlock(fraction float64) string {
	switch int(math.Round(clampFloat(fraction, 0, 1) * 8)) {
	case 1:
		return "▏"
	case 2:
		return "▎"
	case 3:
		return "▍"
	case 4:
		return "▌"
	case 5:
		return "▋"
	case 6:
		return "▊"
	case 7:
		return "▉"
	case 8:
		return "█"
	default:
		return " "
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
