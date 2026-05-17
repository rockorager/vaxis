package ui_test

import (
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type fakeBackend struct {
	size   ui.Size
	events chan ui.Event
	frames []*ui.Painter
}

func newFakeBackend(size ui.Size) *fakeBackend {
	return &fakeBackend{size: size, events: make(chan ui.Event, 8)}
}

func (b *fakeBackend) Events() <-chan ui.Event { return b.events }
func (b *fakeBackend) Size() ui.Size           { return b.size }
func (b *fakeBackend) Render(p *ui.Painter) error {
	b.frames = append(b.frames, p)
	return nil
}
func (b *fakeBackend) Close() error { close(b.events); return nil }

func TestRunnerRendersInitialFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 5, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text("hi")), backend, ui.NewFrameScheduler(time.Second/60))
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
	if got := backend.frames[0].Cell(0, 0).Character.Grapheme; got != "h" {
		t.Fatalf("first cell = %q, want h", got)
	}
}

func TestRunnerIgnoredEventDoesNotScheduleFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 1, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Text("x")), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Key{Keycode: 'z'}, now.Add(time.Millisecond))
	if _, ok := runner.NextFrame(); ok {
		t.Fatal("ignored event should not schedule frame")
	}
}

func TestRunnerDirtyEventSchedulesCoalescedFrame(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Row(
		ui.Button("one", func(ctx ui.EventContext) {}),
		ui.Button("two", func(ctx ui.EventContext) {}),
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
	runner := ui.NewRunner(ui.NewApp(ui.Text("x")), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Resize{Cols: 2, Rows: 1}, now.Add(time.Millisecond))
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("resize should schedule frame")
	}
}

func TestRunnerQuitEventStopsRunner(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 20, Height: 1})
	runner := ui.NewRunner(ui.NewApp(ui.Keymap(map[string]ui.VoidCallback{"q": func(ctx ui.EventContext) { ctx.Quit() }}, ui.Button("x", nil))), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	runner.HandleEvent(vaxis.Key{Text: "q", Keycode: 'q'}, now)
	if !runner.Done() {
		t.Fatal("quit shortcut should stop runner")
	}
}
