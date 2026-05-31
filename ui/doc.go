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
// Build methods and lazy row builders should stay cheap. Cache expensive
// derived data, such as syntax-highlighted lines or parsed document structure,
// in State and invalidate that cache from StateUpdater.DidUpdateWidget or when
// theme-dependent inputs change. Builders may run often during scrolling,
// layout correction, and resize handling, so they should usually select from
// already-prepared data rather than recomputing whole-document results.
//
// Layout flows through render objects using Constraints and Size in terminal
// cells. A widget that needs custom measurement or painting can implement
// RenderObjectWidget and produce a RenderObject; ordinary applications usually
// compose the built-in widgets instead.
//
// Run uses the default Vaxis backend. Tests and integrations can use App,
// Runner, and Backend directly to drive events and frames without owning the
// terminal event loop.
//
// Use WithPrimaryScreen or WithDynamicPrimaryScreen when the application should
// keep terminal scrollback instead of entering the alternate screen. In
// primary-screen mode the root widget is rendered into a live region, and event
// handlers can write normal terminal output before that region:
//
//	err := ui.Run(root, ui.WithDynamicPrimaryScreen())
//
//	button := ui.Button{Label: "log", OnPressed: func(ctx ui.EventContext) {
//		ctx.AppendWidget(ui.Text{Value: "clicked"})
//	}}
//
// Append, AppendString, AppendWriter, AppendText, AppendTextLn, and AppendWidget are
// available through EventContext when the backend supports primary-screen
// appends. AppendText writes styled inline spans without layout, so text can
// reflow naturally in terminal scrollback. AppendWidget appends a rendered
// snapshot, so soft-wrapped widget text becomes hard line breaks at the current
// terminal width.
package ui
