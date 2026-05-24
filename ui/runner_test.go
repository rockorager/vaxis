package ui_test

import (
	"errors"
	"testing"
	"time"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/ui"
)

type fakeBackend struct {
	size        ui.Size
	events      chan ui.Event
	frames      []*ui.Painter
	mouseShapes []ui.MouseShape
	titles      []string
	copies      []string
	notices     []notice
	renderErr   error
}

type notice struct {
	title string
	body  string
}

type runnerQuitIntent struct{}

func (runnerQuitIntent) IntentType() ui.IntentType {
	return "test.runner.quit"
}

func newFakeBackend(size ui.Size) *fakeBackend {
	return &fakeBackend{size: size, events: make(chan ui.Event, 8)}
}

func (b *fakeBackend) Events() <-chan ui.Event {
	return b.events
}

func (b *fakeBackend) Size() ui.Size {
	return b.size
}

func (b *fakeBackend) Render(p *ui.Painter) error {
	b.frames = append(b.frames, p)
	return b.renderErr
}

func (b *fakeBackend) Dispatch(fn func()) {
	b.events <- vaxis.SyncFunc(fn)
}

func (b *fakeBackend) SetMouseShape(shape ui.MouseShape) {
	b.mouseShapes = append(b.mouseShapes, shape)
}

func (b *fakeBackend) SetTitle(title string) {
	b.titles = append(b.titles, title)
}

func (b *fakeBackend) CopyToClipboard(text string) {
	b.copies = append(b.copies, text)
}

func (b *fakeBackend) Notify(title, body string) {
	b.notices = append(b.notices, notice{title: title, body: body})
}

func (b *fakeBackend) Close() error {
	close(b.events)
	return nil
}

func TestRunnerRendersInitialFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 5, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "hi"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("expected initial frame to be scheduled")
	}
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	if len(backend.frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(backend.frames))
	}
	if got := backend.frames[0].Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first cell = %q, want h", got)
	}
}

func TestRunnerPropagatesBackendRenderError(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 5, Height: 1})
	backend.renderErr = errors.New("render failed")
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "hi"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); !errors.Is(err, backend.renderErr) {
		t.Fatalf("HandleFrame error = %v, want %v", err, backend.renderErr)
	}
}

func TestRunnerIgnoredEventDoesNotScheduleFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 1, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "x"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Key{Keycode: 'z'}, now.Add(time.Millisecond))
	if _, ok := runner.NextFrame(); ok {
		t.Fatal("ignored event should not schedule frame")
	}
}

func TestRunnerAppliesMouseShapeOnMouseEvent(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Align{Alignment: ui.TopLeft, Child: ui.Button{Label: "x"}}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion}, now.Add(time.Millisecond))
	if got := backend.mouseShapes[len(backend.mouseShapes)-1]; got != ui.MouseShapeClickable {
		t.Fatalf("mouse shape = %q, want clickable", got)
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("mouse shape change should schedule a frame so Vaxis flushes the shape")
	}
	if err := runner.HandleFrame(now.Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Mouse{Col: 10, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion}, now.Add(2*time.Millisecond))
	if got := backend.mouseShapes[len(backend.mouseShapes)-1]; got != ui.MouseShapeDefault {
		t.Fatalf("mouse shape = %q, want default", got)
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("mouse shape reset should schedule a frame so Vaxis flushes the shape")
	}
}

func TestRunnerDirtyEventSchedulesCoalescedFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) {}},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) {}},
	)), backend, ui.NewFrameScheduler(16*time.Millisecond))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now.Add(5*time.Millisecond))
	due, ok := runner.NextFrame()
	if !ok {
		t.Fatal("dirty event should schedule frame")
	}
	if want := now.Add(16 * time.Millisecond); !due.Equal(want) {
		t.Fatalf("due = %v, want %v", due, want)
	}
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now.Add(6*time.Millisecond))
	if due2, _ := runner.NextFrame(); !due2.Equal(due) {
		t.Fatalf("due changed from %v to %v; requests should coalesce", due, due2)
	}
}

func TestRunnerResizeSchedulesFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 1, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "x"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Resize{Cols: 2, Rows: 1}, now.Add(time.Millisecond))
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("resize should schedule frame")
	}
}

func TestRunnerRedrawSchedulesFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 1, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "x"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Redraw{}, now.Add(time.Millisecond))
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("redraw should schedule frame")
	}
}

func TestRunnerResizeRelayoutsAtBackendSize(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 5, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Center(ui.Text{Value: "x"})), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	if got := backend.frames[0].Cell(2, 0).Grapheme; got != "x" {
		t.Fatalf("initial centered cell = %q, want x", got)
	}
	backend.size = ui.Size{Width: 9, Height: 1}
	runner.HandleEvent(vaxis.Resize{Cols: 9, Rows: 1}, now.Add(time.Millisecond))
	if err := runner.HandleFrame(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := backend.frames[1].Cell(4, 0).Grapheme; got != "x" {
		t.Fatalf("resized centered cell = %q, want x", got)
	}
}

func TestRunnerSyncFuncRunsAndSchedulesFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 1, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text{Value: "x"}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	called := false
	runner.HandleEvent(vaxis.SyncFunc(func() { called = true }), now.Add(time.Millisecond))
	if !called {
		t.Fatal("sync func was not called")
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("sync func should schedule frame")
	}
}

func TestRunnerQuitEventStopsRunner(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			runnerQuitIntent{}.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				ctx.Quit()
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: map[string]ui.Intent{"q": runnerQuitIntent{}},
			Child:    ui.Button{Label: "x"},
		},
	}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Key{Text: "q", Keycode: 'q'}, now)
	if !runner.Done() {
		t.Fatal("quit shortcut should stop runner")
	}
}

func TestRunnerEventContextEffectsUseBackend(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Button{
		Label: "effects",
		OnPressed: func(ctx ui.EventContext) {
			ctx.SetTitle("Demo")
			ctx.Copy("copied")
			ctx.CopyToClipboard("legacy")
			ctx.Notify("Notice", "body")
		},
	}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}

	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyTab}, now)
	runner.HandleEvent(vaxis.Key{Keycode: vaxis.KeyEnter}, now)
	if len(backend.titles) != 1 || backend.titles[0] != "Demo" {
		t.Fatalf("titles = %#v, want Demo", backend.titles)
	}
	if len(backend.copies) != 2 || backend.copies[0] != "copied" || backend.copies[1] != "legacy" {
		t.Fatalf("copies = %#v, want copied and legacy", backend.copies)
	}
	if len(backend.notices) != 1 || backend.notices[0] != (notice{title: "Notice", body: "body"}) {
		t.Fatalf("notices = %#v, want Notice/body", backend.notices)
	}
}
