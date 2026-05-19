package ui_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
)

type selectionAreaHarness struct {
	now     time.Time
	backend *fakeBackend
	runner  *ui.Runner
}

func newSelectionAreaHarness(t *testing.T, size ui.Size, root ui.Widget) selectionAreaHarness {
	t.Helper()
	now := time.Unix(10, 0)
	backend := newFakeBackend(size)
	runner := ui.NewRunner(ui.NewApp(root), backend, ui.NewFrameScheduler(time.Second/60))
	runner.Start(now)
	if err := runner.HandleFrame(now); err != nil {
		t.Fatal(err)
	}
	return selectionAreaHarness{now: now, backend: backend, runner: runner}
}

func selectionAreaRoot(child ui.Widget) ui.Widget {
	return ui.SelectionArea{Child: child}
}

func (h selectionAreaHarness) send(ev ui.Event) {
	h.runner.HandleEvent(ev, h.now)
}

func (h selectionAreaHarness) drag(from, to ui.Point) {
	h.send(vaxis.Mouse{Col: from.X, Row: from.Y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: to.X, Row: to.Y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.send(vaxis.Mouse{Col: to.X, Row: to.Y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
}

func (h selectionAreaHarness) click(pt ui.Point) {
	h.send(vaxis.Mouse{Col: pt.X, Row: pt.Y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: pt.X, Row: pt.Y, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
}

func (h selectionAreaHarness) clickN(pt ui.Point, count int) {
	for i := 0; i < count; i++ {
		h.click(pt)
	}
}

func (h selectionAreaHarness) copy() {
	h.send(vaxis.Key{Text: "c", Keycode: 'c', Modifiers: vaxis.ModCtrl})
}

func (h selectionAreaHarness) selectAll() {
	h.send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
}

func (h selectionAreaHarness) frame(t *testing.T) {
	t.Helper()
	if err := h.runner.HandleFrame(h.now); err != nil {
		t.Fatal(err)
	}
}

func (h *selectionAreaHarness) frameAfter(t *testing.T, d time.Duration) {
	t.Helper()
	h.now = h.now.Add(d)
	if err := h.runner.HandleFrame(h.now); err != nil {
		t.Fatal(err)
	}
}

func (h selectionAreaHarness) tab() {
	h.send(vaxis.Key{Keycode: vaxis.KeyTab})
}

func assertCopies(t *testing.T, backend *fakeBackend, want ...string) {
	t.Helper()
	if len(backend.copies) != len(want) {
		t.Fatalf("copies = %#v, want %#v", backend.copies, want)
	}
	for i := range want {
		if backend.copies[i] != want[i] {
			t.Fatalf("copies = %#v, want %#v", backend.copies, want)
		}
	}
}

func TestSelectionAreaSelectsTextWithMouse(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Text{Value: "abcd"}})
	app.Pump(ui.Size{Width: 10, Height: 1})

	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 0).Background; got != want {
		t.Fatalf("selected second cell background = %#v, want %#v", got, want)
	}
	if got := p.Cell(3, 0).Background; got == want {
		t.Fatalf("unselected third cell background = %#v, should not be selection background", got)
	}
}

func TestSelectionAreaCopiesSelectedText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.Text{Value: "abcd"}))

	h.drag(ui.Point{X: 1}, ui.Point{X: 3})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaDoubleClickCopiesWord(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 1}, selectionAreaRoot(ui.Text{Value: "alpha beta"}))

	h.clickN(ui.Point{X: 7}, 2)
	h.copy()
	assertCopies(t, h.backend, "beta")
}

func TestSelectionAreaTripleClickCopiesLine(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 2}, selectionAreaRoot(ui.Text{Value: "alpha beta\ngamma"}))

	h.clickN(ui.Point{X: 2}, 3)
	h.copy()
	assertCopies(t, h.backend, "alpha beta\n")
}

func TestSelectionAreaSelectsRichTextAcrossSpans(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.RichText{Spans: []ui.TextSpan{
		{Text: "ab"},
		{Text: "cd"},
	}}))

	h.drag(ui.Point{X: 1}, ui.Point{X: 3})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaDoubleClickSelectsRichTextWordAcrossSpans(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 1}, selectionAreaRoot(ui.RichText{Spans: []ui.TextSpan{
		{Text: "al"},
		{Text: "pha beta"},
	}}))

	h.clickN(ui.Point{X: 3}, 2)
	h.copy()
	assertCopies(t, h.backend, "alpha")
}

