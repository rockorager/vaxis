package ui

type Axis int

const (
	Horizontal Axis = iota
	Vertical
)

type MainAxisSize int

const (
	MainAxisSizeMax MainAxisSize = iota
	MainAxisSizeMin
)

type MainAxisAlignment int

const (
	MainAxisStart MainAxisAlignment = iota
	MainAxisEnd
	MainAxisCenter
	MainAxisSpaceBetween
	MainAxisSpaceAround
	MainAxisSpaceEvenly
)

type CrossAxisAlignment int

const (
	CrossAxisCenter CrossAxisAlignment = iota
	CrossAxisStart
	CrossAxisEnd
	CrossAxisStretch
)

type FlexFit int

const (
	FlexFitTight FlexFit = iota
	FlexFitLoose
)

type Flex struct {
	Axis               Axis
	MainAxisSize       MainAxisSize
	MainAxisAlignment  MainAxisAlignment
	CrossAxisAlignment CrossAxisAlignment
	ChildrenWidget     []Widget
}

func Row(children ...Widget) Widget {
	return Flex{Axis: Horizontal, ChildrenWidget: children}
}

func Column(children ...Widget) Widget {
	return Flex{Axis: Vertical, ChildrenWidget: children}
}

func (w Flex) Children() []Widget {
	return w.ChildrenWidget
}

func (w Flex) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderFlex{Axis: w.Axis, MainAxisSize: w.MainAxisSize, MainAxisAlignment: w.MainAxisAlignment, CrossAxisAlignment: w.CrossAxisAlignment}
}

func (w Flex) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderFlex)
	if r.Axis != w.Axis || r.MainAxisSize != w.MainAxisSize || r.MainAxisAlignment != w.MainAxisAlignment || r.CrossAxisAlignment != w.CrossAxisAlignment {
		r.Axis = w.Axis
		r.MainAxisSize = w.MainAxisSize
		r.MainAxisAlignment = w.MainAxisAlignment
		r.CrossAxisAlignment = w.CrossAxisAlignment
		r.MarkNeedsLayout()
	}
}

type FlexParentData struct {
	Flex   int
	Fit    FlexFit
	Offset Offset
}

func (d FlexParentData) RenderOffset() Offset {
	return d.Offset
}

type RenderFlex struct {
	MultiChildRenderObject
	Axis               Axis
	MainAxisSize       MainAxisSize
	MainAxisAlignment  MainAxisAlignment
	CrossAxisAlignment CrossAxisAlignment
}

func (r *RenderFlex) Layout(ctx LayoutContext, c Constraints) {
	size, childSizes := r.layoutSizes(ctx, c, false)
	freeSpace := max(0, mainSize(r.Axis, size)-mainUsedSize(r.Axis, childSizes))
	leading, between := mainAxisGaps(r.MainAxisAlignment, freeSpace, len(childSizes))
	pos := leading
	for i, child := range r.Children() {
		pd, _ := child.Base().ParentData().(FlexParentData)
		crossOffset := crossAxisOffset(r.CrossAxisAlignment, crossSize(r.Axis, size), crossSize(r.Axis, childSizes[i]))
		if r.Axis == Horizontal {
			pd.Offset = Offset{X: pos, Y: crossOffset}
		} else {
			pd.Offset = Offset{X: crossOffset, Y: pos}
		}
		child.Base().SetParentData(pd)
		pos += mainSize(r.Axis, childSizes[i]) + between
	}
	r.SetSize(size)
}

func (r *RenderFlex) DryLayout(ctx LayoutContext, c Constraints) Size {
	size, _ := r.layoutSizes(ctx, c, true)
	return size
}

func (r *RenderFlex) layoutSizes(ctx LayoutContext, c Constraints, dry bool) (Size, []Size) {
	children := r.Children()
	childSizes := make([]Size, len(children))
	mainUsed, cross := 0, 0
	flexTotal := 0
	for _, child := range children {
		if pd, _ := child.Base().ParentData().(FlexParentData); pd.Flex > 0 {
			flexTotal += pd.Flex
		}
	}
	for i, child := range children {
		pd, _ := child.Base().ParentData().(FlexParentData)
		if pd.Flex > 0 {
			continue
		}
		s := r.layoutChild(ctx, child, r.childConstraints(c, 0, FlexFitLoose), dry)
		childSizes[i] = s
		mainUsed += mainSize(r.Axis, s)
		cross = max(cross, crossSize(r.Axis, s))
	}
	remaining := 0
	if maxMain(r.Axis, c) != Unbounded {
		remaining = max(0, maxMain(r.Axis, c)-mainUsed)
	}
	remainingFlex := flexTotal
	remainingSpace := remaining
	for i, child := range children {
		pd, _ := child.Base().ParentData().(FlexParentData)
		if pd.Flex <= 0 {
			continue
		}
		share := 0
		if remainingFlex > 0 {
			share = remainingSpace * pd.Flex / remainingFlex
		}
		remainingFlex -= pd.Flex
		remainingSpace -= share
		s := r.layoutChild(ctx, child, r.childConstraints(c, share, pd.Fit), dry)
		childSizes[i] = s
		mainUsed += mainSize(r.Axis, s)
		cross = max(cross, crossSize(r.Axis, s))
	}
	return r.flexSize(c, mainUsed, cross), childSizes
}

