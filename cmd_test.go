package vaxis_test

import "git.sr.ht/~rockorager/vaxis"

type Worker struct{}

func NewWorker() *Worker {
	return &Worker{}
}

func (w *Worker) HandleCmd(cmd vaxis.Cmd) {
	// Do something
}

func ExamplePollCmd() {
	worker := NewWorker()
	for {
		cmd := vaxis.PollCmd()
		go worker.HandleCmd(cmd)
	}
}

func ExampleCmdChannel() {
	worker := NewWorker()
	for cmd := range vaxis.CmdChannel() {
		go worker.HandleCmd(cmd)
	}
}
