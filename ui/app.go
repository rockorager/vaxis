package ui

import "time"

type App struct {
	build           *BuildOwner
	rootRO          RenderObject
	size            Size
	focusables      []Element
	focused         Element
	quit            bool
	theme           Theme
	frameRequested  bool
	mouseShape      MouseShape
	mouseShapeDirty bool
	dispatch        func(func())
	setTitle        func(string)
	copyToClipboard func(string)
	notify          func(string, string)
	pendingFocus    []Element
	hoverPath       []Element
	animations      map[*AnimationController]struct{}
}

func NewApp(root Widget, opts ...Option) *App {
	owner := NewBuildOwner()
	options := options{theme: DefaultTheme()}
	for _, opt := range opts {
		opt(&options)
	}
	app := &App{build: owner, theme: options.theme}
	app.dispatch = func(fn func()) { fn() }
	app.setTitle = func(string) {}
	app.copyToClipboard = func(string) {}
	app.notify = func(string, string) {}
	owner.app = app
	owner.Mount(Provider[Theme]{Value: app.theme, ChildWidget: root})
	return app
}

func (a *App) UpdateRoot(root Widget) {
	a.build.UpdateRoot(Provider[Theme]{Value: a.theme, ChildWidget: root})
}

func (a *App) Send(ev Event) {
	a.dispatchEvent(ev)
}

func (a *App) ShouldQuit() bool {
	return a.quit
}

func (a *App) RequestFrame() {
	a.frameRequested = true
}

func (a *App) FrameRequested() bool {
	return a.frameRequested
}

func (a *App) registerAnimation(controller *AnimationController) {
	if a.animations == nil {
		a.animations = make(map[*AnimationController]struct{})
	}
	a.animations[controller] = struct{}{}
	a.RequestFrame()
}

func (a *App) unregisterAnimation(controller *AnimationController) {
	delete(a.animations, controller)
}

func (a *App) tickAnimations(now time.Time) {
	for controller := range a.animations {
		if controller.tick(now) && controller.owner != nil && controller.owner.element != nil {
			controller.owner.MarkNeedsBuild()
		}
	}
}

func (a *App) hasActiveAnimations() bool {
	return len(a.animations) > 0
}

func (a *App) MouseShape() MouseShape {
	if a.mouseShape == "" {
		return MouseShapeDefault
	}
	return a.mouseShape
}

func (a *App) setMouseShape(shape MouseShape) {
	if shape == "" {
		shape = MouseShapeDefault
	}
	if a.MouseShape() == shape {
		return
	}
	a.mouseShape = shape
	a.mouseShapeDirty = true
}

func (a *App) consumeMouseShapeDirty() bool {
	dirty := a.mouseShapeDirty
	a.mouseShapeDirty = false
	return dirty
}

func (a *App) Pump(size Size) {
	a.frameRequested = false
	a.size = size
	a.build.BuildScope()
	a.flushFocusNotifications()
	a.rootRO = findRenderObject(a.build.Root())
	if a.rootRO != nil {
		a.rootRO.Layout(LayoutContext{}, Tight(size))
		clearNeedsLayout(a.rootRO)
	}
}

func (a *App) dispatchEvent(ev Event) EventResult {
	if mouse, ok := ev.(Mouse); ok {
		a.setMouseShape(MouseShapeDefault)
		path := a.hitPath(Point{X: mouse.Col, Y: mouse.Row})
		a.updateHoverPath(path)
		if len(path) > 0 {
			a.applyMouseShape(path, mouse)
			return a.dispatchPath(path, ev)
		}
	}
	if key, ok := ev.(Key); ok {
		if !keyIsRelease(key) && key.MatchString("Tab") {
			a.focusNext()
			return EventHandled
		}
		if !keyIsRelease(key) && key.MatchString("Shift+Tab") {
			a.focusPrevious()
			return EventHandled
		}
	}
	target := a.focused
	if target == nil {
		target = a.build.Root()
	}
	return a.dispatchPath(a.pathTo(target), ev)
}

func (a *App) updateHoverPath(next []Element) {
	for _, old := range a.hoverPath {
		if !elementInPath(old, next) {
			a.handle(old, TargetPhase, hoverExit{})
		}
	}
	a.hoverPath = next
}

func elementInPath(e Element, path []Element) bool {
	for _, candidate := range path {
		if candidate == e {
			return true
		}
	}
	return false
}

func (a *App) dispatchPath(path []Element, ev Event) EventResult {
	points := pathMousePoints(path, ev)
	for i := 0; i < len(path)-1; i++ {
		if a.handle(path[i], CapturePhase, eventForPathElement(ev, points, i)) == EventHandled {
			return EventHandled
		}
	}
	if len(path) > 0 && a.handle(path[len(path)-1], TargetPhase, eventForPathElement(ev, points, len(path)-1)) == EventHandled {
		return EventHandled
	}
	for i := len(path) - 2; i >= 0; i-- {
		if a.handle(path[i], BubblePhase, eventForPathElement(ev, points, i)) == EventHandled {
			return EventHandled
		}
	}
	return EventIgnored
}

func (a *App) applyMouseShape(path []Element, mouse Mouse) {
	ctx := EventContext{app: a, phase: TargetPhase}
	points := pathMousePoints(path, mouse)
	for i := len(path) - 1; i >= 0; i-- {
		mh, ok := path[i].(MouseShapeHandler)
		if !ok {
			continue
		}
		local := mouse
		if i < len(points) {
			local.Col = points[i].X
			local.Row = points[i].Y
		}
		shape := mh.MouseShape(ctx, local)
		if shape != "" && shape != MouseShapeDefault {
			a.setMouseShape(shape)
			return
		}
	}
}

