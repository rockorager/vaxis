package ui

// Alignment describes child placement within extra horizontal and vertical space.
type Alignment struct{ X, Y int }

var (
	// TopLeft aligns a child to the top-left corner.
	TopLeft = Alignment{X: -1, Y: -1}
	// TopCenter aligns a child to the top edge.
	TopCenter = Alignment{X: 0, Y: -1}
	// TopRight aligns a child to the top-right corner.
	TopRight = Alignment{X: 1, Y: -1}
	// CenterLeft aligns a child to the left edge.
	CenterLeft = Alignment{X: -1, Y: 0}
	// CenterAlign centers a child on both axes.
	CenterAlign = Alignment{X: 0, Y: 0}
	// CenterRight aligns a child to the right edge.
	CenterRight = Alignment{X: 1, Y: 0}
	// BottomLeft aligns a child to the bottom-left corner.
	BottomLeft = Alignment{X: -1, Y: 1}
	// BottomCenter aligns a child to the bottom edge.
	BottomCenter = Alignment{X: 0, Y: 1}
	// BottomRight aligns a child to the bottom-right corner.
	BottomRight = Alignment{X: 1, Y: 1}
)

// Align positions its child within the space allowed by its parent.
type Align struct {
	// Alignment controls where the child is placed.
	Alignment Alignment
	// Child is laid out loosely within the available space.
	Child Widget
}

func (w Align) WidgetChild() Widget {
	return w.Child
}

func (w Align) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderAlign{Alignment: w.Alignment}
}

func (w Align) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*renderAlign)
	if r.Alignment != w.Alignment {
		r.Alignment = w.Alignment
		r.MarkNeedsLayout()
	}
}

// renderAlign lays out one child and paints it at an aligned offset.
type renderAlign struct {
	SingleChildRenderObject
	Alignment Alignment
	offset    Offset
}

func (r *renderAlign) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	size := alignOuterSize(c, Size{})
	if child != nil {
		child.Layout(ctx, alignChildConstraints(c))
		cs := child.Base().Size()
		size = alignOuterSize(c, cs)
		r.offset = alignOffset(size, cs, r.Alignment)
	}
	r.SetSize(size)
}

func (r *renderAlign) DryLayout(ctx LayoutContext, c Constraints) Size {
	if child := r.Child(); child != nil {
		return alignOuterSize(c, DryLayout(ctx, child, alignChildConstraints(c)))
	}
	return alignOuterSize(c, Size{})
}

func (r *renderAlign) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *renderAlign) ChildOffset(RenderObject) Offset {
	return r.offset
}

func (r *renderAlign) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderAlign) HitTestSelf(Point) bool {
	return false
}

func alignChildConstraints(c Constraints) Constraints {
	return Constraints{MaxWidth: c.MaxWidth, MaxHeight: c.MaxHeight}
}

func alignOuterSize(c Constraints, child Size) Size {
	size := child
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func alignOffset(parent, child Size, a Alignment) Offset {
	dx := max(0, parent.Width-child.Width)
	dy := max(0, parent.Height-child.Height)
	return Offset{X: alignDelta(dx, a.X), Y: alignDelta(dy, a.Y)}
}

func alignDelta(delta, alignment int) int {
	switch {
	case alignment <= -1:
		return 0
	case alignment >= 1:
		return delta
	default:
		return delta / 2
	}
}