func TestSelectionAreaUsesLocalTextCoordinates(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 1}, ui.Padding(ui.Symmetric(2, 0), selectionAreaRoot(ui.Text{Value: "abcd"})))

	h.drag(ui.Point{X: 3}, ui.Point{X: 5})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaSelectAllCopiesText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.Text{Value: "abcd"}))

	h.click(ui.Point{X: 1})
	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "abcd")
}

func TestSelectionAreaMouseSelectionCopiesClippedVisibleText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 3, Height: 1}, selectionAreaRoot(ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowClip,
	}))

	h.drag(ui.Point{}, ui.Point{X: 3})
	h.copy()
	assertCopies(t, h.backend, "abc")
}

func TestSelectionAreaMouseSelectionCopiesEllipsisVisibleText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 3, Height: 1}, selectionAreaRoot(ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowEllipsis,
		MaxLines: 1,
	}))

	h.drag(ui.Point{}, ui.Point{X: 3})
	h.copy()
	assertCopies(t, h.backend, "ab")
}

func TestSelectionAreaSelectAllCopiesHiddenText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 3, Height: 1}, selectionAreaRoot(ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowEllipsis,
		MaxLines: 1,
	}))

	h.click(ui.Point{})
	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "abcdef")
}

func TestSelectionAreaSelectAllPaintsOverflowVisibleText(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Text{
		Value:    "abcdef",
		Overflow: ui.TextOverflowVisible,
	}})
	app.Pump(ui.Size{Width: 3, Height: 1})

	app.Send(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Send(vaxis.Mouse{Col: 0, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Send(vaxis.Key{Text: "a", Keycode: 'a', Modifiers: vaxis.ModCtrl})
	app.Pump(ui.Size{Width: 3, Height: 1})

	p := ui.NewPainter(ui.Size{Width: 6, Height: 1})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(5, 0).Background; got != want {
		t.Fatalf("overflow selected background = %#v, want %#v", got, want)
	}
}

func TestSelectionAreaMouseSelectionCopiesVisibleMaxLines(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 8, Height: 1}, selectionAreaRoot(ui.Text{
		Value:    "ab\ncd",
		MaxLines: 1,
	}))

	h.drag(ui.Point{}, ui.Point{X: 2})
	h.copy()
	assertCopies(t, h.backend, "ab")
}

func TestSelectionAreaSoftWrapDoesNotCopySyntheticNewline(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 3, Height: 2}, selectionAreaRoot(ui.Text{
		Value:    "abcdef",
		SoftWrap: true,
	}))

	h.drag(ui.Point{}, ui.Point{X: 3, Y: 1})
	h.copy()
	assertCopies(t, h.backend, "abcdef")
}

func TestSelectionAreaStaticTextSelectsEndOfLineWhenPointerBelow(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 3}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "line 1"},
			ui.Text{Value: "line 2"},
		},
	}))

	h.drag(ui.Point{Y: 1}, ui.Point{X: 2, Y: 2})
	h.copy()
	assertCopies(t, h.backend, "line 2")
}

func TestSelectionAreaMouseNoButtonMotionEndsDrag(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.Text{Value: "abcd"}))

	h.send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseNoButton, EventType: vaxis.EventMotion})
	h.send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaCapturedReleaseOutsideStopsDrag(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 1}, ui.Flex{
		Axis: ui.Horizontal,
		ChildrenWidget: []ui.Widget{
			ui.SizedBox{Width: 4, Height: 1, Child: selectionAreaRoot(ui.Text{Value: "abcd"})},
			ui.Text{Value: "outside"},
		},
	})

	h.send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: 3, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.send(vaxis.Mouse{Col: 8, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	h.send(vaxis.Mouse{Col: 4, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaSelectsAcrossTextWidgets(t *testing.T) {
	app := ui.NewApp(ui.SelectionArea{Child: ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 1, Row: 0, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	app.Pump(ui.Size{Width: 10, Height: 2})
	app.Send(vaxis.Mouse{Col: 2, Row: 1, Button: vaxis.MouseLeftButton, EventType: vaxis.EventRelease})
	app.Pump(ui.Size{Width: 10, Height: 2})

	p := ui.NewPainter(ui.Size{Width: 10, Height: 2})
	app.Paint(p)
	want := ui.DefaultTheme().TextField.Selection.Background
	if got := p.Cell(1, 0).Background; got != want {
		t.Fatalf("selected first line background = %#v, want %#v", got, want)
	}
	if got := p.Cell(1, 1).Background; got != want {
		t.Fatalf("selected second line background = %#v, want %#v", got, want)
	}
	if got := p.Cell(2, 1).Background; got == want {
		t.Fatalf("unselected second line cell background = %#v, should not be selection background", got)
	}
}

func TestSelectionAreaCopiesAcrossTextWidgets(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 2}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}))

	h.drag(ui.Point{X: 1}, ui.Point{X: 2, Y: 1})
	h.copy()
	assertCopies(t, h.backend, "bcd\nef")
}

func TestSelectionAreaSelectsAcrossTextWidgetsInReverse(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 2}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "abcd"},
		ui.Text{Value: "efgh"},
	}}))

	h.drag(ui.Point{X: 2, Y: 1}, ui.Point{X: 1})
	h.copy()
	assertCopies(t, h.backend, "bcd\nef")
}