func (a *App) handle(e Element, phase EventPhase, ev Event) EventResult {
	ctx := EventContext{app: a, phase: phase}
	h, ok := e.(EventHandler)
	if !ok {
		return EventIgnored
	}
	return h.HandleEvent(ctx, ev)
}

func pathMousePoints(path []Element, ev Event) []Point {
	mouse, ok := ev.(Mouse)
	if !ok {
		return nil
	}
	points := make([]Point, len(path))
	if len(path) == 0 {
		return points
	}
	points[0] = Point{X: mouse.Col, Y: mouse.Row}
	for i := 1; i < len(path); i++ {
		points[i] = childPoint(path[i-1], path[i], points[i-1])
	}
	return points
}

func eventForPathElement(ev Event, points []Point, idx int) Event {
	if len(points) == 0 || idx >= len(points) {
		return ev
	}
	mouse, ok := ev.(Mouse)
	if !ok {
		return ev
	}
	mouse.Col = points[idx].X
	mouse.Row = points[idx].Y
	return mouse
}

func (a *App) pathTo(target Element) []Element {
	var out []Element
	var walk func(Element) bool
	walk = func(e Element) bool {
		out = append(out, e)
		if e == target {
			return true
		}
		found := false
		e.VisitChildren(func(child Element) {
			if !found && walk(child) {
				found = true
			}
		})
		if found {
			return true
		}
		out = out[:len(out)-1]
		return false
	}
	walk(a.build.Root())
	return out
}

func (a *App) registerFocusable(e Element) {
	for _, existing := range a.focusables {
		if existing == e {
			return
		}
	}
	a.focusables = append(a.focusables, e)
	if a.focused == nil {
		a.setFocused(e)
	}
}

func (a *App) unregisterFocusable(e Element) {
	for i, existing := range a.focusables {
		if existing == e {
			a.focusables = append(a.focusables[:i], a.focusables[i+1:]...)
			break
		}
	}
	if a.focused == e {
		a.focused = nil
		if len(a.focusables) > 0 {
			a.setFocused(a.focusables[0])
		}
	}
}

func (a *App) focusNext() {
	a.moveFocus(1)
}

func (a *App) focusPrevious() {
	a.moveFocus(-1)
}

func (a *App) moveFocus(delta int) {
	if len(a.focusables) == 0 {
		return
	}
	idx := 0
	for i, e := range a.focusables {
		if e == a.focused {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(a.focusables)) % len(a.focusables)
	a.setFocused(a.focusables[idx])
}

func (a *App) setFocused(next Element) {
	if a.focused == next {
		return
	}
	old := a.focused
	a.focused = next
	a.notifyFocusChanged(old)
	a.notifyFocusChanged(next)
}

func (a *App) notifyFocusChanged(e Element) {
	if e == nil {
		return
	}
	if a.build.building {
		a.deferFocusNotification(e)
		return
	}
	if f, ok := e.Base().widget.(FocusWidget); ok && f.Node != nil && f.Node.onChange != nil {
		f.Node.onChange()
	}
}

func (a *App) deferFocusNotification(e Element) {
	for _, existing := range a.pendingFocus {
		if existing == e {
			return
		}
	}
	a.pendingFocus = append(a.pendingFocus, e)
}

func (a *App) flushFocusNotifications() {
	pending := a.pendingFocus
	a.pendingFocus = nil
	for _, e := range pending {
		if e.Base().owner == nil {
			continue
		}
		a.notifyFocusChanged(e)
	}
}

func (a *App) hitPath(pt Point) []Element {
	return hitElement(a.build.Root(), pt)
}

func hitElement(e Element, pt Point) []Element {
	ro := ownRenderObject(e)
	if ro != nil && !pointInSize(pt, ro.Base().Size()) {
		return nil
	}
	var best []Element
	e.VisitChildren(func(child Element) {
		if best != nil {
			return
		}
		if path := hitElement(child, childPoint(e, child, pt)); path != nil {
			best = path
		}
	})
	if best != nil {
		return append([]Element{e}, best...)
	}
	if ro != nil {
		return []Element{e}
	}
	return nil
}

func childPoint(parent, child Element, pt Point) Point {
	pro := ownRenderObject(parent)
	if pro == nil {
		return pt
	}
	op, ok := pro.(ChildOffsetProvider)
	if !ok {
		return pt
	}
	ro := findRenderObject(child)
	if ro == nil {
		return pt
	}
	off := op.ChildOffset(ro)
	return Point{X: pt.X - off.X, Y: pt.Y - off.Y}
}

func pointInSize(pt Point, size Size) bool {
	return pt.X >= 0 && pt.Y >= 0 && pt.X < size.Width && pt.Y < size.Height
}

func ownRenderObject(e Element) RenderObject {
	if r, ok := e.(interface{ RenderObject() RenderObject }); ok {
		return r.RenderObject()
	}
	return nil
}

func (a *App) Paint(p *Painter) {
	if a.rootRO != nil {
		a.rootRO.Paint(p, Offset{})
		clearNeedsPaint(a.rootRO)
	}
}

func clearNeedsLayout(ro RenderObject) {
	ro.Base().ClearNeedsLayout()
	ro.VisitChildren(clearNeedsLayout)
}

func clearNeedsPaint(ro RenderObject) {
	ro.Base().ClearNeedsPaint()
	ro.VisitChildren(clearNeedsPaint)
}

type (
	Option  func(*options)
	options struct {
		theme    Theme
		hasTheme bool
	}
)

func WithTheme(theme Theme) Option {
	return func(o *options) { o.theme, o.hasTheme = theme, true }
}
