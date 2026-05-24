package ui_test

import (
	"testing"
	"time"

	"go.rockorager.dev/vaxis/ui"
)

type runtimeCaptureWidget struct{ Runtime *ui.Runtime }

func (w runtimeCaptureWidget) Build(ctx ui.BuildContext) ui.Widget {
	rt := ctx.Runtime()
	*w.Runtime = rt
	return ui.Text{Value: "x"}
}

func TestBuildContextRuntimeDispatchRunsOnRunner(t *testing.T) {
	now := time.Unix(10, 0)
	var rt ui.Runtime
	backend := newFakeBackend(ui.Size{Width: 5, Height: 1})
	runner := ui.NewRunner(ui.NewApp(runtimeCaptureWidget{Runtime: &rt}), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	called := false
	rt.Dispatch(func() { called = true })
	select {
	case ev := <-backend.events:
		runner.HandleEvent(ev, now.Add(time.Millisecond))
	default:
		t.Fatal("Dispatch did not enqueue work on backend event loop")
	}
	if !called {
		t.Fatal("dispatched function did not run")
	}
	if _, ok := runner.NextFrame(); !ok {
		t.Fatal("dispatched function should schedule a frame")
	}
}

func TestRuntimeDispatchNilPanics(t *testing.T) {
	var rt ui.Runtime
	app := ui.NewApp(runtimeCaptureWidget{Runtime: &rt})
	app.Pump(ui.Size{Width: 1, Height: 1})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Dispatch(nil) did not panic")
		}
	}()
	rt.Dispatch(nil)
}
