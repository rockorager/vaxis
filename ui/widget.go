package ui

import (
	"fmt"
	"reflect"
)

type (
	// StatelessWidget builds child widgets from configuration and context.
	StatelessWidget interface{ Build(BuildContext) Widget }
	// StatefulWidget creates persistent State for a widget location.
	StatefulWidget interface{ CreateState() State }
	// RenderObjectWidget creates and updates a render object.
	RenderObjectWidget interface {
		CreateRenderObject(BuildContext) RenderObject
		UpdateRenderObject(BuildContext, RenderObject)
	}
)

// ParentDataWidget writes layout data onto its child's render object.
type ParentDataWidget interface {
	Child() Widget
	ApplyParentData(RenderObject)
}

// elementWidget creates a custom element implementation.
type elementWidget interface{ CreateElement() element }

func createElement(w Widget) element {
	if w == nil {
		panic("ui: nil is not a widget")
	}
	if ew, ok := w.(elementWidget); ok {
		return ew.CreateElement()
	}
	kinds := 0
	var e element
	if pw, ok := w.(ParentDataWidget); ok {
		kinds++
		e = newParentDataElement(pw)
	}
	if sw, ok := w.(StatefulWidget); ok {
		kinds++
		e = newStatefulElement(sw)
	}
	if sw, ok := w.(StatelessWidget); ok {
		kinds++
		e = newStatelessElement(sw)
	}
	if rw, ok := w.(RenderObjectWidget); ok {
		kinds++
		e = newRenderObjectElement(rw)
	}
	if kinds == 0 {
		panic(fmt.Sprintf("ui: %T is not a widget", w))
	}
	if kinds > 1 {
		panic(fmt.Sprintf("ui: %T implements multiple widget kinds", w))
	}
	return e
}

func canUpdate(old, next Widget) bool {
	if old == nil || next == nil || reflect.TypeOf(old) != reflect.TypeOf(next) {
		return false
	}
	ok, hasOld := widgetKey(old)
	nk, hasNext := widgetKey(next)
	return hasOld == hasNext && ok == nk
}

func widgetKey(w Widget) (KeyValue, bool) {
	k, ok := w.(Keyed)
	if !ok || k.WidgetKey() == "" {
		return "", false
	}
	return k.WidgetKey(), true
}
