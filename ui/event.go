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

type EventContext struct{ phase EventPhase }

func (c EventContext) Phase() EventPhase         { return c.phase }
func (c EventContext) Quit()                     {}
func (c EventContext) SetTitle(string)           {}
func (c EventContext) CopyToClipboard(string)    {}
func (c EventContext) Notify(title, body string) {}
func (c EventContext) FocusNext()                {}
func (c EventContext) FocusPrevious()            {}

type EventHandler interface {
	HandleEvent(EventContext, Event) EventResult
}
type VoidCallback func(EventContext)
type EventCallback func(EventContext, Event) EventResult
type KeyCallback func(EventContext, Key) EventResult

type FocusNode struct{}

func (n *FocusNode) RequestFocus()  {}
func (n *FocusNode) HasFocus() bool { return false }
