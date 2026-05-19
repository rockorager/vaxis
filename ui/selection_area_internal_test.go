package ui

import (
	"math"
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

func TestSelectionAutoScrollVelocity(t *testing.T) {
	metrics := ScrollMetrics{ViewportHeight: 4}
	if got := selectionAutoScrollVelocity(2.4, Offset{}, metrics); got != 0 {
		t.Fatalf("middle velocity = %v, want 0", got)
	}
	if got := selectionAutoScrollVelocity(3.75, Offset{}, metrics); got <= 0 || got >= 80 {
		t.Fatalf("bottom edge velocity = %v, want between 0 and 80", got)
	}
	if got := selectionAutoScrollVelocity(-2.5, Offset{}, metrics); math.Abs(got+80) > 0.001 {
		t.Fatalf("outside top velocity = %v, want -80", got)
	}
	if got := selectionAutoScrollVelocity(6, Offset{}, metrics); math.Abs(got-80) > 0.001 {
		t.Fatalf("outside bottom velocity = %v, want 80", got)
	}
}
