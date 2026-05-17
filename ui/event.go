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

func (c EventContext) Phase() EventPhase         { return c.phase }
func (c EventContext) Quit()                     { c.app.quit = true }
func (c EventContext) SetTitle(string)           {}
func (c EventContext) CopyToClipboard(string)    {}
func (c EventContext) Notify(title, body string) {}
func (c EventContext) FocusNext()                { c.app.focusNext() }
func (c EventContext) FocusPrevious()            { c.app.focusPrevious() }

type EventHandler interface {
	HandleEvent(EventContext, Event) EventResult
}
type VoidCallback func(EventContext)
type EventCallback func(EventContext, Event) EventResult
type KeyCallback func(EventContext, Key) EventResult

type FocusNode struct {
	app     *App
	element Element
}

func (n *FocusNode) RequestFocus() {
	if n != nil && n.app != nil && n.element != nil {
		n.app.focused = n.element
	}
}
func (n *FocusNode) HasFocus() bool { return n != nil && n.app != nil && n.app.focused == n.element }

func (n *FocusNode) attach(app *App, element Element) { n.app, n.element = app, element }
func (n *FocusNode) detach(element Element) {
	if n != nil && n.element == element {
		n.app, n.element = nil, nil
	}
}
