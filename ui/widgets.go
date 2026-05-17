package ui

type TextWidget struct {
	Value string
	Style Style
}

func Text(s string, opts ...TextOption) Widget {
	w := TextWidget{Value: s}
	for _, opt := range opts {
		opt(&w)
	}
	return w
}

type TextOption func(*TextWidget)

func TextStyle(style Style) TextOption { return func(w *TextWidget) { w.Style = style } }

func (w TextWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderText{Text: w.Value, Style: w.Style}
}
func (w TextWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderText)
	r.Text, r.Style = w.Value, w.Style
}

type RenderText struct {
	LeafRenderObject
	Text  string
	Style Style
}

func (r *RenderText) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(c.Constrain(ctx.MeasureText(r.Text, r.Style)))
}
func (r *RenderText) Paint(p *Painter, off Offset) { p.DrawText(off, r.Text, r.Style) }

type PaddingWidget struct {
	Insets      Insets
	ChildWidget Widget
}

func Padding(in Insets, child Widget) Widget { return PaddingWidget{Insets: in, ChildWidget: child} }
func (w PaddingWidget) Child() Widget        { return w.ChildWidget }
func (w PaddingWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderPadding{Insets: w.Insets}
}
func (w PaddingWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	ro.(*RenderPadding).Insets = w.Insets
}

type RenderPadding struct {
	SingleChildRenderObject
	Insets Insets
}

func (r *RenderPadding) Layout(ctx LayoutContext, c Constraints) {
	child := r.Child()
	if child == nil {
		r.SetSize(c.Constrain(Size{}))
		return
	}
	child.Layout(ctx, c.Deflate(r.Insets))
	r.SetSize(c.Constrain(child.Base().Size().Inflate(r.Insets)))
}
func (r *RenderPadding) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(Offset{X: r.Insets.Left, Y: r.Insets.Top}))
	}
}
func (r *RenderPadding) HitTest(*HitTestResult, Point) bool { return false }

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
func (r *RenderCenter) HitTest(*HitTestResult, Point) bool { return false }

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
	ro.Base().SetParentData(FlexParentData{Flex: flex})
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

type FocusWidget struct {
	Node        *FocusNode
	ChildWidget Widget
}

func Focus(node *FocusNode, child Widget) Widget { return FocusWidget{Node: node, ChildWidget: child} }
func (w FocusWidget) CreateElement() Element     { return &focusElement{} }

type focusElement struct {
	ElementBase
	child Element
}

func (e *focusElement) mounted() {
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.owner.app.registerFocusable(e)
}

func (e *focusElement) unmounted() {
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.detach(e)
	}
	e.owner.app.unregisterFocusable(e)
}

func (e *focusElement) Rebuild() {
	w := e.widget.(FocusWidget)
	if w.Node != nil {
		w.Node.attach(e.owner.app, e)
	}
	e.child = e.UpdateChild(e.child, w.ChildWidget, nil)
}

func (e *focusElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}

type KeymapWidget struct {
	Bindings    map[string]VoidCallback
	ChildWidget Widget
}

func Keymap(bindings map[string]VoidCallback, child Widget) Widget {
	return KeymapWidget{Bindings: bindings, ChildWidget: child}
}
func (w KeymapWidget) CreateElement() Element { return &keymapElement{} }

type keymapElement struct {
	ElementBase
	child Element
}

func (e *keymapElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(KeymapWidget).ChildWidget, nil)
}

func (e *keymapElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *keymapElement) HandleEvent(ctx EventContext, ev Event) EventResult {
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	for binding, cb := range e.widget.(KeymapWidget).Bindings {
		if key.MatchString(binding) {
			if cb != nil {
				cb(ctx)
			}
			return EventHandled
		}
	}
	return EventIgnored
}

type ButtonWidget struct {
	Label     string
	OnPressed VoidCallback
}

func Button(label string, onPressed VoidCallback) Widget {
	return ButtonWidget{Label: label, OnPressed: onPressed}
}
func (w ButtonWidget) CreateState() State { return &buttonState{} }

type buttonState struct {
	StateBase
	node FocusNode
}

func (s *buttonState) Build(ctx BuildContext) Widget {
	w := s.Widget().(ButtonWidget)
	return Focus(&s.node, Padding(Symmetric(1, 0), Text(w.Label)))
}

func (s *buttonState) HandleEvent(ctx EventContext, ev Event) EventResult {
	if ctx.Phase() != TargetPhase && ctx.Phase() != BubblePhase {
		return EventIgnored
	}
	key, ok := ev.(Key)
	if !ok {
		return EventIgnored
	}
	if key.MatchString("Enter") || key.MatchString("Space") {
		if cb := s.Widget().(ButtonWidget).OnPressed; cb != nil {
			cb(ctx)
		}
		return EventHandled
	}
	return EventIgnored
}
