package ui

import (
	"context"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

// Run creates a vaxis-backed app for root and blocks until it exits.
func Run(root Widget, opts ...Option) error {
	vx, err := vaxis.New(vaxis.Options{EnableSGRPixels: true})
	if err != nil {
		return err
	}
	backend := vaxisBackend{vx: vx}
	defer func() { _ = backend.Close() }()
	return runWithBackend(root, backend, opts...)
}

func runWithBackend(root Widget, backend Backend, opts ...Option) error {
	options := options{theme: DefaultTheme()}
	for _, opt := range opts {
		opt(&options)
	}
	if !options.hasTheme {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		options.theme = themeFromTerminal(ctx, backendColorQuerier{backend: backend})
		cancel()
	}
	opts = append(append([]Option{}, opts...), WithTheme(options.theme))
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
			if resize, ok := ev.(Resize); ok {
				if backend, ok := backend.(interface{ Resize(Resize) }); ok {
					backend.Resize(resize)
				}
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
