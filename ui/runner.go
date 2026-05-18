package ui

import "time"

type Runner struct {
	app       *App
	backend   Backend
	scheduler *FrameScheduler
	done      bool
}

func NewRunner(app *App, backend Backend, scheduler *FrameScheduler) *Runner {
	if scheduler == nil {
		scheduler = NewFrameScheduler(DefaultFrameInterval)
	}
	app.dispatch = backend.Dispatch
	return &Runner{app: app, backend: backend, scheduler: scheduler}
}

func (r *Runner) Start(now time.Time) { r.RequestFrame(now) }
func (r *Runner) Done() bool          { return r.done || r.app.ShouldQuit() }

func (r *Runner) NextFrame() (time.Time, bool) {
	if !r.scheduler.Scheduled() {
		return time.Time{}, false
	}
	return r.scheduler.Due(), true
}

func (r *Runner) RequestFrame(now time.Time) {
	r.app.RequestFrame()
	r.scheduler.Request(now)
}

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

func (r *Runner) HandleFrame(now time.Time) error {
	if !r.app.FrameRequested() {
		r.scheduler.DidFrame(now)
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
	if r.app.FrameRequested() {
		r.scheduler.Request(now)
	}
	return nil
}