func TestSelectionAreaSelectAllCopiesAllTextWidgets(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 2}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "ab"},
		ui.Text{Value: "cd"},
	}}))

	h.click(ui.Point{})
	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "ab\ncd")
}

func TestSelectionAreaMouseSelectionAcrossScrollViewIncludesHiddenRows(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 4}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "before"},
			ui.SizedBox{Width: 10, Height: 2, Child: ui.ScrollView{Child: ui.Flex{
				Axis:               ui.Vertical,
				CrossAxisAlignment: ui.CrossAxisStart,
				ChildrenWidget: []ui.Widget{
					ui.Text{Value: "one"},
					ui.Text{Value: "two"},
					ui.Text{Value: "three"},
					ui.Text{Value: "four"},
				},
			}}},
			ui.Text{Value: "after"},
		},
	}))

	h.send(vaxis.Mouse{Col: 0, Row: 1, Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress})
	h.frame(t)
	h.drag(ui.Point{}, ui.Point{X: 5, Y: 3})
	h.copy()
	assertCopies(t, h.backend, "before\none\ntwo\nthree\nfour\nafter")
}

func TestSelectionAreaMouseSelectionFromScrollViewSkipsHiddenRows(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 4}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "before"},
			ui.SizedBox{Width: 10, Height: 2, Child: ui.ScrollView{Child: ui.Flex{
				Axis:               ui.Vertical,
				CrossAxisAlignment: ui.CrossAxisStart,
				ChildrenWidget: []ui.Widget{
					ui.Text{Value: "one"},
					ui.Text{Value: "two"},
					ui.Text{Value: "three"},
					ui.Text{Value: "four"},
				},
			}}},
			ui.Text{Value: "after"},
		},
	}))

	h.send(vaxis.Mouse{Col: 0, Row: 1, Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress})
	h.frame(t)
	h.drag(ui.Point{Y: 1}, ui.Point{X: 5, Y: 3})
	h.copy()
	assertCopies(t, h.backend, "two\nthree\nafter")
}

func TestSelectionAreaAutoScrollsSelectionNearScrollViewEdge(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 3}, selectionAreaRoot(ui.SizedBox{Width: 12, Height: 3, Child: ui.ScrollView{Child: scrollSelectionLines(30)}}))
	h.send(vaxis.Resize{Cols: 12, Rows: 3, XPixel: 120, YPixel: 300})

	h.send(vaxis.Mouse{Col: 0, Row: 2, XPixel: 5, YPixel: 290, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.frameAfter(t, time.Second/60)
	h.frameAfter(t, time.Second)
	h.copy()

	if len(h.backend.copies) != 1 || !strings.Contains(h.backend.copies[0], "row 7") {
		t.Fatalf("autoscroll copy = %#v, want selection extended after scrolling", h.backend.copies)
	}
}

func TestSelectionAreaAutoScrollSchedulesNextFrameBeforeWholeLine(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 3}, selectionAreaRoot(ui.SizedBox{Width: 12, Height: 3, Child: ui.ScrollView{Child: scrollSelectionLines(30)}}))
	h.send(vaxis.Resize{Cols: 12, Rows: 3, XPixel: 120, YPixel: 300})

	h.send(vaxis.Mouse{Col: 0, Row: 2, XPixel: 5, YPixel: 260, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.frameAfter(t, time.Second/60)
	if _, ok := h.runner.NextFrame(); !ok {
		t.Fatal("expected active autoscroll to schedule another frame")
	}
}

func TestSelectionAreaAutoScrollSelectsEndOfLineWhenPointerBelowViewport(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 3}, selectionAreaRoot(ui.SizedBox{Width: 12, Height: 3, Child: ui.ScrollView{Child: scrollSelectionLines(30)}}))
	h.send(vaxis.Resize{Cols: 12, Rows: 3, XPixel: 120, YPixel: 300})

	h.send(vaxis.Mouse{Col: 0, Row: 1, XPixel: 5, YPixel: 150, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: 2, Row: 4, XPixel: 25, YPixel: 450, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.frameAfter(t, time.Second/60)
	h.frameAfter(t, 500*time.Millisecond)
	h.copy()

	if len(h.backend.copies) != 1 {
		t.Fatalf("copies = %#v, want one copy", h.backend.copies)
	}
	lines := strings.Split(strings.TrimSuffix(h.backend.copies[0], "\n"), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1], "row ") {
		t.Fatalf("autoscroll copy = %#v, want row lines", h.backend.copies)
	}
	if got := lines[len(lines)-1]; len(got) < len("row 10") {
		t.Fatalf("last selected line = %q, want full line", got)
	}
}

