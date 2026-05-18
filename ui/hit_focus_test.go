package ui

import "testing"

func TestRemovedWidgetNoLongerReceivesMouseEvents(t *testing.T) {
	clicks := 0
	app := NewApp(Row(Button{Label: "hit", OnPressed: func(EventContext) { clicks++ }}))
	app.Pump(Size{Width: 10, Height: 1})
	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if clicks != 1 {
		t.Fatalf("clicks = %d, want 1", clicks)
	}
	app.UpdateRoot(Row())
	app.Pump(Size{Width: 10, Height: 1})
	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if clicks != 1 {
		t.Fatalf("removed widget received click; clicks = %d, want 1", clicks)
	}
}

func TestMovedWidgetOnlyHitsAtNewLocation(t *testing.T) {
	clicks := 0
	app := NewApp(Row(Button{Label: "hit", OnPressed: func(EventContext) { clicks++ }}))
	app.Pump(Size{Width: 20, Height: 1})
	app.UpdateRoot(Row(SizedBox{Width: 5, Height: 1}, Button{Label: "hit", OnPressed: func(EventContext) { clicks++ }}))
	app.Pump(Size{Width: 20, Height: 1})
	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if clicks != 0 {
		t.Fatalf("old location received click; clicks = %d, want 0", clicks)
	}
	app.Send(Mouse{Col: 6, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if clicks != 1 {
		t.Fatalf("new location clicks = %d, want 1", clicks)
	}
}

func TestFocusedNodeDetachesAndReplacementIsNotified(t *testing.T) {
	var firstChanges, secondChanges int
	first := &FocusNode{onChange: func() { firstChanges++ }}
	second := &FocusNode{onChange: func() { secondChanges++ }}
	app := NewApp(Column(Focus(first, Text{Value: "one"}), Focus(second, Text{Value: "two"})))
	app.Pump(Size{Width: 10, Height: 2})
	if !first.HasFocus() {
		t.Fatal("first node should be initially focused")
	}
	app.UpdateRoot(Column(Focus(second, Text{Value: "two"})))
	app.Pump(Size{Width: 10, Height: 1})
	if first.HasFocus() || first.app != nil || first.element != nil {
		t.Fatal("removed focused node should detach")
	}
	if !second.HasFocus() {
		t.Fatal("remaining node should become focused")
	}
	if firstChanges == 0 || secondChanges == 0 {
		t.Fatalf("focus changes = first:%d second:%d, want both notified", firstChanges, secondChanges)
	}
}
