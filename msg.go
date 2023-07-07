package vaxis

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

func PostMsg(msg Msg) {
	if msg == nil {
		return
	}
	msgs.push(msg)
}

// PollMsg returns the next Msg. When a QuitMsg is received, all input processing
// will cease.
func PollMsg() Msg {
	var m Msg
	for msg := range msgs.ch {
		switch msg := msg.(type) {
		case QuitMsg:
			close(chQuit)
			return msg
		case Resize:
			stdScreen.resize(msg.Cols, msg.Rows)
			lastRender.resize(msg.Cols, msg.Rows)
		}
		m = msg
		break
	}
	return m
}

// Msgs provides access to the channel of Msgs
func Msgs() chan Msg {
	return msgs.Chan()
}

type FuncMsg struct {
	Func func()
}

// InitMsg will always be the first Msg delivered
type InitMsg struct{}

// QuitMsg is delivered whenever the application is about to close
type QuitMsg struct{}

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

type Visible bool
