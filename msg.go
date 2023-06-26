package rtk

// Msg is a passable message conveying some event, data, or just a friendly
// hello
type Msg interface{}

func PostMsg(msg Msg) {
	msgs.push(msg)
}

// Posts a function to be run from the main thread
func PostFunc(fn func()) {
	PostMsg(funcMsg{
		fn: fn,
	})
}

type funcMsg struct {
	fn func()
}

// Init will always be the first Msg delivered
type Init struct{}

// QuitMsg is delivered whenever the application is about to close
type QuitMsg struct{}

// Resize is delivered whenever a window size change is detected (likely via
// SIGWINCH)
type Resize struct {
	Cols int
	Rows int
}

// Paste is delivered when a bracketed paste was detected. The value of
// Paste if the pasted content
type Paste string

// SendMsg sends a Msg directly to a Model
func SendMsg(msg Msg, model Model) {
	PostMsg(sendMsg{
		msg:   msg,
		model: model,
	})
}

type sendMsg struct {
	msg   Msg
	model Model
}

// PartialDraw draws the provided model to the provided surface. It doesn't call
// draw on the primary model.
func PartialDraw(model Model, window Window) {
	PostMsg(partialDrawMsg{
		model:  model,
		window: window,
	})
}

type partialDrawMsg struct {
	model  Model
	window Window
}
