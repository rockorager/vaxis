package ui

import "time"

// App owns a widget tree and dispatches events, layout, painting, and focus.
type App struct {
	build                     *buildOwner
	root                      Widget
	rootRO                    RenderObject
	size                      Size
	window                    Resize
	focusables                []focusTarget
	focused                   focusTarget
	quit                      bool
	theme                     Theme
	themeSet                  ThemeSet
	hasThemeSet               bool
	frameRequested            bool
	mouseShape                MouseShape
	mouseShapeDirty           bool
	mouseCapture              element
	dispatch                  func(func())
	setTitle                  func(string)
	copyToClipboard           func(string)
	notify                    func(string, string)
	pendingFocus              []focusTarget
	pendingFocusFallback      bool
	pendingFocusFallbackIndex int
	pendingFocusFallbackID    string
	pendingFocusFallbackLabel string
	hoverPath                 []element
	animations                map[*AnimationController]struct{}
	profileOverlay            bool
	shortcuts                 ShortcutMap
}

// NewApp creates an app mounted with root.
func NewApp(root Widget, opts ...Option) *App {
	owner := newBuildOwner()
	options := options{theme: DefaultTheme()}
	for _, opt := range opts {
		opt(&options)
	}
	if options.shortcuts == nil {
		options.shortcuts = DefaultShortcuts()
	}
	app := &App{
		build:          owner,
		root:           root,
		theme:          options.theme,
		themeSet:       options.themeSet,
		hasThemeSet:    options.hasThemeSet,
		profileOverlay: options.profileOverlay,
		shortcuts:      cloneShortcuts(options.shortcuts),
	}
	app.dispatch = func(fn func()) { fn() }
	app.setTitle = func(string) {}
	app.copyToClipboard = func(string) {}
	app.notify = func(string, string) {}
	owner.app = app
	owner.Mount(app.rootWidget(root))
	return app
}

// UpdateRoot replaces the root widget while preserving compatible elements.
func (a *App) UpdateRoot(root Widget) {
	a.root = root
	a.build.UpdateRoot(a.rootWidget(root))
}

// Send dispatches ev through the widget tree.
func (a *App) Send(ev Event) {
	if resize, ok := ev.(Resize); ok {
		a.setResize(resize)
	}
	if update, ok := ev.(ColorThemeUpdate); ok {
		if mode, ok := themeModeFromColorThemeMode(update.Mode); ok {
			a.SetThemeMode(mode)
		}
	}
	a.dispatchEvent(ev)
}

// ShouldQuit reports whether a quit request has been made.
func (a *App) ShouldQuit() bool {
	return a.quit
}

// RequestFrame marks the app as needing another frame.
func (a *App) RequestFrame() {
	a.frameRequested = true
}

// FrameRequested reports whether the app needs another frame.
func (a *App) FrameRequested() bool {
	return a.frameRequested
}

// ProfileOverlay reports whether the profiling overlay is visible.
func (a *App) ProfileOverlay() bool {
	return a.profileOverlay
}

// SetProfileOverlay shows or hides the profiling overlay.
func (a *App) SetProfileOverlay(visible bool) {
	if a.profileOverlay == visible {
		return
	}
	a.profileOverlay = visible
	a.RequestFrame()
}

// SetTheme replaces the app theme and rebuilds theme dependents.
func (a *App) SetTheme(theme Theme) {
	if a.theme == theme {
		return
	}
	a.theme = theme
	if a.root != nil {
		a.build.UpdateRoot(a.rootWidget(a.root))
	}
	a.RequestFrame()
}

// SetThemeMode switches to the matching theme from a ThemeSet, if configured.
func (a *App) SetThemeMode(mode ThemeMode) bool {
	if !a.hasThemeSet {
		return false
	}
	a.SetTheme(a.themeSet.Theme(mode))
	return true
}

