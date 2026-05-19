package ui

import (
	"strconv"
	"testing"
)

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

func TestSliverListBuilderBuildsBoundedInitialRange(t *testing.T) {
	built := map[int]bool{}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:      1000,
			ItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				built[i] = true
				return Text{Value: "row"}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 4})

	if len(built) == 0 || len(built) > defaultSliverListBuilderInitialCount {
		t.Fatalf("built %d rows, want a bounded initial range", len(built))
	}
	if built[999] {
		t.Fatal("builder eagerly built the last row")
	}
}

func TestSliverListBuilderScrollsToVisibleRange(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:      100,
			ItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				return Text{Value: "row " + strconv.Itoa(i)}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 3})
	app.Pump(Size{Width: 10, Height: 3})

	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 3})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme + p.Cell(5, 0).Grapheme; got != "97" {
		t.Fatalf("first visible row = %q, want row 97", got)
	}
}

func TestSliverListBuilderReportsScrollbarMetrics(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:      100,
			ItemExtent: 2,
			Builder: func(ctx BuildContext, i int) Widget {
				return Text{Value: strconv.Itoa(i)}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 5})

	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	got := r.ScrollMetrics()
	want := ScrollMetrics{MaxScrollOffset: 195, ViewportHeight: 5, ViewportWidth: 10, ContentHeight: 200}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestSliverListBuilderHitTestsVisibleRows(t *testing.T) {
	clicked := false
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:      10,
			ItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				if i == 9 {
					return Button{Label: "nine", OnPressed: func(EventContext) { clicked = true }}
				}
				return Text{Value: strconv.Itoa(i)}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Col: 1, Row: 1, Button: MouseLeftButton, EventType: EventPress})
	if !clicked {
		t.Fatal("expected hit testing to reach visible lazy list row")
	}
}

func TestFixedSliverExtentModel(t *testing.T) {
	model := fixedSliverExtentModel{Count: 10, Extent: 2}
	if got := model.ScrollExtent(); got != 20 {
		t.Fatalf("scroll extent = %d, want 20", got)
	}
	if got := model.OffsetForIndex(4); got != 8 {
		t.Fatalf("offset for index = %d, want 8", got)
	}
	if got := model.IndexForOffset(9); got != 4 {
		t.Fatalf("index for offset = %d, want 4", got)
	}
	first, last := model.VisibleRange(1, SliverConstraints{ViewportHeight: 5, RemainingPaintExtent: 5, ScrollOffset: 6})
	if first != 2 || last != 7 {
		t.Fatalf("visible range = %d,%d, want 2,7", first, last)
	}
}

func TestMeasuredSliverExtentModel(t *testing.T) {
	model := measuredSliverExtentModel{
		Count:    5,
		Estimate: 2,
		Extents:  map[int]int{1: 4, 3: 1},
	}
	if got := model.ScrollExtent(); got != 11 {
		t.Fatalf("scroll extent = %d, want 11", got)
	}
	if got := model.OffsetForIndex(3); got != 8 {
		t.Fatalf("offset for index = %d, want 8", got)
	}
	if got := model.IndexForOffset(6); got != 2 {
		t.Fatalf("index for offset = %d, want 2", got)
	}
	first, last := model.VisibleRange(1, SliverConstraints{ViewportHeight: 3, RemainingPaintExtent: 3, ScrollOffset: 4})
	if first != 0 || last != 4 {
		t.Fatalf("visible range = %d,%d, want 0,4", first, last)
	}
	model.Update(2, 5)
	if got := model.ScrollExtent(); got != 14 {
		t.Fatalf("scroll extent after update = %d, want 14", got)
	}
}

func TestSliverListBuilderVariableHeightsUpdateMetrics(t *testing.T) {
	heights := []int{1, 3, 2, 1}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:               len(heights),
			EstimatedItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				return SizedBox{Width: 10, Height: heights[i], Child: Text{Value: strconv.Itoa(i)}}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 3})

	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	got := r.ScrollMetrics()
	want := ScrollMetrics{MaxScrollOffset: 4, ViewportHeight: 3, ViewportWidth: 10, ContentHeight: 7}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestSliverListBuilderVariableHeightsScrollToMeasuredOffset(t *testing.T) {
	heights := []int{1, 3, 1, 1, 1}
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:               len(heights),
			EstimatedItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				return SizedBox{Width: 10, Height: heights[i], Child: Text{Value: "row " + strconv.Itoa(i)}}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	r.ScrollToOffset(4)
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme; got != "2" {
		t.Fatalf("first visible row suffix = %q, want row 2", got)
	}
}

