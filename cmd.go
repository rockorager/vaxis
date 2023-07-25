package vaxis

// cmds is an asynchronous queue, provided as a helper for applications
var cmds = NewQueue[Cmd]()

// Cmd is some long-running task. The Cmd queue is provided as a helper for
// applications to use as a queue running in parallel with the main event loop.
// For example, http requests should happen asynchronously, so an application
// can post a Cmd which some worker thread is polling, does the request, then
// posts a Msg with the result into the main event loop
type Cmd interface{}

// PostCmd posts a Cmd into the Cmd queue
func PostCmd(cmd Cmd) {
	if cmd == nil {
		return
	}
	cmds.Push(cmd)
}

// PollCmd returns the next Cmd. PollCmd blocks until a Cmd is available
func PollCmd() Cmd {
	var c Cmd
	for cmd := range cmds.ch {
		c = cmd
		break
	}
	return c
}

// CmdChannel provides access to a channel of Cmds from the queue
func CmdChannel() chan Cmd {
	return cmds.Chan()
}
