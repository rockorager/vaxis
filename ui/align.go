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

func (w Align) ChildWidget() Widget {
	return w.Child
}

func (w Align) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderAlign{Alignment: w.Alignment}
}

func (w Align) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderAlign)
	if r.Alignment != w.Alignment {
		r.Alignment = w.Alignment
		r.MarkNeedsLayout()
	}
}

// RenderAlign lays out one child and paints it at an aligned offset.
type RenderAlign struct {
	SingleChildRenderObject
	Alignment Alignment
	offset    Offset
}

func (r *RenderAlign) Layout(ctx LayoutContext, c Constraints) {
	size := c.Constrain(Size{Width: maxFinite(c.MaxWidth), Height: maxFinite(c.MaxHeight)})
	child := r.Child()
	if child != nil {
		child.Layout(ctx, Loose(size))
		cs := child.Base().Size()
		r.offset = alignOffset(size, cs, r.Alignment)
	}
	r.SetSize(size)
}

func (r *RenderAlign) DryLayout(_ LayoutContext, c Constraints) Size {
	return c.Constrain(Size{Width: maxFinite(c.MaxWidth), Height: maxFinite(c.MaxHeight)})
}

func (r *RenderAlign) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *RenderAlign) ChildOffset(RenderObject) Offset {
	return r.offset
}

func (r *RenderAlign) HitTest(*HitTestResult, Point) bool {
	return false
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
