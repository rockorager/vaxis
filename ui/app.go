package ui

type App struct {
	build      *BuildOwner
	rootRO     RenderObject
	size       Size
	focusables []Element
	focused    Element
	quit       bool
	theme      Theme
}

func NewApp(root Widget, opts ...Option) *App {
	owner := NewBuildOwner()
	options := options{theme: DefaultTheme()}
	for _, opt := range opts {
		opt(&options)
	}
	app := &App{build: owner, theme: options.theme}
	owner.app = app
	owner.Mount(Provider[Theme]{Value: app.theme, ChildWidget: root})
	return app
}

func (a *App) UpdateRoot(root Widget) {
	a.build.UpdateRoot(Provider[Theme]{Value: a.theme, ChildWidget: root})
}
func (a *App) Send(ev Event)    { a.dispatchEvent(ev) }
func (a *App) ShouldQuit() bool { return a.quit }

func (a *App) Pump(size Size) {
	a.size = size
	a.build.BuildScope()
	a.rootRO = findRenderObject(a.build.Root())
	if a.rootRO != nil {
		a.rootRO.Layout(LayoutContext{}, Tight(size))
	}
}

func (a *App) dispatchEvent(ev Event) EventResult {
	if mouse, ok := ev.(Mouse); ok {
		path := a.hitPath(Point{X: mouse.Col, Y: mouse.Row})
		if len(path) > 0 {
			return a.dispatchPath(path, ev)
		}
	}
	if key, ok := ev.(Key); ok {
		if key.MatchString("Tab") {
			a.focusNext()
			return EventHandled
		}
		if key.MatchString("Shift+Tab") {
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

func (a *App) dispatchPath(path []Element, ev Event) EventResult {
	for i := 0; i < len(path)-1; i++ {
		if a.handle(path[i], CapturePhase, ev) == EventHandled {
			return EventHandled
		}
	}
	if len(path) > 0 && a.handle(path[len(path)-1], TargetPhase, ev) == EventHandled {
		return EventHandled
	}
	for i := len(path) - 2; i >= 0; i-- {
		if a.handle(path[i], BubblePhase, ev) == EventHandled {
			return EventHandled
		}
	}
	return EventIgnored
}

func (a *App) handle(e Element, phase EventPhase, ev Event) EventResult {
	h, ok := e.(EventHandler)
	if !ok {
		return EventIgnored
	}
	return h.HandleEvent(EventContext{app: a, phase: phase}, ev)
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
			a.focused = a.focusables[0]
		}
	}
}

func (a *App) focusNext()     { a.moveFocus(1) }
func (a *App) focusPrevious() { a.moveFocus(-1) }
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
	old := a.focused
	a.focused = next
	a.notifyFocusChanged(old)
	a.notifyFocusChanged(next)
}

func (a *App) notifyFocusChanged(e Element) {
	if e == nil {
		return
	}
	if f, ok := e.Base().widget.(FocusWidget); ok && f.Node != nil && f.Node.onChange != nil {
		f.Node.onChange()
	}
}

func (a *App) hitPath(pt Point) []Element { return hitElement(a.build.Root(), pt) }

func hitElement(e Element, pt Point) []Element {
	ro := findRenderObject(e)
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
	pro := findRenderObject(parent)
	if pro == nil {
		return pt
	}
	switch r := pro.(type) {
	case *RenderPadding:
		return Point{X: pt.X - r.Insets.Left, Y: pt.Y - r.Insets.Top}
	case *RenderCenter:
		return Point{X: pt.X - r.offset.X, Y: pt.Y - r.offset.Y}
	case *RenderFlex:
		if ro := findRenderObject(child); ro != nil {
			pd, _ := ro.Base().ParentData().(FlexParentData)
			return Point{X: pt.X - pd.Offset.X, Y: pt.Y - pd.Offset.Y}
		}
	}
	return pt
}

func pointInSize(pt Point, size Size) bool {
	return pt.X >= 0 && pt.Y >= 0 && pt.X < size.Width && pt.Y < size.Height
}

func (a *App) Paint(p *Painter) {
	if a.rootRO != nil {
		a.rootRO.Paint(p, Offset{})
	}
}

type Option func(*options)
type options struct{ theme Theme }

func WithTheme(theme Theme) Option { return func(o *options) { o.theme = theme } }
