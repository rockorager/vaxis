package ui

type EventPhase int

const (
	CapturePhase EventPhase = iota
	TargetPhase
	BubblePhase
)

type EventResult int

const (
	EventIgnored EventResult = iota
	EventHandled
)

type EventContext struct {
	app   *App
	phase EventPhase
}

type Runtime interface{ Dispatch(func()) }

type appRuntime struct{ app *App }

func (r appRuntime) Dispatch(fn func()) {
	if fn == nil {
		panic("ui: Dispatch called with nil function")
	}
	r.app.dispatch(fn)
}

func (c EventContext) Phase() EventPhase {
	return c.phase
}

func (c EventContext) Runtime() Runtime {
	return appRuntime{app: c.app}
}

func (c EventContext) Quit() {
	c.app.quit = true
}

func (c EventContext) SetTitle(string) {
}

func (c EventContext) CopyToClipboard(string) {
}

func (c EventContext) Notify(title, body string) {
}

func (c EventContext) FocusNext() {
	c.app.focusNext()
}

func (c EventContext) FocusPrevious() {
	c.app.focusPrevious()
}

func (c EventContext) SetMouseShape(shape MouseShape) {
	c.app.setMouseShape(shape)
}

type EventHandler interface {
	HandleEvent(EventContext, Event) EventResult
}
type MouseShapeHandler interface {
	MouseShape(EventContext, Mouse) MouseShape
}
type (
	hoverExit     struct{}
	VoidCallback  func(EventContext)
	EventCallback func(EventContext, Event) EventResult
	KeyCallback   func(EventContext, Key) EventResult
)

type FocusNode struct {
	app      *App
	element  Element
	onChange func()
}

func (n *FocusNode) RequestFocus() {
	if n != nil && n.app != nil && n.element != nil {
		n.app.setFocused(n.element)
	}
}

func (n *FocusNode) HasFocus() bool {
	return n != nil && n.app != nil && n.app.focused == n.element
}

func (n *FocusNode) attach(app *App, element Element) {
	n.app, n.element = app, element
}

func (n *FocusNode) detach(element Element) {
	if n != nil && n.element == element {
		n.app, n.element = nil, nil
	}
}