func (r *RenderFlex) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
	if dry {
		return DryLayout(ctx, child, c)
	}
	child.Layout(ctx, c)
	return child.Base().Size()
}

func mainUsedSize(axis Axis, sizes []Size) int {
	used := 0
	for _, size := range sizes {
		used += mainSize(axis, size)
	}
	return used
}

func (r *RenderFlex) childConstraints(c Constraints, flexMain int, fit FlexFit) Constraints {
	minMain := 0
	maxMain := Unbounded
	if flexMain > 0 {
		maxMain = flexMain
		if fit == FlexFitTight {
			minMain = flexMain
		}
	}
	minCross := 0
	maxCross := crossMax(r.Axis, c)
	if r.CrossAxisAlignment == CrossAxisStretch && maxCross != Unbounded {
		minCross = maxCross
	}
	if r.Axis == Horizontal {
		return Constraints{MinWidth: minMain, MaxWidth: maxMain, MinHeight: minCross, MaxHeight: maxCross}
	}
	return Constraints{MinWidth: minCross, MaxWidth: maxCross, MinHeight: minMain, MaxHeight: maxMain}
}

func (r *RenderFlex) flexSize(c Constraints, mainUsed, crossUsed int) Size {
	main := mainUsed
	if r.MainAxisSize == MainAxisSizeMax && maxMain(r.Axis, c) != Unbounded {
		main = maxMain(r.Axis, c)
	}
	cross := crossUsed
	if r.CrossAxisAlignment == CrossAxisStretch && crossMax(r.Axis, c) != Unbounded {
		cross = crossMax(r.Axis, c)
	}
	return c.Constrain(sizeFromAxis(r.Axis, main, cross))
}

func (r *RenderFlex) Paint(p *Painter, off Offset) {
	for _, child := range r.Children() {
		pd, _ := child.Base().ParentData().(FlexParentData)
		child.Paint(p, off.Add(pd.Offset))
	}
}

func (r *RenderFlex) HitTest(*HitTestResult, Point) bool {
	return false
}

type ExpandedWidget struct {
	Flex        int
	ChildWidget Widget
}

func Expanded(child Widget) Widget {
	return ExpandedWidget{Flex: 1, ChildWidget: child}
}

func (w ExpandedWidget) Child() Widget {
	return w.ChildWidget
}

func (w ExpandedWidget) ApplyParentData(ro RenderObject) {
	flex := w.Flex
	if flex <= 0 {
		flex = 1
	}
	applyFlexParentData(ro, flex, FlexFitTight)
}

type FlexibleWidget struct {
	Flex        int
	Fit         FlexFit
	ChildWidget Widget
}

func Flexible(child Widget) Widget {
	return FlexibleWidget{Flex: 1, Fit: FlexFitLoose, ChildWidget: child}
}

func (w FlexibleWidget) Child() Widget {
	return w.ChildWidget
}

func (w FlexibleWidget) ApplyParentData(ro RenderObject) {
	flex := w.Flex
	if flex <= 0 {
		flex = 1
	}
	fit := w.Fit
	applyFlexParentData(ro, flex, fit)
}

func applyFlexParentData(ro RenderObject, flex int, fit FlexFit) {
	pd, _ := ro.Base().ParentData().(FlexParentData)
	if pd.Flex == flex && pd.Fit == fit {
		return
	}
	pd.Flex = flex
	pd.Fit = fit
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

func crossMax(axis Axis, c Constraints) int {
	if axis == Horizontal {
		return c.MaxHeight
	}
	return c.MaxWidth
}

func sizeFromAxis(axis Axis, main, cross int) Size {
	if axis == Horizontal {
		return Size{Width: main, Height: cross}
	}
	return Size{Width: cross, Height: main}
}

func mainAxisGaps(alignment MainAxisAlignment, freeSpace, childCount int) (int, int) {
	if childCount == 0 {
		return 0, 0
	}
	switch alignment {
	case MainAxisEnd:
		return freeSpace, 0
	case MainAxisCenter:
		return freeSpace / 2, 0
	case MainAxisSpaceBetween:
		if childCount <= 1 {
			return 0, 0
		}
		return 0, freeSpace / (childCount - 1)
	case MainAxisSpaceAround:
		between := freeSpace / childCount
		return between / 2, between
	case MainAxisSpaceEvenly:
		between := freeSpace / (childCount + 1)
		return between, between
	default:
		return 0, 0
	}
}

func crossAxisOffset(alignment CrossAxisAlignment, container, child int) int {
	delta := max(0, container-child)
	switch alignment {
	case CrossAxisEnd:
		return delta
	case CrossAxisCenter:
		return delta / 2
	default:
		return 0
	}
}
