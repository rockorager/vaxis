package ui_test

import (
	"fmt"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis/ui"
)

type autoAnimationWidget struct {
	Start      time.Time
	Duration   time.Duration
	Controller **ui.AnimationController
}

func (w autoAnimationWidget) CreateState() ui.State {
	return &autoAnimationState{controller: w.Controller}
}

type autoAnimationState struct {
	ui.StateBase
	controller **ui.AnimationController
}

func (s *autoAnimationState) InitState() {
	w := s.Widget().(autoAnimationWidget)
	controller := s.NewAnimation(ui.AnimationOptions{Duration: w.Duration})
	controller.ForwardAt(w.Start)
	*s.controller = controller
}

func (s *autoAnimationState) Build(ui.BuildContext) ui.Widget {
	return ui.Text{Value: fmt.Sprintf("%.2f", (*s.controller).Value())}
}

func TestAnimationControllerSchedulesFramesUntilComplete(t *testing.T) {
	now := time.Unix(10, 0)
	backend := newFakeBackend(ui.Size{Width: 4, Height: 1})
	var controller *ui.AnimationController
	runner := ui.NewRunner(ui.NewApp(autoAnimationWidget{
		Start:      now,
		Duration:   time.Second,
		Controller: &controller,
	}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	if got := frameText(backend.frames[0]); got != "0.00" {
		t.Fatalf("initial frame = %q, want 0.00", got)
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("running animation should schedule another frame")
	}

	if err := runner.HandleFrame(now.Add(500 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if got := frameText(backend.frames[1]); got != "0.50" {
		t.Fatalf("halfway frame = %q, want 0.50", got)
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("running animation should keep scheduling frames")
	}

	if err := runner.HandleFrame(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := frameText(backend.frames[2]); got != "1.00" {
		t.Fatalf("final frame = %q, want 1.00", got)
	}
	if controller.Status() != ui.AnimationCompleted {
		t.Fatalf("status = %v, want completed", controller.Status())
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("final render update should schedule a cleanup frame")
	}
	if err := runner.HandleFrame(now.Add(time.Second + 16*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if _, ok := runner.NextFrame(); ok {
		t.Fatal("completed animation should not keep scheduling frames")
	}
}

func TestAnimationControllerDisposedWithState(t *testing.T) {
	now := time.Unix(10, 0)
	var controller *ui.AnimationController
	app := ui.NewApp(autoAnimationWidget{
		Start:      now,
		Duration:   time.Second,
		Controller: &controller,
	})
	app.Pump(ui.Size{Width: 4, Height: 1})
	app.UpdateRoot(ui.Text{Value: "done"})
	app.Pump(ui.Size{Width: 4, Height: 1})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("ForwardAt after state disposal did not panic")
		}
	}()
	controller.ForwardAt(now)
}

func TestAnimationHelpers(t *testing.T) {
	if got := ui.Linear(2); got != 1 {
		t.Fatalf("Linear(2) = %v, want 1", got)
	}
	if got := ui.EaseInOut(0.5); got != 0.5 {
		t.Fatalf("EaseInOut(0.5) = %v, want 0.5", got)
	}
	if got := (ui.FloatTween{Begin: 10, End: 20}).At(0.25); got != 12.5 {
		t.Fatalf("FloatTween.At = %v, want 12.5", got)
	}
}

func frameText(p *ui.Painter) string {
	var out string
	for x := 0; x < p.Size().Width; x++ {
		out += p.Cell(x, 0).Grapheme
	}
	return out
}
