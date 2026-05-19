// Package ui provides a Flutter-inspired widget, layout, and painting layer for
// terminal applications built with Vaxis.
//
// Most applications start with Run:
//
//	err := ui.Run(ui.Text{Value: "hello"})
//
// Widgets are immutable descriptions of UI. Stateful widgets create State
// values, call StateBase.SetState to schedule rebuilds, and return more widgets
// from Build. Built-in text inputs are controlled widgets: the Value field is
// the source of truth, and OnChanged is responsible for storing the next value
// in application state.
//
// Layout flows through render objects using Constraints and Size in terminal
// cells. A widget that needs custom measurement or painting can implement
// RenderObjectWidget and produce a RenderObject; ordinary applications usually
// compose the built-in widgets instead.
//
// Run uses the default Vaxis backend. Tests and integrations can use App,
// Runner, and Backend directly to drive events and frames without owning the
// terminal event loop.
package ui
