package ui

type CenterWidget struct{ ChildWidget Widget }

func Center(child Widget) Widget                                            { return CenterWidget{ChildWidget: child} }
func (w CenterWidget) Child() Widget                                        { return w.ChildWidget }
func (w CenterWidget) CreateRenderObject(ctx BuildContext) RenderObject     { return &RenderCenter{} }
func (w CenterWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {}

type RenderCenter struct {
	SingleChildRenderObject
	offset Offset
}

func (r *RenderCenter) Layout(ctx LayoutContext, c Constraints) {
	size := c.Constrain(Size{Width: maxFinite(c.MaxWidth), Height: maxFinite(c.MaxHeight)})
	child := r.Child()
	if child != nil {
		child.Layout(ctx, Loose(size))
		cs := child.Base().Size()
		r.offset = Offset{X: max(0, (size.Width-cs.Width)/2), Y: max(0, (size.Height-cs.Height)/2)}
	}
	r.SetSize(size)
}

func (r *RenderCenter) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *RenderCenter) ChildOffset(RenderObject) Offset { return r.offset }

func (r *RenderCenter) HitTest(*HitTestResult, Point) bool { return false }
