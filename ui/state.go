package ui

// State stores mutable widget state and builds a widget subtree.
type State interface{ Build(BuildContext) Widget }

// StateBase provides lifecycle helpers for State implementations.
type StateBase struct {
	element    *statefulElement
	animations []*AnimationController
}

// SetState applies fn and schedules this state to rebuild.
func (s *StateBase) SetState(fn func()) {
	if fn == nil {
		panic("ui: SetState called with nil function")
	}
	fn()
	s.MarkNeedsBuild()
}

// MarkNeedsBuild schedules this state to rebuild.
func (s *StateBase) MarkNeedsBuild() {
	if s.element == nil || s.element.owner == nil {
		panic("ui: MarkNeedsBuild called after Dispose")
	}
	if s.element.owner.building {
		panic("ui: MarkNeedsBuild called during build")
	}
	s.element.MarkNeedsBuild()
}

// Context returns the build context for this state.
func (s *StateBase) Context() BuildContext {
	return s.element.Context()
}

// Widget returns the current widget configuration for this state.
func (s *StateBase) Widget() Widget {
	return s.element.widget
}

// NewAnimation creates an animation controller owned by this state.
func (s *StateBase) NewAnimation(opts AnimationOptions) *AnimationController {
	if s.element == nil || s.element.owner == nil {
		panic("ui: NewAnimation called after Dispose")
	}
	curve := opts.Curve
	if curve == nil {
		curve = Linear
	}
	controller := &AnimationController{
		owner:    s,
		duration: opts.Duration,
		curve:    curve,
	}
	s.animations = append(s.animations, controller)
	return controller
}

type stateBaseSetter interface{ setElement(*statefulElement) }

func (s *StateBase) setElement(e *statefulElement) {
	s.element = e
}

type stateBaseAnimationDisposer interface{ disposeAnimations() }

func (s *StateBase) disposeAnimations() {
	for _, controller := range s.animations {
		controller.dispose()
	}
	s.animations = nil
}

type (
	// StateInitializer is implemented by State values that need mount-time initialization.
	StateInitializer interface{ InitState() }
	// StateDisposer is implemented by State values that need unmount cleanup.
	StateDisposer interface{ Dispose() }
	// StateUpdater is implemented by State values that observe compatible widget updates.
	StateUpdater interface{ DidUpdateWidget(old Widget) }
)

type statefulElement struct {
	elementBase
	state State
	child element
}

func newStatefulElement(w StatefulWidget) element {
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

func (e *statefulElement) VisitChildren(fn func(element)) {
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
	if d, ok := e.state.(stateBaseAnimationDisposer); ok {
		d.disposeAnimations()
	}
	if setter, ok := e.state.(stateBaseSetter); ok {
		setter.setElement(nil)
	}
}

type statelessElement struct {
	elementBase
	child element
}

func newStatelessElement(w StatelessWidget) element {
	return &statelessElement{}
}

func (e *statelessElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(StatelessWidget).Build(e.Context()), nil)
}

func (e *statelessElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}
