package ui

import "time"

// Runner connects an App to a Backend and frame scheduler.
type Runner struct {
	app       *App
	backend   Backend
	scheduler *FrameScheduler
	done      bool
	lastFrame *Painter
	profile   *profileStore
	options   options
}

// NewRunner creates a runner for app and backend.
func NewRunner(app *App, backend Backend, scheduler *FrameScheduler) *Runner {
	if scheduler == nil {
		scheduler = NewFrameScheduler(DefaultFrameInterval)
	}
	app.dispatch = backend.Dispatch
	if b, ok := backend.(PrimaryScreenAppender); ok {
		app.append = b.Append
		app.appendString = b.AppendString
		app.appendWriter = b.AppendWriter
	}
	if b, ok := backend.(interface{ SetTitle(string) }); ok {
		app.setTitle = b.SetTitle
	}
	if b, ok := backend.(interface{ CopyToClipboard(string) }); ok {
		app.copyToClipboard = b.CopyToClipboard
	}
	if b, ok := backend.(interface{ Notify(string, string) }); ok {
		app.notify = b.Notify
	}
	return &Runner{app: app, backend: backend, scheduler: scheduler, profile: &profileStore{}, options: app.options}
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
	var metric profileMetric
	var profiled bool
	switch ev.(type) {
	case Key:
		metric, profiled = profileKey, true
	case Mouse:
		metric, profiled = profileMouse, true
	}
	start := time.Now()
	defer func() {
		if profiled {
			r.profile.record(metric, time.Since(start))
		}
	}()
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
	frameStart := time.Now()
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
	if r.options.dynamicPrimary {
		terminal := Resize{Cols: size.Width, Rows: size.Height}
		if b, ok := r.backend.(terminalSizer); ok {
			terminal = b.TerminalSize()
		}
		regionHeight := r.app.preferredHeight(terminal.Cols, terminal.Rows)
		if b, ok := r.backend.(PrimaryScreenRegionSizer); ok {
			b.SetPrimaryScreenRegionHeight(regionHeight)
		}
		size = Size{Width: terminal.Cols, Height: regionHeight}
	}
	build, layout := r.app.pumpProfiled(size)
	painter := NewPainter(size)
	start := time.Now()
	r.app.Paint(painter)
	paint := time.Since(start)
	if r.app.profileOverlay {
		drawProfileOverlay(painter, r.profile.snapshot())
	}
	r.backend.SetMouseShape(r.app.MouseShape())
	start = time.Now()
	if err := r.backend.Render(painter); err != nil {
		return err
	}
	render := time.Since(start)
	r.profile.recordFrame(frameProfileTimings{
		build:  build,
		layout: layout,
		paint:  paint,
		render: render,
		frame:  time.Since(frameStart),
	})
	r.lastFrame = painter.clone()
	r.scheduler.DidFrame(now)
	if r.app.FrameRequested() || r.app.hasActiveAnimations() || activeFrameTicks {
		r.scheduler.Request(now)
	}
	return nil
}

func (r *Runner) debugRenderedSnapshot() (DebugRenderedSnapshot, bool) {
	if r.lastFrame == nil {
		return DebugRenderedSnapshot{}, false
	}
	return debugRenderedSnapshot(r.lastFrame), true
}

func (r *Runner) debugProfileSnapshot() (DebugProfileSnapshot, bool) {
	return r.profile.snapshot(), true
}
