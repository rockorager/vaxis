package ui

type Axis int

const (
	Horizontal Axis = iota
	Vertical
)

type FlexWidget struct {
	Axis           Axis
	ChildrenWidget []Widget
}

func Row(children ...Widget) Widget     { return FlexWidget{Axis: Horizontal, ChildrenWidget: children} }
func Column(children ...Widget) Widget  { return FlexWidget{Axis: Vertical, ChildrenWidget: children} }
func (w FlexWidget) Children() []Widget { return w.ChildrenWidget }
func (w FlexWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderFlex{Axis: w.Axis}
}
func (w FlexWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	ro.(*RenderFlex).Axis = w.Axis
}

type FlexParentData struct {
	Flex   int
	Offset Offset
}

func (d FlexParentData) RenderOffset() Offset { return d.Offset }

type RenderFlex struct {
	MultiChildRenderObject
	Axis Axis
}

func (r *RenderFlex) Layout(ctx LayoutContext, c Constraints) {
	children := r.Children()
	mainUsed, cross := 0, 0
	flexTotal := 0
	for _, child := range children {
		if pd, _ := child.Base().ParentData().(FlexParentData); pd.Flex > 0 {
			flexTotal += pd.Flex
		}
	}
	for _, child := range children {
		pd, _ := child.Base().ParentData().(FlexParentData)
		if pd.Flex > 0 {
			continue
		}
		child.Layout(ctx, r.childConstraints(c, 0))
		s := child.Base().Size()
		mainUsed += mainSize(r.Axis, s)
		cross = max(cross, crossSize(r.Axis, s))
	}
	remaining := 0
	if maxMain(r.Axis, c) != Unbounded {
		remaining = max(0, maxMain(r.Axis, c)-mainUsed)
	}
	for _, child := range children {
		pd, _ := child.Base().ParentData().(FlexParentData)
		if pd.Flex <= 0 {
			continue
		}
		share := 0
		if flexTotal > 0 {
			share = remaining * pd.Flex / flexTotal
		}
		child.Layout(ctx, r.childConstraints(c, share))
		s := child.Base().Size()
		mainUsed += mainSize(r.Axis, s)
		cross = max(cross, crossSize(r.Axis, s))
	}
	pos := 0
	for _, child := range children {
		pd, _ := child.Base().ParentData().(FlexParentData)
		if r.Axis == Horizontal {
			pd.Offset = Offset{X: pos}
		} else {
			pd.Offset = Offset{Y: pos}
		}
		child.Base().SetParentData(pd)
		pos += mainSize(r.Axis, child.Base().Size())
	}
	r.SetSize(c.Constrain(sizeFromAxis(r.Axis, mainUsed, cross)))
}

func (r *RenderFlex) childConstraints(c Constraints, tightMain int) Constraints {
	if r.Axis == Horizontal {
		if tightMain > 0 {
			return Constraints{MinWidth: tightMain, MaxWidth: tightMain, MaxHeight: c.MaxHeight}
		}
		return Constraints{MaxWidth: Unbounded, MaxHeight: c.MaxHeight}
	}
	if tightMain > 0 {
		return Constraints{MaxWidth: c.MaxWidth, MinHeight: tightMain, MaxHeight: tightMain}
	}
	return Constraints{MaxWidth: c.MaxWidth, MaxHeight: Unbounded}
}

func (r *RenderFlex) Paint(p *Painter, off Offset) {
	for _, child := range r.Children() {
		pd, _ := child.Base().ParentData().(FlexParentData)
		child.Paint(p, off.Add(pd.Offset))
	}
}

func (r *RenderFlex) HitTest(*HitTestResult, Point) bool { return false }

type ExpandedWidget struct {
	Flex        int
	ChildWidget Widget
}

func Expanded(child Widget) Widget     { return ExpandedWidget{Flex: 1, ChildWidget: child} }
func (w ExpandedWidget) Child() Widget { return w.ChildWidget }
func (w ExpandedWidget) ApplyParentData(ro RenderObject) {
	flex := w.Flex
	if flex <= 0 {
		flex = 1
	}
	pd, _ := ro.Base().ParentData().(FlexParentData)
	if pd.Flex == flex {
		return
	}
	pd.Flex = flex
	ro.Base().SetParentData(pd)
	if parent := ro.Base().parent; parent != nil {
		parent.Base().MarkNeedsLayout()
	}
}

func maxFinite(v int) int {
	if v == Unbounded {
		return 0
	}
	return v
}
func mainSize(axis Axis, s Size) int {
	if axis == Horizontal {
		return s.Width
	}
	return s.Height
}
func crossSize(axis Axis, s Size) int {
	if axis == Horizontal {
		return s.Height
	}
	return s.Width
}
func maxMain(axis Axis, c Constraints) int {
	if axis == Horizontal {
		return c.MaxWidth
	}
	return c.MaxHeight
}
func sizeFromAxis(axis Axis, main, cross int) Size {
	if axis == Horizontal {
		return Size{Width: main, Height: cross}
	}
	return Size{Width: cross, Height: main}
}
