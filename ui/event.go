package ui

// EventPhase identifies where an event is in capture, target, and bubble dispatch.
type EventPhase int

const (
	// CapturePhase is delivered from the root toward the target.
	CapturePhase EventPhase = iota
	// TargetPhase is delivered to the target element.
	TargetPhase
	// BubblePhase is delivered from the target's parent back toward the root.
	BubblePhase
)

// EventResult controls whether event propagation continues.
type EventResult int

const (
	// EventIgnored allows event propagation to continue.
	EventIgnored EventResult = iota
	// EventHandled stops event propagation.
	EventHandled
)

// EventContext exposes the current event phase and runtime side effects.
type EventContext struct {
	app     *App
	phase   EventPhase
	element element
	target  element
}

// FractionalMousePoint is a mouse location in terminal cell coordinates.
type FractionalMousePoint struct {
	// Col is the horizontal cell coordinate. Values may include a fractional
	// offset within the cell when pixel mouse reports are available.
	Col float64
	// Row is the vertical cell coordinate. Values may include a fractional
	// offset within the cell when pixel mouse reports are available.
	Row float64
}

// Runtime schedules work back onto the UI event loop.
type Runtime interface{ Dispatch(func()) }

type appRuntime struct{ app *App }

func (r appRuntime) Dispatch(fn func()) {
	if fn == nil {
		panic("ui: Dispatch called with nil function")
	}
	r.app.dispatch(fn)
}

// Phase returns the current dispatch phase.
func (c EventContext) Phase() EventPhase {
	return c.phase
}

// FractionalMousePoint converts mouse to fractional cell coordinates when
// pixel mouse reports and terminal pixel dimensions are available.
func (c EventContext) FractionalMousePoint(mouse Mouse) FractionalMousePoint {
	if c.app == nil {
		return FractionalMousePoint{Col: float64(mouse.Col), Row: float64(mouse.Row)}
	}
	size := c.app.window
	if mouse.XPixel <= 0 || mouse.YPixel <= 0 || size.XPixel <= 0 || size.YPixel <= 0 || size.Cols <= 0 || size.Rows <= 0 {
		return FractionalMousePoint{Col: float64(mouse.Col), Row: float64(mouse.Row)}
	}
	return FractionalMousePoint{
		Col: float64(mouse.XPixel) * float64(size.Cols) / float64(size.XPixel),
		Row: float64(mouse.YPixel) * float64(size.Rows) / float64(size.YPixel),
	}
}

// Runtime returns a dispatcher for scheduling work on the UI event loop.
func (c EventContext) Runtime() Runtime {
	return appRuntime{app: c.app}
}

// Invoke runs the nearest action for intent, resolving from the current event
// target. Default actions are used only when no regular action is found.
func (c EventContext) Invoke(intent Intent) EventResult {
	if intent == nil {
		return EventIgnored
	}
	target := c.target
	if target == nil {
		target = c.element
	}
	intentType := intent.IntentType()
	var fallback ActionFunc
	var fallbackOK bool
	for e := target; e != nil; e = e.Base().parent {
		if actions, ok := e.(actionProvider); ok {
			action, ok := actions.action(intentType)
			if ok {
				return runAction(action, c, intent)
			}
		}
		if !fallbackOK {
			defaults, ok := e.(defaultActionProvider)
			if ok {
				action, ok := defaults.defaultAction(intentType)
				if ok {
					fallback = action
					fallbackOK = true
				}
			}
		}
	}
	if fallbackOK {
		return runAction(fallback, c, intent)
	}
	return EventIgnored
}

func runAction(action ActionFunc, ctx EventContext, intent Intent) EventResult {
	if action == nil {
		return EventHandled
	}
	return action(ctx, intent)
}

// Quit requests that the current runner stop.
func (c EventContext) Quit() {
	c.app.quit = true
}

// SetTitle asks the backend to set the terminal title.
func (c EventContext) SetTitle(title string) {
	c.app.setTitle(title)
}

// Copy asks the backend to place text on the clipboard.
func (c EventContext) Copy(text string) {
	c.app.copyToClipboard(text)
}

// Notify asks the backend to display a notification.
func (c EventContext) Notify(title, body string) {
	c.app.notify(title, body)
}

// CopyToClipboard asks the backend to place text on the clipboard.
func (c EventContext) CopyToClipboard(text string) {
	c.Copy(text)
}

// FocusNext moves focus to the next focusable widget.
func (c EventContext) FocusNext() {
	c.app.focusNext()
}

// FocusPrevious moves focus to the previous focusable widget.
func (c EventContext) FocusPrevious() {
	c.app.focusPrevious()
}

// SetMouseShape requests a mouse cursor shape for the current pointer location.
func (c EventContext) SetMouseShape(shape MouseShape) {
	c.app.setMouseShape(shape)
}

// ProfileOverlay reports whether the profiling overlay is visible.
func (c EventContext) ProfileOverlay() bool {
	return c.app.ProfileOverlay()
}

// SetProfileOverlay shows or hides the profiling overlay.
func (c EventContext) SetProfileOverlay(visible bool) {
	c.app.SetProfileOverlay(visible)
}

// ToggleProfileOverlay toggles the profiling overlay and returns its new state.
func (c EventContext) ToggleProfileOverlay() bool {
	return c.app.ToggleProfileOverlay()
}

// EventHandler receives events during capture, target, or bubble dispatch.
type EventHandler interface {
	HandleEvent(EventContext, Event) EventResult
}

// MouseShapeHandler chooses the mouse cursor shape for a hovered element.
type MouseShapeHandler interface {
	MouseShape(EventContext, Mouse) MouseShape
}
type (
	hoverExit struct{}
	// VoidCallback handles an action with event context.
	VoidCallback func(EventContext)
	// EventCallback handles an event and controls propagation.
	EventCallback func(EventContext, Event) EventResult
	// KeyCallback handles a key event and controls propagation.
	KeyCallback func(EventContext, Key) EventResult
)

// FocusNode controls and observes focus for a Focus widget.
type FocusNode struct {
	app      *App
	element  element
	onChange func()
}

// RequestFocus moves focus to this node if it is attached.
func (n *FocusNode) RequestFocus() {
	if n != nil && n.app != nil && n.element != nil {
		n.app.setFocusedElement(n.element)
	}
}

// HasFocus reports whether this node is currently focused.
func (n *FocusNode) HasFocus() bool {
	return n != nil && n.app != nil && n.app.focused == (focusTarget{element: n.element, index: elementFocusIndex})
}

func (n *FocusNode) attach(app *App, element element) {
	n.app, n.element = app, element
}

func (n *FocusNode) detach(element element) {
	if n != nil && n.element == element {
		n.app, n.element = nil, nil
	}
}
