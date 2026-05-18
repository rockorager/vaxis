package ui

type State interface{ Build(BuildContext) Widget }

type StateBase struct{ element *statefulElement }

func (s *StateBase) SetState(fn func()) {
	if fn == nil {
		panic("ui: SetState called with nil function")
	}
	fn()
	s.MarkNeedsBuild()
}

func (s *StateBase) MarkNeedsBuild() {
	if s.element == nil || s.element.owner == nil {
		panic("ui: MarkNeedsBuild called after Dispose")
	}
	if s.element.owner.building {
		panic("ui: MarkNeedsBuild called during build")
	}
	s.element.MarkNeedsBuild()
}

func (s *StateBase) Context() BuildContext {
	return s.element.Context()
}

func (s *StateBase) Widget() Widget {
	return s.element.widget
}

type stateBaseSetter interface{ setElement(*statefulElement) }

func (s *StateBase) setElement(e *statefulElement) {
	s.element = e
}

type (
	StateInitializer interface{ InitState() }
	StateDisposer    interface{ Dispose() }
	StateUpdater     interface{ DidUpdateWidget(old Widget) }
)

type statefulElement struct {
	ElementBase
	state State
	child Element
}

func newStatefulElement(w StatefulWidget) Element {
	return &statefulElement{}
}

func (e *statefulElement) update(old Widget) {
	if u, ok := e.state.(StateUpdater); ok {
		u.DidUpdateWidget(old)
	}
}

func (e *statefulElement) Rebuild() {
	if e.state == nil {
		e.state = e.widget.(StatefulWidget).CreateState()
		if setter, ok := e.state.(stateBaseSetter); ok {
			setter.setElement(e)
		}
		if init, ok := e.state.(StateInitializer); ok {
			init.InitState()
		}
	}
	e.child = e.UpdateChild(e.child, e.state.Build(e.Context()), nil)
}

func (e *statefulElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *statefulElement) HandleEvent(ctx EventContext, ev Event) EventResult {
	h, ok := e.state.(EventHandler)
	if !ok {
		return EventIgnored
	}
	return h.HandleEvent(ctx, ev)
}

func (e *statefulElement) MouseShape(ctx EventContext, mouse Mouse) MouseShape {
	h, ok := e.state.(MouseShapeHandler)
	if !ok {
		return MouseShapeDefault
	}
	return h.MouseShape(ctx, mouse)
}

func (e *statefulElement) dispose() {
	if d, ok := e.state.(StateDisposer); ok {
		d.Dispose()
	}
	if setter, ok := e.state.(stateBaseSetter); ok {
		setter.setElement(nil)
	}
}

type statelessElement struct {
	ElementBase
	child Element
}

func newStatelessElement(w StatelessWidget) Element {
	return &statelessElement{}
}

func (e *statelessElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(StatelessWidget).Build(e.Context()), nil)
}

func (e *statelessElement) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}
