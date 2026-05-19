package ui

import "time"

// Runner connects an App to a Backend and frame scheduler.
type Runner struct {
	app       *App
	backend   Backend
	scheduler *FrameScheduler
	done      bool
}

// NewRunner creates a runner for app and backend.
func NewRunner(app *App, backend Backend, scheduler *FrameScheduler) *Runner {
	if scheduler == nil {
		scheduler = NewFrameScheduler(DefaultFrameInterval)
	}
	app.dispatch = backend.Dispatch
	if b, ok := backend.(interface{ SetTitle(string) }); ok {
		app.setTitle = b.SetTitle
	}
	if b, ok := backend.(interface{ CopyToClipboard(string) }); ok {
		app.copyToClipboard = b.CopyToClipboard
	}
	if b, ok := backend.(interface{ Notify(string, string) }); ok {
		app.notify = b.Notify
	}
	return &Runner{app: app, backend: backend, scheduler: scheduler}
}

// Start schedules the initial frame.
func (r *Runner) Start(now time.Time) {
	r.RequestFrame(now)
}

// Done reports whether the runner should stop.
func (r *Runner) Done() bool {
	return r.done || r.app.ShouldQuit()
}

// NextFrame returns the next scheduled frame time.
func (r *Runner) NextFrame() (time.Time, bool) {
	if !r.scheduler.Scheduled() {
		return time.Time{}, false
	}
	return r.scheduler.Due(), true
}

// RequestFrame asks the scheduler for another frame.
func (r *Runner) RequestFrame(now time.Time) {
	r.app.RequestFrame()
	r.scheduler.Request(now)
}

// HandleEvent dispatches one backend event to the app.
func (r *Runner) HandleEvent(ev Event, now time.Time) {
	if fn, ok := ev.(SyncFunc); ok {
		fn()
		r.RequestFrame(now)
		return
	}
	r.app.Send(ev)
	if _, ok := ev.(Mouse); ok {
		r.backend.SetMouseShape(r.app.MouseShape())
		if r.app.consumeMouseShapeDirty() {
			r.RequestFrame(now)
		}
	}
	if r.app.ShouldQuit() {
		r.done = true
		return
	}
	if _, ok := ev.(Resize); ok {
		r.app.RequestFrame()
	}
	if _, ok := ev.(Redraw); ok {
		r.app.RequestFrame()
	}
	if r.app.FrameRequested() && !r.scheduler.Scheduled() {
		r.scheduler.Request(now)
	}
}

// HandleFrame rebuilds, lays out, paints, and renders one frame if needed.
func (r *Runner) HandleFrame(now time.Time) error {
	r.app.tickAnimations(now)
	activeFrameTicks := r.app.tickFrameCallbacks(now)
	if !r.app.FrameRequested() {
		r.scheduler.DidFrame(now)
		if activeFrameTicks {
			r.scheduler.Request(now)
		}
		return nil
	}
	size := r.backend.Size()
	r.app.Pump(size)
	painter := NewPainter(size)
	r.app.Paint(painter)
	r.backend.SetMouseShape(r.app.MouseShape())
	if err := r.backend.Render(painter); err != nil {
		return err
	}
	r.scheduler.DidFrame(now)
	if r.app.FrameRequested() || r.app.hasActiveAnimations() || activeFrameTicks {
		r.scheduler.Request(now)
	}
	return nil
}