func TestSelectionAreaAutoScrollSelectionContinuesIntoFollowingText(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 5}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.SizedBox{Width: 12, Height: 3, Child: ui.ScrollView{Child: scrollSelectionLines(30)}},
			ui.Text{Value: "after"},
		},
	}))
	h.send(vaxis.Resize{Cols: 12, Rows: 5, XPixel: 120, YPixel: 500})

	h.send(vaxis.Mouse{Col: 0, Row: 1, XPixel: 5, YPixel: 150, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: 2, Row: 4, XPixel: 25, YPixel: 450, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.frameAfter(t, time.Second/60)
	h.frameAfter(t, time.Second)
	h.send(vaxis.Mouse{Col: 2, Row: 3, XPixel: 25, YPixel: 350, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.copy()

	if len(h.backend.copies) != 1 || !strings.Contains(h.backend.copies[0], "\naf") {
		t.Fatalf("copy after moving into following text = %#v, want following text included", h.backend.copies)
	}
}

func TestSelectionAreaAutoScrollsSelectionAboveScrollViewEdge(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 12, Height: 3}, selectionAreaRoot(ui.SizedBox{Width: 12, Height: 3, Child: ui.ScrollView{Child: scrollSelectionLines(30)}}))
	h.send(vaxis.Resize{Cols: 12, Rows: 3, XPixel: 120, YPixel: 300})
	for i := 0; i < 27; i++ {
		h.send(vaxis.Mouse{Col: 0, Row: 1, Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress})
		h.frame(t)
	}

	h.send(vaxis.Mouse{Col: 0, Row: 1, XPixel: 5, YPixel: 150, Button: vaxis.MouseLeftButton, EventType: vaxis.EventPress})
	h.send(vaxis.Mouse{Col: 2, Row: -2, XPixel: 25, YPixel: -200, Button: vaxis.MouseLeftButton, EventType: vaxis.EventMotion})
	h.frameAfter(t, time.Second/60)
	h.frameAfter(t, time.Second)
	h.copy()

	if len(h.backend.copies) != 1 || !strings.Contains(h.backend.copies[0], "row 1") {
		t.Fatalf("reverse autoscroll copy = %#v, want selection extended before scrolling", h.backend.copies)
	}
}

func TestSelectionAreaMouseSelectionAcrossNestedScrollViewIncludesHiddenRows(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 5}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "before"},
			ui.SizedBox{Width: 10, Height: 3, Child: ui.ScrollView{Child: ui.Flex{
				Axis:               ui.Vertical,
				CrossAxisAlignment: ui.CrossAxisStart,
				ChildrenWidget: []ui.Widget{
					ui.Text{Value: "top"},
					ui.SizedBox{Width: 10, Height: 2, Child: ui.ScrollView{Child: ui.Flex{
						Axis:               ui.Vertical,
						CrossAxisAlignment: ui.CrossAxisStart,
						ChildrenWidget: []ui.Widget{
							ui.Text{Value: "inner 1"},
							ui.Text{Value: "inner 2"},
							ui.Text{Value: "inner 3"},
							ui.Text{Value: "inner 4"},
						},
					}}},
					ui.Text{Value: "bottom"},
				},
			}}},
			ui.Text{Value: "after"},
		},
	}))

	h.send(vaxis.Mouse{Col: 0, Row: 2, Button: vaxis.MouseWheelDown, EventType: vaxis.EventPress})
	h.frame(t)
	h.drag(ui.Point{}, ui.Point{X: 5, Y: 4})
	h.copy()
	assertCopies(t, h.backend, "before\ntop\ninner 1\ninner 2\ninner 3\ninner 4\nbottom\nafter")
}

