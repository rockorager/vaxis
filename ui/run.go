package ui

import (
	"context"
	"time"

	"go.rockorager.dev/vaxis"
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
	terminalTheme := !options.hasTheme
	if !options.hasTheme {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		options.theme = themeFromTerminal(ctx, backendColorQuerier{backend: backend})
		cancel()
	}
	appOpts := append([]Option{}, opts...)
	if terminalTheme {
		appOpts = append(appOpts, WithTheme(options.theme))
	}
	app := NewApp(root, appOpts...)
	runner := NewRunner(app, backend, NewFrameScheduler(DefaultFrameInterval))
	updateTerminalTheme := func(ev Event) {
		if _, ok := ev.(ColorThemeUpdate); !ok || !terminalTheme {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		app.SetTheme(themeFromTerminal(ctx, backendColorQuerier{backend: backend}))
		cancel()
	}
	submitDebugEvent := func(ev Event) {
		if resize, ok := ev.(Resize); ok {
			if backend, ok := backend.(interface{ Resize(Resize) }); ok {
				backend.Resize(resize)
			}
		}
		updateTerminalTheme(ev)
		runner.HandleEvent(ev, time.Now())
	}
	stopDebug, err := startDebugServer(app, app.dispatch, submitDebugEvent, runner.debugRenderedSnapshot, func() (string, bool) {
		if runner.lastFrame == nil {
			return "", false
		}
		return debugRenderedText(runner.lastFrame), true
	}, runner.debugProfileSnapshot)
	if err != nil {
		return err
	}
	defer stopDebug()
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
			updateTerminalTheme(ev)
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
