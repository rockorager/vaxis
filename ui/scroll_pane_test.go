package ui

import "testing"

func TestScrollPaneScrollsBothAxes(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 2, Child: ScrollPane{
		Child: Text{Value: "abcde\nfghij\nklmno"},
	}})
	app.Pump(Size{Width: 3, Height: 2})

	p := NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc\nfgh" {
		t.Fatalf("initial pane = %q, want visible top-left", got)
	}

	app.Send(Key{Keycode: KeyRight})
	app.Pump(Size{Width: 3, Height: 2})
	p = NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "bcd\nghi" {
		t.Fatalf("after right = %q, want horizontally scrolled", got)
	}

	app.Send(Key{Keycode: KeyDown})
	app.Pump(Size{Width: 3, Height: 2})
	p = NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "ghi\nlmn" {
		t.Fatalf("after down = %q, want both offsets applied", got)
	}
}

func TestScrollPaneScrollIntentCanScrollHorizontally(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 2, Child: Shortcuts{
		Bindings: map[string]Intent{
			"x": ScrollIntent{Axis: ScrollHorizontal, Direction: ScrollForward, Unit: ScrollUnitLine},
		},
		Child: ScrollPane{
			Child: Text{Value: "abcde\nfghij\nklmno"},
		},
	}})
	app.Pump(Size{Width: 3, Height: 2})

	app.Send(Key{Text: "x", Keycode: 'x'})
	app.Pump(Size{Width: 3, Height: 2})

	p := NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "bcd\nghi" {
		t.Fatalf("after horizontal scroll shortcut = %q, want bcd/ghi", got)
	}
}

func TestScrollPaneMouseWheelsScrollBothAxes(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 2, Child: ScrollPane{
		Child: Text{Value: "abcde\nfghij\nklmno"},
	}})
	app.Pump(Size{Width: 3, Height: 2})

	app.Send(Mouse{Button: MouseWheelRight, EventType: EventPress})
	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 3, Height: 2})

	p := NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "ghi\nlmn" {
		t.Fatalf("after wheel right/down = %q, want both offsets applied", got)
	}
}

func TestScrollPaneReportsMetricsForEachAxis(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 2, Child: ScrollPane{
		Child: Text{Value: "abcde\nfghij\nklmno"},
	}})
	app.Pump(Size{Width: 3, Height: 2})
	app.Send(Key{Keycode: KeyRight})
	app.Send(Key{Keycode: KeyDown})
	app.Pump(Size{Width: 3, Height: 2})

	box, ok := app.rootRO.(*renderSizedBox)
	if !ok {
		t.Fatalf("root render object = %T, want *renderSizedBox", app.rootRO)
	}
	pane, ok := box.Child().(*renderScrollPane)
	if !ok {
		t.Fatalf("child render object = %T, want *renderScrollPane", box.Child())
	}
	if got, want := pane.ScrollMetricsForAxis(ScrollVertical), (ScrollMetrics{ScrollOffset: 1, MaxScrollOffset: 1, ViewportHeight: 2, ViewportWidth: 3, ContentHeight: 3, ContentWidth: 5}); got != want {
		t.Fatalf("vertical metrics = %#v, want %#v", got, want)
	}
	if got, want := pane.ScrollMetricsForAxis(ScrollHorizontal), (ScrollMetrics{ScrollOffset: 1, MaxScrollOffset: 2, ViewportHeight: 2, ViewportWidth: 3, ContentHeight: 3, ContentWidth: 5}); got != want {
		t.Fatalf("horizontal metrics = %#v, want %#v", got, want)
	}
}

func TestScrollPaneControllerScrollsBothAxes(t *testing.T) {
	controller := &ScrollPaneController{}
	app := NewApp(SizedBox{Width: 3, Height: 2, Child: ScrollPane{
		Controller: controller,
		Child:      Text{Value: "abcde\nfghij\nklmno"},
	}})
	app.Pump(Size{Width: 3, Height: 2})

	if !controller.ScrollTo(2, 1) {
		t.Fatal("controller did not scroll pane")
	}
	app.Pump(Size{Width: 3, Height: 2})

	p := NewPainter(Size{Width: 3, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got != "hij\nmno" {
		t.Fatalf("after controller scroll = %q, want bottom-right", got)
	}
}

func TestScrollPaneWorksWithBothScrollbarAxes(t *testing.T) {
	app := NewApp(SizedBox{Width: 4, Height: 3, Child: Scrollbar{
		Axis: ScrollHorizontal,
		Child: Scrollbar{
			Child: ScrollPane{Child: Text{Value: "abcdef\nabcdef\nabcdef\nabcdef\nabcdef"}},
		},
	}})
	app.Pump(Size{Width: 4, Height: 3})

	app.Send(Mouse{Col: 0, Row: 2, Button: MouseLeftButton, EventType: EventPress})
	app.Send(Mouse{Col: 2, Row: 2, Button: MouseLeftButton, EventType: EventMotion})
	app.Send(Mouse{Col: 2, Row: 2, Button: MouseLeftButton, EventType: EventRelease})
	app.Send(Mouse{Col: 3, Row: 1, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 4, Height: 3})

	p := NewPainter(Size{Width: 4, Height: 3})
	app.Paint(p)
	if got := debugRenderedText(p); got[:3] != "cde" {
		t.Fatalf("visible text after both scrollbars = %q, want horizontal offset", got)
	}
	if got := p.Cell(3, 0).Grapheme; got == "" {
		t.Fatalf("vertical scrollbar did not paint after scrolling: %#v", p.Cell(3, 0))
	}
	if got := p.Cell(0, 2).Grapheme; got != "▄" {
		t.Fatalf("horizontal scrollbar cell = %q, want lower block", got)
	}
}
