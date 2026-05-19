package ui

import (
	"testing"
	"time"
)

func TestSelectionAreaMouseClickCountUsesClock(t *testing.T) {
	now := time.Unix(10, 0)
	area := selectionAreaState{now: func() time.Time { return now }}
	mouse := Mouse{Col: 1, Row: 2, Button: MouseLeftButton, EventType: EventPress}

	if got := area.mouseClickCount(mouse); got != 1 {
		t.Fatalf("first click = %d, want 1", got)
	}
	now = now.Add(100 * time.Millisecond)
	if got := area.mouseClickCount(mouse); got != 2 {
		t.Fatalf("second click = %d, want 2", got)
	}
	now = now.Add(100 * time.Millisecond)
	if got := area.mouseClickCount(mouse); got != 3 {
		t.Fatalf("third click = %d, want 3", got)
	}
	now = now.Add(time.Second)
	if got := area.mouseClickCount(mouse); got != 1 {
		t.Fatalf("late click = %d, want reset to 1", got)
	}
}

func TestSelectionAreaMouseClickCountResetsWhenCellChanges(t *testing.T) {
	now := time.Unix(10, 0)
	area := selectionAreaState{now: func() time.Time { return now }}

	if got := area.mouseClickCount(Mouse{Col: 1, Row: 2, Button: MouseLeftButton, EventType: EventPress}); got != 1 {
		t.Fatalf("first click = %d, want 1", got)
	}
	now = now.Add(100 * time.Millisecond)
	if got := area.mouseClickCount(Mouse{Col: 2, Row: 2, Button: MouseLeftButton, EventType: EventPress}); got != 1 {
		t.Fatalf("moved click = %d, want reset to 1", got)
	}
}
