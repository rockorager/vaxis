package ui

import "testing"

func TestCustomScrollViewComposesSlivers(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
		SliverToBox{Child: Text{Value: "footer"}},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first row = %q, want header", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "o" {
		t.Fatalf("second row = %q, want one", got)
	}
	if got := p.Cell(0, 2).Grapheme; got != "t" {
		t.Fatalf("third row = %q, want two", got)
	}
	if got := p.Cell(0, 3).Grapheme; got != "" {
		t.Fatalf("clipped row = %q, want empty", got)
	}
}

func TestCustomScrollViewWheelAndKeyboardScroll(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
			Text{Value: "four"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after wheel = %q, want one", got)
	}

	app.Send(Key{Keycode: KeyPgDown})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after page down = %q, want three", got)
	}

	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after end = %q, want three", got)
	}

	app.Send(Key{Keycode: KeyHome})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first visible row after home = %q, want header", got)
	}
}

func TestCustomScrollViewReportsScrollMetrics(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	got := r.ScrollMetrics()
	want := ScrollMetrics{ScrollOffset: 1, MaxScrollOffset: 2, ViewportHeight: 2, ViewportWidth: 10, ContentHeight: 4}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestCustomScrollViewWorksWithScrollbar(t *testing.T) {
	app := NewApp(Scrollbar{Child: CustomScrollView{Slivers: []Widget{
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
			Text{Value: "four"},
			Text{Value: "five"},
		}},
	}}})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Col: 9, Row: 1, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after scrollbar track click = %q, want three", got)
	}
}

func TestCustomScrollViewClampsOffsetWhenSliversShrink(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
			Text{Value: "four"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 2})

	app.UpdateRoot(CustomScrollView{Slivers: []Widget{
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after shrink = %q, want two", got)
	}
}

func TestSliverListReportsChildOffsetsForHitTesting(t *testing.T) {
	clicked := false
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Button{Label: "one"},
			Button{Label: "two", OnPressed: func(EventContext) { clicked = true }},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Col: 1, Row: 1, Button: MouseLeftButton, EventType: EventPress})
	if !clicked {
		t.Fatal("expected hit testing to reach scrolled sliver list child")
	}
}
