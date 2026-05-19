package ui

// Axis identifies the main direction for a Flex.
type Axis int

const (
	// Horizontal lays out children from left to right.
	Horizontal Axis = iota
	// Vertical lays out children from top to bottom.
	Vertical
)

// MainAxisSize controls how much space a Flex occupies on its main axis.
type MainAxisSize int

const (
	// MainAxisSizeMax expands the Flex to the incoming maximum main-axis size.
	MainAxisSizeMax MainAxisSize = iota
	// MainAxisSizeMin sizes the Flex to its children on the main axis.
	MainAxisSizeMin
)

// MainAxisAlignment controls how free space is distributed on a Flex main axis.
type MainAxisAlignment int

const (
	// MainAxisStart places children at the start of the main axis.
	MainAxisStart MainAxisAlignment = iota
	// MainAxisEnd places children at the end of the main axis.
	MainAxisEnd
	// MainAxisCenter centers children on the main axis.
	MainAxisCenter
	// MainAxisSpaceBetween distributes free space between children.
	MainAxisSpaceBetween
	// MainAxisSpaceAround distributes free space around children.
	MainAxisSpaceAround
	// MainAxisSpaceEvenly distributes free space evenly before, between, and after children.
	MainAxisSpaceEvenly
)

// CrossAxisAlignment controls how children are placed on a Flex cross axis.
type CrossAxisAlignment int

const (
	// CrossAxisCenter centers children on the cross axis.
	CrossAxisCenter CrossAxisAlignment = iota
	// CrossAxisStart places children at the start of the cross axis.
	CrossAxisStart
	// CrossAxisEnd places children at the end of the cross axis.
	CrossAxisEnd
	// CrossAxisStretch tightens children to the maximum cross-axis size.
	CrossAxisStretch
)

// FlexFit controls how a flexible child uses its allocated main-axis space.
type FlexFit int

const (
	// FlexFitTight forces the child to fill its allocated flex space.
	FlexFitTight FlexFit = iota
	// FlexFitLoose allows the child to be smaller than its allocated flex space.
	FlexFitLoose
)

// Flex lays out children in a horizontal or vertical run.
type Flex struct {
	// Axis is the direction children are placed.
	Axis Axis
	// MainAxisSize controls whether the flex expands or shrinks on its main axis.
	MainAxisSize MainAxisSize
	// MainAxisAlignment controls how extra main-axis space is distributed.
	MainAxisAlignment MainAxisAlignment
	// CrossAxisAlignment controls child placement on the cross axis.
	CrossAxisAlignment CrossAxisAlignment
	// ChildrenWidget is the ordered list of children.
	ChildrenWidget []Widget
}

// Row creates a horizontal Flex.
func Row(children ...Widget) Widget {
	return Flex{Axis: Horizontal, ChildrenWidget: children}
}

// Column creates a vertical Flex.
func Column(children ...Widget) Widget {
	return Flex{Axis: Vertical, ChildrenWidget: children}
}

func (w Flex) Children() []Widget {
	return w.ChildrenWidget
}

func (w Flex) CreateRenderObject(ctx BuildContext) RenderObject {
	return &renderFlex{Axis: w.Axis, MainAxisSize: w.MainAxisSize, MainAxisAlignment: w.MainAxisAlignment, CrossAxisAlignment: w.CrossAxisAlignment}
}

func (w Flex) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*renderFlex)
	if r.Axis != w.Axis || r.MainAxisSize != w.MainAxisSize || r.MainAxisAlignment != w.MainAxisAlignment || r.CrossAxisAlignment != w.CrossAxisAlignment {
		r.Axis = w.Axis
		r.MainAxisSize = w.MainAxisSize
		r.MainAxisAlignment = w.MainAxisAlignment
		r.CrossAxisAlignment = w.CrossAxisAlignment
		r.MarkNeedsLayout()
	}
}

// FlexParentData stores layout data for children of renderFlex.
type FlexParentData struct {
	// Flex is the child's flex factor.
	Flex int
	// Fit controls whether the child must fill its flex allocation.
	Fit FlexFit
	// Offset is the child paint offset computed by renderFlex.
	Offset Offset
}

// RenderOffset returns the child's paint offset.
func (d FlexParentData) RenderOffset() Offset {
	return d.Offset
}

// renderFlex lays out render children along a main axis.
type renderFlex struct {
	MultiChildRenderObject
	Axis               Axis
	MainAxisSize       MainAxisSize
	MainAxisAlignment  MainAxisAlignment
	CrossAxisAlignment CrossAxisAlignment
}

func (r *renderFlex) Layout(ctx LayoutContext, c Constraints) {
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

func (r *renderFlex) DryLayout(ctx LayoutContext, c Constraints) Size {
	size, _ := r.layoutSizes(ctx, c, true)
	return size
}

func (r *renderFlex) layoutSizes(ctx LayoutContext, c Constraints, dry bool) (Size, []Size) {
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

func (r *renderFlex) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
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

func (r *renderFlex) childConstraints(c Constraints, flexMain int, fit FlexFit) Constraints {
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

func (r *renderFlex) flexSize(c Constraints, mainUsed, crossUsed int) Size {
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

func (r *renderFlex) Paint(p *Painter, off Offset) {
	for _, child := range r.Children() {
		pd, _ := child.Base().ParentData().(FlexParentData)
		child.Paint(p, off.Add(pd.Offset))
	}
}

func (r *renderFlex) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderFlex) SelectionChildOffset(child RenderObject) Offset {
	off := r.ChildOffset(child)
	extra := 0
	for _, candidate := range r.Children() {
		if candidate == child {
			break
		}
		visual := candidate.Base().Size()
		logical := selectionSize(candidate)
		if r.Axis == Vertical {
			extra += logical.Height - visual.Height
		} else {
			extra += logical.Width - visual.Width
		}
	}
	if r.Axis == Vertical {
		off.Y += extra
	} else {
		off.X += extra
	}
	return off
}

func (r *renderFlex) SelectionSize() Size {
	size := r.Size()
	mainUsed := 0
	crossUsed := 0
	for _, child := range r.Children() {
		childSize := selectionSize(child)
		if r.Axis == Vertical {
			mainUsed += childSize.Height
			crossUsed = max(crossUsed, childSize.Width)
		} else {
			mainUsed += childSize.Width
			crossUsed = max(crossUsed, childSize.Height)
		}
	}
	if r.Axis == Vertical {
		size.Height = max(size.Height, mainUsed)
		size.Width = max(size.Width, crossUsed)
	} else {
		size.Width = max(size.Width, mainUsed)
		size.Height = max(size.Height, crossUsed)
	}
	return size
}

// ExpandedWidget gives a Flex child a tight share of remaining space.
type ExpandedWidget struct {
	// Flex is the share of remaining space assigned to the child.
	Flex int
	// ChildWidget is the wrapped child.
	ChildWidget Widget
}

// Expanded wraps child with a tight flex factor of 1.
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

// FlexibleWidget gives a Flex child a configurable share of remaining space.
type FlexibleWidget struct {
	// Flex is the share of remaining space assigned to the child.
	Flex int
	// Fit controls whether the child must fill its flex allocation.
	Fit FlexFit
	// ChildWidget is the wrapped child.
	ChildWidget Widget
}

// Flexible wraps child with a loose flex factor of 1.
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