func TestSelectionAreaSelectAllIncludesHiddenScrollViewRows(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 2}, selectionAreaRoot(ui.SizedBox{Width: 10, Height: 2, Child: ui.ScrollView{Child: ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "one"},
			ui.Text{Value: "two"},
			ui.Text{Value: "three"},
		},
	}}}))

	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "one\ntwo\nthree")
}

func TestSelectionAreaMouseSelectionAcrossCustomScrollViewIncludesHiddenRows(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 4}, selectionAreaRoot(ui.Flex{
		Axis:               ui.Vertical,
		CrossAxisAlignment: ui.CrossAxisStart,
		ChildrenWidget: []ui.Widget{
			ui.Text{Value: "before"},
			ui.SizedBox{Width: 10, Height: 2, Child: ui.CustomScrollView{Slivers: []ui.Widget{
				ui.SliverList{ChildrenWidget: []ui.Widget{
					ui.Text{Value: "one"},
					ui.Text{Value: "two"},
					ui.Text{Value: "three"},
					ui.Text{Value: "four"},
				}},
			}}},
			ui.Text{Value: "after"},
		},
	}))

	h.drag(ui.Point{}, ui.Point{X: 5, Y: 3})
	h.copy()
	assertCopies(t, h.backend, "before\none\ntwo\nthree\nfour\nafter")
}

func scrollSelectionLines(n int) ui.Widget {
	children := make([]ui.Widget, 0, n)
	for i := 1; i <= n; i++ {
		children = append(children, ui.Text{Value: "row " + strconv.Itoa(i)})
	}
	return ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: children}
}

func TestSelectionContainerDisabledExcludesSubtreeFromCopy(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 16, Height: 3}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "aa"},
		ui.SelectionContainer{Disabled: true, Child: ui.Text{Value: "bb"}},
		ui.Text{Value: "cc"},
	}}))

	h.drag(ui.Point{}, ui.Point{X: 2, Y: 2})
	h.copy()
	assertCopies(t, h.backend, "aa\ncc")
}

func TestSelectionContainerDisabledDoesNotStartSelection(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.SelectionContainer{
		Disabled: true,
		Child:    ui.Text{Value: "abcd"},
	}))

	h.drag(ui.Point{}, ui.Point{X: 4})
	h.copy()
	assertCopies(t, h.backend)
}

func TestSelectionContainerEnabledIsTransparent(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 10, Height: 1}, selectionAreaRoot(ui.SelectionContainer{
		Child: ui.Text{Value: "abcd"},
	}))

	h.drag(ui.Point{X: 1}, ui.Point{X: 3})
	h.copy()
	assertCopies(t, h.backend, "bc")
}

func TestSelectionAreaSelectAllSkipsTextFieldContents(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 3}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}))

	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "before\nafter")
}

func TestSelectionAreaFocusedTextFieldHandlesSelectAllAndCopy(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 3}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}))

	h.tab()
	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "field")
}

func TestSelectionAreaMousePressInTextFieldDoesNotStartOuterSelection(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 3}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextField{Value: "field", MinWidth: 10},
		ui.Text{Value: "after"},
	}}))

	h.drag(ui.Point{X: 2, Y: 1}, ui.Point{X: 5, Y: 2})
	h.copy()
	assertCopies(t, h.backend)
}

func TestSelectionAreaSelectAllSkipsTextAreaContents(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 4}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}))

	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "before\nafter")
}

func TestSelectionAreaFocusedTextAreaHandlesSelectAllAndCopy(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 4}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}))

	h.tab()
	h.selectAll()
	h.copy()
	assertCopies(t, h.backend, "area")
}

func TestSelectionAreaMousePressInTextAreaSelectsTextArea(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 4}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}))

	h.drag(ui.Point{X: 3, Y: 1}, ui.Point{X: 1, Y: 1})
	h.copy()
	assertCopies(t, h.backend, "ar")
}

func TestSelectionAreaReadOnlyTextStillSelectableAroundTextArea(t *testing.T) {
	h := newSelectionAreaHarness(t, ui.Size{Width: 20, Height: 4}, selectionAreaRoot(ui.Flex{Axis: ui.Vertical, CrossAxisAlignment: ui.CrossAxisStart, ChildrenWidget: []ui.Widget{
		ui.Text{Value: "before"},
		ui.TextArea{Value: "area", MinWidth: 10, MinHeight: 1},
		ui.Text{Value: "after"},
	}}))

	h.drag(ui.Point{}, ui.Point{X: 5})
	h.copy()
	assertCopies(t, h.backend, "befor")
}
