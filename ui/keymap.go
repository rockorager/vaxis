package ui

type Keymap struct {
	Bindings map[string]VoidCallback
	Child    Widget
}

func (w Keymap) CreateElement() Element {
	return &keymapElement{}
}

type keymapElement struct {
	ElementBase
	child Element
}

func (e *keymapElement) Rebuild() {
	e.child = e.UpdateChild(e.child, e.widget.(Keymap).Child, nil)
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
	if keyIsRelease(key) {
		return EventIgnored
	}
	for binding, cb := range e.widget.(Keymap).Bindings {
		if key.MatchString(binding) {
			if cb != nil {
				cb(ctx)
			}
			return EventHandled
		}
	}
	return EventIgnored
}
