package ui

type Alignment struct{ X, Y int }

var (
	TopLeft      = Alignment{X: -1, Y: -1}
	TopCenter    = Alignment{X: 0, Y: -1}
	TopRight     = Alignment{X: 1, Y: -1}
	CenterLeft   = Alignment{X: -1, Y: 0}
	CenterAlign  = Alignment{X: 0, Y: 0}
	CenterRight  = Alignment{X: 1, Y: 0}
	BottomLeft   = Alignment{X: -1, Y: 1}
	BottomCenter = Alignment{X: 0, Y: 1}
	BottomRight  = Alignment{X: 1, Y: 1}
)

type Align struct {
	Alignment Alignment
	Child     Widget
}

func (w Align) ChildWidget() Widget { return w.Child }
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

func (r *RenderAlign) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *RenderAlign) ChildOffset(RenderObject) Offset { return r.offset }

func (r *RenderAlign) HitTest(*HitTestResult, Point) bool { return false }

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
