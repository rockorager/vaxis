package ui

type App struct {
	build      *BuildOwner
	rootRO     RenderObject
	size       Size
	focusables []Element
	focused    Element
	quit       bool
}

func NewApp(root Widget, opts ...Option) *App {
	owner := NewBuildOwner()
	app := &App{build: owner}
	owner.app = app
	owner.Mount(Provider[Theme]{Value: DefaultTheme(), ChildWidget: root})
	return app
}

func (a *App) UpdateRoot(root Widget) {
	a.build.UpdateRoot(Provider[Theme]{Value: DefaultTheme(), ChildWidget: root})
}
func (a *App) Send(ev Event) { a.dispatchEvent(ev) }

func (a *App) Pump(size Size) {
	a.size = size
	a.build.BuildScope()
	a.rootRO = findRenderObject(a.build.Root())
	if a.rootRO != nil {
		a.rootRO.Layout(LayoutContext{}, Tight(size))
	}
}

func (a *App) dispatchEvent(ev Event) EventResult {
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
	path := a.pathTo(target)
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
		a.focused = e
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
	a.focused = a.focusables[idx]
}

func (a *App) Paint(p *Painter) {
	if a.rootRO != nil {
		a.rootRO.Paint(p, Offset{})
	}
}

type Option func(*options)
type options struct{}