// ToggleProfileOverlay toggles the profiling overlay and returns its new state.
func (a *App) ToggleProfileOverlay() bool {
	a.SetProfileOverlay(!a.profileOverlay)
	return a.profileOverlay
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

func (a *App) tickFrameCallbacks(now time.Time) bool {
	if a.build.root == nil {
		return false
	}
	active := false
	walkElements(a.build.root, func(e element) {
		stateful, ok := e.(*statefulElement)
		if !ok {
			return
		}
		ticker, ok := stateful.state.(frameTicker)
		if ok && ticker.TickFrame(now) {
			active = true
		}
	})
	return active
}

func (a *App) hasActiveAnimations() bool {
	return len(a.animations) > 0
}

// MouseShape returns the current requested pointer shape.
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

func (a *App) setResize(size Resize) {
	a.window = size
}

// Pump rebuilds and lays out the widget tree for size.
func (a *App) Pump(size Size) {
	a.pumpProfiled(size)
}

func (a *App) pumpProfiled(size Size) (build, layout time.Duration) {
	a.frameRequested = false
	a.size = size
	if a.window.Cols != size.Width || a.window.Rows != size.Height {
		a.window = Resize{Cols: size.Width, Rows: size.Height}
	}
	start := time.Now()
	a.build.BuildScope()
	a.resolvePendingFocusFallback()
	a.flushFocusNotifications()
	a.rootRO = findRenderObject(a.build.Root())
	build = time.Since(start)
	if a.rootRO != nil {
		start = time.Now()
		a.rootRO.Layout(LayoutContext{}, Tight(size))
		clearNeedsLayout(a.rootRO)
		layout = time.Since(start)
	}
	return build, layout
}

func (a *App) dispatchEvent(ev Event) EventResult {
	if mouse, ok := ev.(Mouse); ok {
		a.setMouseShape(MouseShapeDefault)
		path := a.hitPath(Point{X: mouse.Col, Y: mouse.Row})
		a.updateHoverPath(path)
		if len(path) > 0 {
			a.applyMouseShape(path, mouse)
		}
		if a.mouseCapture != nil && mouse.EventType != EventPress {
			captured := a.pathTo(a.mouseCapture)
			if len(captured) == 0 {
				a.mouseCapture = nil
			} else {
				result := a.dispatchPath(captured, ev)
				if mouse.EventType == EventRelease {
					a.mouseCapture = nil
				}
				return result
			}
		}
		if len(path) > 0 {
			return a.dispatchPath(path, ev)
		}
	}
	target := a.focused.element
	var path []element
	if target != nil && target.Base().owner != nil {
		path = a.pathTo(target)
	}
	if len(path) == 0 {
		path = a.fallbackEventPath()
	}
	return a.dispatchPath(path, ev)
}

func (a *App) captureMouse(e element) {
	a.mouseCapture = e
}

func (a *App) releaseMouseCapture(e element) {
	if a.mouseCapture == e {
		a.mouseCapture = nil
	}
}

func (a *App) updateHoverPath(next []element) {
	for _, old := range a.hoverPath {
		if old.Base().owner == nil {
			continue
		}
		if !elementInPath(old, next) {
			a.handle(old, TargetPhase, hoverExit{})
		}
	}
	a.hoverPath = next
}

func elementInPath(e element, path []element) bool {
	for _, candidate := range path {
		if candidate == e {
			return true
		}
	}
	return false
}

func walkElements(e element, fn func(element)) {
	fn(e)
	e.VisitChildren(func(child element) { walkElements(child, fn) })
}

type frameTicker interface {
	TickFrame(time.Time) bool
}

func (a *App) dispatchPath(path []element, ev Event) EventResult {
	points := pathMousePoints(path, ev)
	var target element
	if len(path) > 0 {
		target = path[len(path)-1]
	}
	for i := 0; i < len(path)-1; i++ {
		if a.handleWithTarget(path[i], target, CapturePhase, eventForPathElement(ev, points, i)) == EventHandled {
			return EventHandled
		}
	}
	if len(path) > 0 && a.handleWithTarget(path[len(path)-1], target, TargetPhase, eventForPathElement(ev, points, len(path)-1)) == EventHandled {
		return EventHandled
	}
	for i := len(path) - 2; i >= 0; i-- {
		if a.handleWithTarget(path[i], target, BubblePhase, eventForPathElement(ev, points, i)) == EventHandled {
			return EventHandled
		}
	}
	return EventIgnored
}

func (a *App) applyMouseShape(path []element, mouse Mouse) {
	var target element
	if len(path) > 0 {
		target = path[len(path)-1]
	}
	ctx := EventContext{app: a, phase: TargetPhase, target: target}
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

func (a *App) handle(e element, phase EventPhase, ev Event) EventResult {
	return a.handleWithTarget(e, e, phase, ev)
}

func (a *App) handleWithTarget(e element, target element, phase EventPhase, ev Event) EventResult {
	ctx := EventContext{app: a, phase: phase, element: e, target: target}
	h, ok := e.(EventHandler)
	if !ok {
		return EventIgnored
	}
	return h.HandleEvent(ctx, ev)
}

func (a *App) rootWidget(root Widget) Widget {
	return Actions{
		Bindings: map[IntentType]ActionFunc{
			NextFocusIntentType: func(ctx EventContext, intent Intent) EventResult {
				ctx.FocusNext()
				return EventHandled
			},
			PreviousFocusIntentType: func(ctx EventContext, intent Intent) EventResult {
				ctx.FocusPrevious()
				return EventHandled
			},
		},
		Child: Shortcuts{
			Bindings: cloneShortcuts(a.shortcuts),
			Child:    Provider[Theme]{Value: a.theme, Child: root},
		},
	}
}

func pathMousePoints(path []element, ev Event) []Point {
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

func (a *App) pathTo(target element) []element {
	var out []element
	var walk func(element) bool
	walk = func(e element) bool {
		out = append(out, e)
		if e == target {
			return true
		}
		found := false
		e.VisitChildren(func(child element) {
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

func (a *App) fallbackEventPath() []element {
	for _, target := range a.focusables {
		if target.element == nil || target.element.Base().owner == nil {
			continue
		}
		if path := a.pathTo(target.element); len(path) > 0 {
			return path
		}
	}
	return firstElementPath(a.build.Root())
}

func firstElementPath(root element) []element {
	if root == nil {
		return nil
	}
	path := []element{root}
	root.VisitChildren(func(child element) {
		if len(path) == 1 {
			path = append(path, firstElementPath(child)...)
		}
	})
	return path
}

func (a *App) registerFocusable(e element) {
	a.registerFocusTarget(focusTarget{element: e, index: elementFocusIndex})
}

func (a *App) registerFocusTarget(target focusTarget) {
	for _, existing := range a.focusables {
		if existing == target {
			return
		}
	}
	a.focusables = append(a.focusables, target)
	if a.focused.element == nil && !a.pendingFocusFallback {
		a.setFocused(target)
	}
}

func (a *App) unregisterFocusable(e element) {
	a.unregisterFocusTarget(focusTarget{element: e, index: elementFocusIndex})
}

func (a *App) unregisterFocusTarget(target focusTarget) {
	removed := -1
	focusedIndex := -1
	for i, existing := range a.focusables {
		if existing == a.focused {
			focusedIndex = i
		}
		if existing == target {
			a.focusables = append(a.focusables[:i], a.focusables[i+1:]...)
			removed = i
			break
		}
	}
	if a.focused == target {
		id := debugFocusTargetID(debugElementID(target.element), target.index)
		label := a.debugFocusLabel(target.element, target.index)
		old := a.focused
		a.focused = focusTarget{}
		a.notifyFocusChanged(old)
		if a.build.building {
			a.pendingFocusFallback = true
			if removed < 0 {
				removed = focusedIndex
			}
			if removed < 0 {
				removed = 0
			}
			a.pendingFocusFallbackIndex = removed
			a.pendingFocusFallbackID = id
			a.pendingFocusFallbackLabel = label
			return
		}
		if len(a.focusables) > 0 {
			a.setFocused(a.focusables[0])
		}
	}
}

func (a *App) resolvePendingFocusFallback() {
	if !a.pendingFocusFallback {
		return
	}
	idx := a.pendingFocusFallbackIndex
	id := a.pendingFocusFallbackID
	label := a.pendingFocusFallbackLabel
	a.pendingFocusFallback = false
	a.pendingFocusFallbackIndex = 0
	a.pendingFocusFallbackID = ""
	a.pendingFocusFallbackLabel = ""
	if a.focused.element != nil || len(a.focusables) == 0 {
		return
	}
	if id != "" {
		for _, target := range a.focusables {
			if debugFocusTargetID(debugElementID(target.element), target.index) == id {
				a.setFocused(target)
				return
			}
		}
	}
	if label != "" {
		for _, target := range a.focusables {
			if a.debugFocusLabel(target.element, target.index) == label {
				a.setFocused(target)
				return
			}
		}
	}
	idx = clamp(idx, 0, len(a.focusables)-1)
	a.setFocused(a.focusables[idx])
}

func (a *App) focusNext() {
	if scope := a.focusTrapFor(a.focused.element); scope != nil {
		a.moveFocusWithin(scope, 1)
		return
	}
	a.moveFocus(1)
}

func (a *App) focusPrevious() {
	if scope := a.focusTrapFor(a.focused.element); scope != nil {
		a.moveFocusWithin(scope, -1)
		return
	}
	a.moveFocus(-1)
}

func (a *App) moveFocus(delta int) {
	if len(a.focusables) == 0 {
		return
	}
	idx := 0
	for i, target := range a.focusables {
		if target == a.focused {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(a.focusables)) % len(a.focusables)
	a.setFocused(a.focusables[idx])
}

func (a *App) moveFocusWithin(scope element, delta int) {
	focusables := a.focusablesWithin(scope)
	if len(focusables) == 0 {
		return
	}
	idx := 0
	for i, target := range focusables {
		if target == a.focused {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(focusables)) % len(focusables)
	a.setFocused(focusables[idx])
}

func (a *App) focusFirstWithin(scope element) {
	focusables := a.focusablesWithin(scope)
	if len(focusables) > 0 {
		a.setFocused(focusables[0])
	}
}

func (a *App) focusablesWithin(scope element) []focusTarget {
	var out []focusTarget
	for _, target := range a.focusables {
		if elementContains(scope, target.element) {
			out = append(out, target)
		}
	}
	return out
}

func (a *App) focusedWithin(scope element) bool {
	return a.focused.element != nil && elementContains(scope, a.focused.element)
}

func (a *App) focusTrapFor(e element) element {
	for cur := e; cur != nil; cur = cur.Base().parent {
		scope, ok := cur.Base().widget.(FocusScope)
		if ok && scope.Trap {
			return cur
		}
	}
	return nil
}

func elementContains(root, child element) bool {
	for cur := child; cur != nil; cur = cur.Base().parent {
		if cur == root {
			return true
		}
	}
	return false
}

func (a *App) setFocusedElement(next element) {
	a.setFocused(focusTarget{element: next, index: elementFocusIndex})
}

func (a *App) setFocused(next focusTarget) {
	if a.focused == next {
		return
	}
	old := a.focused
	a.focused = next
	a.notifyFocusChanged(old)
	a.notifyFocusChanged(next)
}

func (a *App) notifyFocusChanged(target focusTarget) {
	if target.element == nil {
		return
	}
	if a.build.building {
		a.deferFocusNotification(target)
		return
	}
	if f, ok := target.element.Base().widget.(focusWidget); ok && target.index == elementFocusIndex && f.Node != nil && f.Node.onChange != nil {
		f.Node.onChange()
	}
	if ro, ok := ownRenderObject(target.element).(renderFocusHandler); ok {
		index := elementFocusIndex
		if a.focused == target {
			index = target.index
		}
		ro.SetFocusedIndex(index)
	}
	if a.focused == target {
		a.revealFocusedTarget(target)
	}
}

func (a *App) revealFocusedTarget(target focusTarget) {
	ro := findRenderObject(target.element)
	if ro == nil {
		return
	}
	rect := focusedRenderRect(ro, target.index)
	for child, parent := ro, ro.Base().parent; parent != nil; child, parent = parent, parent.Base().parent {
		if op, ok := parent.(ChildOffsetProvider); ok {
			off := op.ChildOffset(child)
			rect.X += off.X
			rect.Y += off.Y
		}
		if scroll, ok := parent.(*renderScrollView); ok {
			scroll.RevealRect(rect)
		}
	}
}

type focusRectProvider interface {
	FocusRect(index int) (Rect, bool)
}

func focusedRenderRect(ro RenderObject, index int) Rect {
	if provider, ok := ro.(focusRectProvider); ok {
		if rect, ok := provider.FocusRect(index); ok {
			return rect
		}
	}
	size := ro.Base().Size()
	return Rect{Width: size.Width, Height: size.Height}
}

func (a *App) deferFocusNotification(target focusTarget) {
	for _, existing := range a.pendingFocus {
		if existing == target {
			return
		}
	}
	a.pendingFocus = append(a.pendingFocus, target)
}

func (a *App) flushFocusNotifications() {
	pending := a.pendingFocus
	a.pendingFocus = nil
	for _, target := range pending {
		if target.element.Base().owner == nil {
			continue
		}
		a.notifyFocusChanged(target)
	}
}

func (a *App) hitPath(pt Point) []element {
	return hitElement(a.build.Root(), pt)
}

func hitElement(e element, pt Point) []element {
	ro := ownRenderObject(e)
	if ro != nil && !pointInSize(pt, ro.Base().Size()) {
		return nil
	}
	var best []element
	children := elementChildren(e)
	if ro, ok := ownRenderObject(e).(interface{ HitTestChildrenReverse() bool }); ok && ro.HitTestChildrenReverse() {
		reverseElements(children)
	}
	for _, child := range children {
		if best != nil {
			break
		}
		if ro, ok := ownRenderObject(e).(interface{ HitTestChild(RenderObject) bool }); ok {
			childRO := findRenderObject(child)
			if childRO != nil && !ro.HitTestChild(childRO) {
				continue
			}
		}
		if path := hitElement(child, childPoint(e, child, pt)); path != nil {
			best = path
		}
	}
	if best != nil {
		return append([]element{e}, best...)
	}
	if ro != nil {
		if h, ok := ro.(interface{ HitTestSelf(Point) bool }); ok && !h.HitTestSelf(pt) {
			return nil
		}
		return []element{e}
	}
	return nil
}

func elementChildren(e element) []element {
	var children []element
	e.VisitChildren(func(child element) {
		children = append(children, child)
	})
	return children
}

func reverseElements(elements []element) {
	for i, j := 0, len(elements)-1; i < j; i, j = i+1, j-1 {
		elements[i], elements[j] = elements[j], elements[i]
	}
}

func childPoint(parent, child element, pt Point) Point {
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

func ownRenderObject(e element) RenderObject {
	if r, ok := e.(interface{ RenderObject() RenderObject }); ok {
		return r.RenderObject()
	}
	return nil
}

// Paint paints the current render tree into p.
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
	// Option configures an App or Run invocation.
	Option  func(*options)
	options struct {
		theme          Theme
		hasTheme       bool
		themeSet       ThemeSet
		hasThemeSet    bool
		profileOverlay bool
		shortcuts      ShortcutMap
	}
)

// WithTheme sets the theme used by built-in widgets.
func WithTheme(theme Theme) Option {
	return func(o *options) {
		o.theme, o.hasTheme = theme, true
		o.themeSet, o.hasThemeSet = ThemeSet{}, false
	}
}

// WithThemeSet sets light and dark themes and switches between them on
// ColorThemeUpdate events.
func WithThemeSet(themeSet ThemeSet) Option {
	return func(o *options) {
		o.themeSet = themeSet
		o.hasThemeSet = true
		o.theme = themeSet.Theme(DarkTheme)
		o.hasTheme = true
	}
}

// WithPalette generates light and dark themes from palette and switches between
// them on ColorThemeUpdate events.
func WithPalette(palette Palette) Option {
	return WithThemeSet(ThemeSetFromPalette(palette))
}

// WithBaseColors generates light and dark themes from base colors and switches
// between them on ColorThemeUpdate events.
func WithBaseColors(base BaseColors) Option {
	return WithThemeSet(ThemeSetFromBaseColors(base))
}

// WithProfileOverlay draws recent UI profiling stats in the top-right corner.
func WithProfileOverlay() Option {
	return func(o *options) { o.profileOverlay = true }
}

// WithShortcuts replaces the default app-level key-to-intent bindings.
//
// Start from DefaultShortcuts when you want to keep the built-in bindings and
// add or change only a few keys.
func WithShortcuts(shortcuts ShortcutMap) Option {
	return func(o *options) { o.shortcuts = cloneShortcuts(shortcuts) }
}
