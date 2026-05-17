package ui

import (
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

func Run(root Widget, opts ...Option) error {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		return err
	}
	backend := vaxisBackend{vx: vx}
	defer backend.Close()
	return runWithBackend(root, backend, opts...)
}

func runWithBackend(root Widget, backend Backend, opts ...Option) error {
	runner := NewRunner(NewApp(root, opts...), backend, NewFrameScheduler(DefaultFrameInterval))
	frameTimer := time.NewTimer(time.Hour)
	frameTimer.Stop()
	schedule := func() {
		due, ok := runner.NextFrame()
		if !ok {
			return
		}
		frameTimer.Reset(time.Until(due))
	}
	runner.Start(time.Now())
	schedule()
	events := backend.Events()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			runner.HandleEvent(ev, time.Now())
			if runner.Done() {
				return nil
			}
			schedule()
		case <-frameTimer.C:
			if err := runner.HandleFrame(time.Now()); err != nil {
				return err
			}
			schedule()
		}
	}
}
