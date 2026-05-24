package ui

import (
	"testing"

	"go.rockorager.dev/vaxis"
)

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

	app.Send(Key{Keycode: KeyDown})
	app.Pump(Size{Width: 10, Height: 2})
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after down = %q, want two", got)
	}

	app.Send(Key{Text: "k", Keycode: 'k'})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after k = %q, want one", got)
	}

	app.Send(Key{Text: "j", Keycode: 'j'})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after j = %q, want two", got)
	}

	app.Send(Key{Keycode: vaxis.KeySpace})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "f" {
		t.Fatalf("first visible row after space = %q, want four", got)
	}

	app.Send(Key{Keycode: vaxis.KeySpace, Modifiers: vaxis.ModShift})
	app.Pump(Size{Width: 10, Height: 2})
	p = NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after shift+space = %q, want two", got)
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

func TestScrollViewRevealsFocusedRichTextSpan(t *testing.T) {
	app := NewApp(SizedBox{Width: 10, Height: 2, Child: ScrollView{Child: RichText{Spans: []TextSpan{
		{Text: "top", OnPressed: func(EventContext) {}},
		{Text: "\nmid\n"},
		{Text: "bottom", OnPressed: func(EventContext) {}},
	}}}})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Key{Keycode: vaxis.KeyTab})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Key{Keycode: vaxis.KeyTab})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 1).Grapheme; got != "b" {
		t.Fatalf("focused bottom link row = %q, want b", got)
	}
}

func TestScrollViewScrollIntentCanBeInvokedByShortcut(t *testing.T) {
	app := NewApp(Shortcuts{
		Bindings: map[string]Intent{
			"x": ScrollIntent{Axis: ScrollVertical, Direction: ScrollForward, Unit: ScrollUnitLine},
		},
		Child: ScrollView{Child: scrollViewLines("one", "two", "three")},
	})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Key{Text: "x", Keycode: 'x'})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after shortcut = %q, want two", got)
	}
}

func TestScrollViewScrollIntentCanBeOverridden(t *testing.T) {
	overridden := false
	app := NewApp(Actions{
		Bindings: map[IntentType]ActionFunc{
			ScrollIntentType: func(ctx EventContext, intent Intent) EventResult {
				overridden = true
				return EventHandled
			},
		},
		Child: ScrollView{Child: scrollViewLines("one", "two", "three")},
	})
	app.Pump(Size{Width: 10, Height: 2})

	app.Send(Key{Keycode: KeyDown})
	app.Pump(Size{Width: 10, Height: 2})

	if !overridden {
		t.Fatal("expected ancestor action to override scroll")
	}
	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after override = %q, want one", got)
	}
}

func TestScrollViewHorizontalScrolls(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 1, Child: ScrollView{
		Axis:  ScrollHorizontal,
		Child: Text{Value: "abcdef"},
	}})
	app.Pump(Size{Width: 3, Height: 1})

	p := NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc" {
		t.Fatalf("rendered text = %q, want abc", got)
	}

	app.Send(Key{Keycode: KeyRight})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "bcd" {
		t.Fatalf("after right = %q, want bcd", got)
	}

	app.Send(Key{Text: "l", Keycode: 'l'})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "cde" {
		t.Fatalf("after l = %q, want cde", got)
	}

	app.Send(Key{Text: "h", Keycode: 'h'})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "bcd" {
		t.Fatalf("after h = %q, want bcd", got)
	}

	app.Send(Key{Keycode: vaxis.KeySpace})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "def" {
		t.Fatalf("after space = %q, want def", got)
	}

	app.Send(Key{Keycode: vaxis.KeySpace, Modifiers: vaxis.ModShift})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc" {
		t.Fatalf("after shift+space = %q, want abc", got)
	}

	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "def" {
		t.Fatalf("after end = %q, want def", got)
	}

	app.Send(Key{Keycode: KeyHome})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc" {
		t.Fatalf("after home = %q, want abc", got)
	}
}

func TestScrollViewHorizontalMouseWheelScrolls(t *testing.T) {
	app := NewApp(SizedBox{Width: 3, Height: 1, Child: ScrollView{
		Axis:  ScrollHorizontal,
		Child: Text{Value: "abcdef"},
	}})
	app.Pump(Size{Width: 3, Height: 1})

	app.Send(Mouse{Button: MouseWheelDown, EventType: EventPress})
	app.Pump(Size{Width: 3, Height: 1})
	p := NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc" {
		t.Fatalf("vertical wheel changed horizontal scroll = %q, want abc", got)
	}

	app.Send(Mouse{Button: MouseWheelRight, EventType: EventPress})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "bcd" {
		t.Fatalf("after wheel right = %q, want bcd", got)
	}

	app.Send(Mouse{Button: MouseWheelLeft, EventType: EventPress})
	app.Pump(Size{Width: 3, Height: 1})
	p = NewPainter(Size{Width: 3, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != "abc" {
		t.Fatalf("after wheel left = %q, want abc", got)
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

func TestScrollViewClampsOffsetWhenChildShrinks(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three", "four", "five")})
	app.Pump(Size{Width: 10, Height: 2})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 2})

	app.UpdateRoot(ScrollView{Child: scrollViewLines("one", "two", "three")})
	app.Pump(Size{Width: 10, Height: 2})

	p := NewPainter(Size{Width: 10, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "t" {
		t.Fatalf("first visible row after shrink = %q, want two", got)
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
	want := ScrollMetrics{ScrollOffset: 1, MaxScrollOffset: 1, ViewportHeight: 2, ViewportWidth: 5, ContentHeight: 3, ContentWidth: 5}
	if got != want {
		t.Fatalf("metrics = %#v, want %#v", got, want)
	}
}

func TestRenderScrollViewScrollCommandsClampOffset(t *testing.T) {
	app := NewApp(ScrollView{Child: scrollViewLines("one", "two", "three", "four", "five")})
	app.Pump(Size{Width: 10, Height: 2})
	r, ok := app.rootRO.(*renderScrollView)
	if !ok {
		t.Fatalf("root render object = %T, want *renderScrollView", app.rootRO)
	}
	r.SetChild(&renderText{Text: "one\ntwo\nthree\nfour\nfive"})
	r.Layout(LayoutContext{}, Constraints{MaxWidth: 10, MaxHeight: 2})

	if !r.ScrollByLines(99) {
		t.Fatal("ScrollByLines should report a changed offset")
	}
	if got := r.State.scrollRow; got != 3 {
		t.Fatalf("scroll row after large line scroll = %d, want 3", got)
	}
	if !r.ScrollByPages(-1) {
		t.Fatal("ScrollByPages should report a changed offset")
	}
	if got := r.State.scrollRow; got != 1 {
		t.Fatalf("scroll row after page up = %d, want 1", got)
	}
	if !r.ScrollToEnd() {
		t.Fatal("ScrollToEnd should report a changed offset")
	}
	if got := r.State.scrollRow; got != 3 {
		t.Fatalf("scroll row after end = %d, want 3", got)
	}
	if !r.ScrollToStart() {
		t.Fatal("ScrollToStart should report a changed offset")
	}
	if got := r.State.scrollRow; got != 0 {
		t.Fatalf("scroll row after start = %d, want 0", got)
	}
	if r.ScrollToStart() {
		t.Fatal("ScrollToStart should report unchanged at start")
	}
}

func scrollViewLines(lines ...string) Widget {
	children := make([]Widget, 0, len(lines))
	for _, line := range lines {
		children = append(children, Text{Value: line})
	}
	return Flex{Axis: Vertical, CrossAxisAlignment: CrossAxisStart, Children: children}
}
