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

func TestSliverFillRemainingFillsViewport(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverFillRemaining{Child: Text{Value: "body", Style: Style{Background: RGB(12, 34, 56)}}},
	}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("first row = %q, want header", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "b" {
		t.Fatalf("fill child row = %q, want body", got)
	}
	if got := p.Cell(0, 3).Background; got != RGB(12, 34, 56) {
		t.Fatalf("bottom fill background = %#v, want fill child background", got)
	}
}

func TestSliverFillRemainingScrollsTallChild(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverFillRemaining{Child: Flex{
			Axis:               Vertical,
			CrossAxisAlignment: CrossAxisStart,
			ChildrenWidget: []Widget{
				Text{Value: "one"},
				Text{Value: "two"},
				Text{Value: "three"},
				Text{Value: "four"},
			},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after end = %q, want two", got)
	}
	if got := p.Cell(0, 2).Grapheme; got != "f" {
		t.Fatalf("last visible row after end = %q, want four", got)
	}
}

func TestSliverFillRemainingContributesScrollbarMetrics(t *testing.T) {
	app := NewApp(Scrollbar{Child: CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "header"}},
		SliverFillRemaining{Child: Flex{
			Axis:               Vertical,
			CrossAxisAlignment: CrossAxisStart,
			ChildrenWidget: []Widget{
				Text{Value: "one"},
				Text{Value: "two"},
				Text{Value: "three"},
				Text{Value: "four"},
			},
		}},
	}}})
	app.Pump(Size{Width: 10, Height: 3})

	app.Send(Mouse{Col: 9, Row: 2, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after track click = %q, want three", got)
	}
}

func TestSliverPinnedHeaderStartsInNormalPosition(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "intro"}},
		SliverPinnedHeader{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "i" {
		t.Fatalf("first row = %q, want intro", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "h" {
		t.Fatalf("second row = %q, want header", got)
	}
}

func TestSliverPinnedHeaderStaysAtTopAfterScroll(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverPinnedHeader{Child: Text{Value: "header", Style: Style{Background: RGB(20, 40, 60)}}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
			Text{Value: "four"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "h" {
		t.Fatalf("pinned row = %q, want header", got)
	}
	if got := p.Cell(0, 0).Background; got != RGB(20, 40, 60) {
		t.Fatalf("pinned row background = %#v, want header background", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "t" {
		t.Fatalf("row after pinned header = %q, want two", got)
	}
}

func TestSliverPinnedHeaderFillsWidthWhenPinned(t *testing.T) {
	header := RGB(20, 40, 60)
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverPinnedHeader{Child: Text{Value: "head", Style: Style{Background: header}}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "row number 1"},
			Text{Value: "row number 2"},
			Text{Value: "row number 3"},
		}},
	}})
	app.Pump(Size{Width: 12, Height: 2})

	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 12, Height: 2})
	p := NewPainter(Size{Width: 12, Height: 2})
	app.Paint(p)
	if got := p.Cell(4, 0); got.Grapheme != " " || got.Background != header {
		t.Fatalf("cell after short pinned header = %#v, want header fill", got)
	}
}

func TestSliverPinnedHeaderContributesScrollbarMetrics(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverPinnedHeader{Child: Text{Value: "header"}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})

	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	got := r.ScrollMetrics()
	want := ScrollMetrics{MaxScrollOffset: 2, ViewportHeight: 2, ViewportWidth: 10, ContentHeight: 4}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestSliverPinnedHeaderHitTestsAtPinnedOffset(t *testing.T) {
	clicked := false
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverPinnedHeader{Child: Button{Label: "head", OnPressed: func(EventContext) { clicked = true }}},
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if !clicked {
		t.Fatal("expected pinned header button to receive click at pinned row")
	}
}
