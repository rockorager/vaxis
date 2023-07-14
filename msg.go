package vaxis

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

// PostMsg posts a Msg into the main event loop
func PostMsg(msg Msg) {
	if msg == nil {
		return
	}
	msgs.push(msg)
}

// PollMsg returns the next Msg
func PollMsg() Msg {
	var m Msg
	for msg := range msgs.ch {
		switch msg := msg.(type) {
		case Resize:
			stdScreen.resize(msg.Cols, msg.Rows)
			lastRender.resize(msg.Cols, msg.Rows)
		}
		m = msg
		break
	}
	return m
}

// MsgChannel provides access to the channel of MsgChannel
func MsgChannel() chan Msg {
	return msgs.Chan()
}

// FuncMsg is a Msg which calls Func from the main event loop
type FuncMsg struct {
	Func func()
}

// InitMsg will always be the first Msg delivered
type InitMsg struct{}

// Resize is delivered whenever a window size change is detected (likely via
// SIGWINCH)
type Resize struct {
	Cols   int
	Rows   int
	XPixel int
	YPixel int
}

// PasteMsg is delivered when a bracketed paste was detected. The value of
// PasteMsg if the pasted content
type PasteMsg string

// SendMsg sends a message to a given model from the main thread
type SendMsg struct {
	Msg   Msg
	Model Model
}

// DrawModelMsg draws the provided model with the provided window. It doesn't call
// draw on the primary model.
type DrawModelMsg struct {
	Model  Model
	Window Window
}

// Visible is a Msg which tells any given widget it's visibility state. This is
// used by some provided widgets, and also provided as a helper
type Visible bool

// FocusIn is sent when the terminal has gained focus
type FocusIn struct{}

// FocusOut is sent when the terminal has lost focus
type FocusOut struct{}
