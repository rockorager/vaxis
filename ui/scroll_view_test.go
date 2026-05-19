package ui

import "testing"

func TestScrollViewClipsChildToViewport(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three")})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row = %q, want o", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "t" {
		t.Fatalf("second visible row = %q, want t", got)
	}
	if got := p.Cell(0, 2).Grapheme; got != "" {
		t.Fatalf("clipped row = %q, want empty", got)
	}
}

func TestScrollViewClipIntersectsChildClips(t *testing.T) {
	app := NewApp(Align{Alignment: TopLeft, Child: SizedBox{
		Width:  10,
		Height: 2,
		Child:  ScrollView{Child: scrollViewLines("one", "two", "three")},
	}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 2).Grapheme; got != "" {
		t.Fatalf("paint outside viewport = %q, want empty", got)
	}
}

func TestScrollViewMouseWheelScrolls(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three")})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after wheel = %q, want t", got)
	}
	if got := p.Cell(0, 1).Grapheme; got != "t" {
		t.Fatalf("second visible row after wheel = %q, want t", got)
	}
}

func TestScrollViewKeyboardScrolls(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three", "four", "five")})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Key{Keycode: KeyPgDown})
	app.Pump(Size{Width: 10, Height: 2})
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after page down = %q, want t", got)
	}

	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "f" {
		t.Fatalf("first visible row after end = %q, want f", got)
	}

	app.Send(Key{Keycode: KeyHome})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after home = %q, want o", got)
	}
}

func TestScrollViewPreservesOffsetAcrossUpdate(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three")})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 2})

	app.UpdateRoot(ScrollView{Child: scrollViewLines("one", "two", "three")})
	app.Pump(Size{Width: 10, Height: 2})
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after update = %q, want t", got)
	}
}

func TestScrollViewLaysOutChildWithUnboundedHeight(t *testing.T) {
	r := &renderScrollView{}
	r.SetChild(&renderText{Text: "one\ntwo\nthree"})
	r.Layout(LayoutContext{}, Constraints{MaxWidth: 10, MaxHeight: 2})
	if got := r.childSize.Height; got != 3 {
		t.Fatalf("child height = %d, want 3", got)
	}
	if got := r.Size().Height; got != 2 {
		t.Fatalf("viewport height = %d, want 2", got)
	}
}

func TestScrollViewDryLayoutConstrainsViewport(t *testing.T) {
	r := &renderScrollView{}
	r.SetChild(&renderText{Text: "one\ntwo\nthree"})
	got := r.DryLayout(LayoutContext{}, Constraints{MaxWidth: 10, MaxHeight: 2})
	want := Size{Width: 5, Height: 2}
	if got != want {
		t.Fatalf("dry layout = %#v, want %#v", got, want)
	}
}

func TestScrollViewReportsScrollMetrics(t *testing.T) {
	r := &renderScrollView{}
	r.SetChild(&renderText{Text: "one\ntwo\nthree"})
	r.Layout(LayoutContext{}, Constraints{MaxWidth: 10, MaxHeight: 2})
	r.State = &scrollViewState{scrollRow: 1}

	got := r.ScrollMetrics()
	want := ScrollMetrics{ScrollOffset: 1, MaxScrollOffset: 1, ViewportHeight: 2, ContentHeight: 3}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func scrollViewLines(lines ...string) Widget {
	children := make([]Widget, 0, len(lines))
	for _, line := range lines {
		children = append(children, Text{Value: line})
	}
	return Flex{Axis: Vertical, CrossAxisAlignment: CrossAxisStart, ChildrenWidget: children}
}
