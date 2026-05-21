package ui

// ActionFunc handles an intent.
type ActionFunc func(EventContext, Intent) EventResult

type actionProvider interface {
	action(IntentType) (ActionFunc, bool)
}

type defaultActionProvider interface {
	defaultAction(IntentType) (ActionFunc, bool)
}

// Actions provides intent handlers for its subtree.
type Actions struct {
	// Bindings maps intent types to handlers.
	Bindings map[IntentType]ActionFunc
	// Child is the subtree that can invoke these actions.
	Child Widget
}

func (w Actions) CreateElement() element {
	return &actionsElement{}
}

type actionsElement struct {
	elementBase
	child element
}

func (e *actionsElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(Actions).Child, nil)
}

func (e *actionsElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *actionsElement) action(intent IntentType) (ActionFunc, bool) {
	action, ok := e.widget.(Actions).Bindings[intent]
	return action, ok
}

// DefaultActions provides fallback intent handlers for its subtree.
type DefaultActions struct {
	// Bindings maps intent types to fallback handlers.
	Bindings map[IntentType]ActionFunc
	// Child is the subtree that can invoke these actions.
	Child Widget
}

func (w DefaultActions) CreateElement() element {
	return &defaultActionsElement{}
}

type defaultActionsElement struct {
	elementBase
	child element
}

func (e *defaultActionsElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(DefaultActions).Child, nil)
}

func (e *defaultActionsElement) VisitChildren(fn func(element)) {
	if e.child != nil {
		fn(e.child)
	}
}

func (e *defaultActionsElement) defaultAction(intent IntentType) (ActionFunc, bool) {
	action, ok := e.widget.(DefaultActions).Bindings[intent]
	return action, ok
}
