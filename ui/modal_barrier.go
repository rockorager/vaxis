package ui

const defaultModalBarrierOpacity = 80

// ModalBarrier applies a translucent scrim over already-painted content.
//
// Place ModalBarrier in a Stack above background content and below a dialog or
// other modal surface. The barrier preserves existing graphemes and blends RGB
// foreground, background, and underline colors toward Color.
type ModalBarrier struct {
	// Color is the scrim target color. The zero value defaults to black.
	Color Color
	// Opacity controls the blend amount from 0 to 255. The zero value defaults to
	// a subtle modal dimming opacity.
	Opacity uint8
}

func (w ModalBarrier) CreateRenderObject(BuildContext) RenderObject {
	return &renderModalBarrier{Color: w.Color, Opacity: w.Opacity}
}

func (w ModalBarrier) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderModalBarrier)
	if r.Color != w.Color || r.Opacity != w.Opacity {
		r.Color = w.Color
		r.Opacity = w.Opacity
		r.MarkNeedsPaint()
	}
}

type renderModalBarrier struct {
	LeafRenderObject
	Color   Color
	Opacity uint8
}

func (r *renderModalBarrier) Layout(_ LayoutContext, c Constraints) {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	r.SetSize(c.Constrain(size))
}

func (r *renderModalBarrier) DryLayout(_ LayoutContext, c Constraints) Size {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func (r *renderModalBarrier) Paint(p *Painter, off Offset) {
	color := r.Color
	if color == 0 {
		color = RGB(0, 0, 0)
	}
	opacity := r.Opacity
	if opacity == 0 {
		opacity = defaultModalBarrierOpacity
	}
	size := r.Size()
	p.Scrim(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, color, opacity)
}

func (r *renderModalBarrier) HitTest(*HitTestResult, Point) bool {
	return true
}
