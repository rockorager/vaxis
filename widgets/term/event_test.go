package term

import (
	"testing"
	"time"
)

func TestPostEventDropsWhenQueueFull(t *testing.T) {
	vt := New()
	vt.postEvent(EventBell{})
	vt.postEvent(EventBell{})

	done := make(chan struct{})
	go func() {
		vt.postEvent(EventBell{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("postEvent blocked on full event queue")
	}
}

func TestAttachNilHandlerDoesNotPanic(t *testing.T) {
	vt := New()
	vt.Attach(nil)

	withoutPanic(t, func() {
		vt.dispatchEvent(EventBell{})
	})
}
