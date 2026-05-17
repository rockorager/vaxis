package ui

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
