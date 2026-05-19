package ui

import "reflect"

// Provider makes a typed value available to descendant widgets.
type Provider[T any] struct {
	// Value is the value exposed to descendants.
	Value T
	// ChildWidget is the subtree that can depend on Value.
	ChildWidget Widget
	// ShouldNotify controls whether dependents rebuild after Value changes.
	ShouldNotify func(old, next T) bool
}

func (p Provider[T]) Child() Widget {
	return p.ChildWidget
}

func (p Provider[T]) CreateElement() Element {
	return &providerElement[T]{}
}

type providerElement[T any] struct {
	ElementBase
	child        Element
	dependents   map[Element]struct{}
	value        T
	shouldNotify func(T, T) bool
}

func (e *providerElement[T]) Rebuild() {
	w := e.widget.(Provider[T])
	if e.dependents == nil {
		e.dependents = make(map[Element]struct{})
	}
	if e.child != nil {
		notify := true
		if e.shouldNotify != nil {
			notify = e.shouldNotify(e.value, w.Value)
		}
		e.value = w.Value
		e.shouldNotify = w.ShouldNotify
		if !notify {
			return
		}
		if notify {
			for dep := range e.dependents {
				dep.Base().MarkNeedsBuild()
			}
		}
	}
	e.value = w.Value
	e.shouldNotify = w.ShouldNotify
	e.child = e.UpdateChild(e.child, w.ChildWidget, nil)
}

func (e *providerElement[T]) VisitChildren(fn func(Element)) {
	if e.child != nil {
		fn(e.child)
	}
}

// Depend returns the nearest provided value of type T and subscribes ctx to updates.
func Depend[T any](ctx BuildContext) (T, bool) {
	for e := ctx.element; e != nil; e = e.Base().parent {
		if p, ok := e.(*providerElement[T]); ok {
			p.dependents[ctx.element] = struct{}{}
			return p.value, true
		}
	}
	var zero T
	return zero, false
}

// MustDepend returns the nearest provided value of type T or panics.
func MustDepend[T any](ctx BuildContext) T {
	v, ok := Depend[T](ctx)
	if !ok {
		panic("ui: missing provider for " + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return v
}