func TestSliverListBuilderVariableHeightsCorrectsMeasuredRowsAboveViewport(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:               100,
			EstimatedItemExtent: 1,
			Overscan:            2,
			Builder: func(ctx BuildContext, i int) Widget {
				height := 1
				if i == 40 {
					height = 4
				}
				return SizedBox{Width: 10, Height: height, Child: Text{Value: "row " + strconv.Itoa(i)}}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	r.ScrollToOffset(42)
	app.Pump(Size{Width: 10, Height: 2})
	app.Pump(Size{Width: 10, Height: 2})

	if got := r.ScrollMetrics().ScrollOffset; got != 45 {
		t.Fatalf("scroll offset after measuring row above viewport = %d, want 45", got)
	}
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme + p.Cell(5, 0).Grapheme; got != "42" {
		t.Fatalf("first visible row suffix = %q, want row 42", got)
	}
}

func TestSliverListBuilderVariableHeightsAnchorsVisibleRowOnResize(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:               20,
			EstimatedItemExtent: 1,
			Overscan:            3,
			Builder: func(ctx BuildContext, i int) Widget {
				return Text{Value: "row " + padTestInt(i, 2) + " abcdefghij", SoftWrap: true}
			},
		},
	}})
	app.Pump(Size{Width: 20, Height: 3})
	app.Pump(Size{Width: 20, Height: 3})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	r.ScrollToOffset(6)
	app.Pump(Size{Width: 20, Height: 3})
	app.Pump(Size{Width: 20, Height: 3})

	app.Pump(Size{Width: 10, Height: 3})
	app.Pump(Size{Width: 10, Height: 3})
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(4, 0).Grapheme + p.Cell(5, 0).Grapheme; got != "06" {
		t.Fatalf("first visible row suffix after resize = %q, want row 06", got)
	}
}

func TestCustomScrollViewAnchorsMixedSliversOnResize(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverToBox{Child: Text{Value: "intro"}},
		SliverPinnedHeader{Child: Text{Value: "head", Style: Style{Background: RGB(12, 34, 56)}}},
		SliverListBuilder{
			Count:               20,
			EstimatedItemExtent: 1,
			Overscan:            3,
			Builder: func(ctx BuildContext, i int) Widget {
				return Text{Value: "row " + padTestInt(i, 2) + " abcdefghij", SoftWrap: true}
			},
		},
		SliverToBox{Child: Text{Value: "footer"}},
	}})
	app.Pump(Size{Width: 20, Height: 4})
	app.Pump(Size{Width: 20, Height: 4})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	r.ScrollToOffset(8)
	app.Pump(Size{Width: 20, Height: 4})
	app.Pump(Size{Width: 20, Height: 4})

	app.Pump(Size{Width: 10, Height: 4})
	app.Pump(Size{Width: 10, Height: 4})
	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme + p.Cell(1, 0).Grapheme + p.Cell(2, 0).Grapheme + p.Cell(3, 0).Grapheme; got != "head" {
		t.Fatalf("pinned header row = %q, want head", got)
	}
	if got := p.Cell(4, 1).Grapheme + p.Cell(5, 1).Grapheme; got != "07" {
		t.Fatalf("first visible list row suffix after resize = %q, want row 07", got)
	}
}

func TestCustomScrollViewResizeToFitContentClearsBottomOffset(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverList{ChildrenWidget: []Widget{
			Text{Value: "one"},
			Text{Value: "two"},
			Text{Value: "three"},
		}},
	}})
	app.Pump(Size{Width: 10, Height: 1})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 1})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	if got := r.ScrollMetrics().ScrollOffset; got != 2 {
		t.Fatalf("scroll offset before resize = %d, want bottom offset 2", got)
	}

	app.Pump(Size{Width: 10, Height: 3})
	if got := r.ScrollMetrics().ScrollOffset; got != 0 {
		t.Fatalf("scroll offset after content fits = %d, want 0", got)
	}
	p := NewPainter(Size{Width: 10, Height: 3})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after content fits = %q, want one", got)
	}
}

func TestSliverListBuilderVariableHeightsResetOnWidthChange(t *testing.T) {
	app := NewApp(CustomScrollView{Slivers: []Widget{
		SliverListBuilder{
			Count:               1,
			EstimatedItemExtent: 1,
			Builder: func(ctx BuildContext, i int) Widget {
				return Text{Value: "abcdefghij", SoftWrap: true}
			},
		},
	}})
	app.Pump(Size{Width: 10, Height: 3})
	r, ok := app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	if got := r.ScrollMetrics().ContentHeight; got != 1 {
		t.Fatalf("wide content height = %d, want 1", got)
	}

	app.Pump(Size{Width: 5, Height: 3})
	r, ok = app.rootRO.(*renderCustomScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderCustomScrollView", app.rootRO)
	}
	if got := r.ScrollMetrics().ContentHeight; got != 2 {
		t.Fatalf("narrow content height = %d, want 2", got)
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

func padTestInt(v, width int) string {
	s := strconv.Itoa(v)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
