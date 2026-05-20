package ui

import (
	"math"
	"testing"
)

func TestScrollbarPaintsVerticalThumbForOverflow(t *testing.T) {
	thumbStyle := Style{Background: RGB(200, 200, 200)}
	trackStyle := Style{Background: RGB(30, 30, 30)}
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	)}, ThumbStyle: thumbStyle, TrackStyle: trackStyle})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 0); got.Grapheme != " " || got.Background != thumbStyle.Background {
		t.Fatalf("top thumb cell = %#v, want filled thumb", got)
	}
	if got := p.Cell(9, 1); got.Grapheme != "▃" || got.Foreground != trackStyle.Background || got.Background != thumbStyle.Background {
		t.Fatalf("partial thumb cell = %#v, want upper thumb and lower track", got)
	}
}

func TestScrollbarDefaultStylesDistinguishThumbAndTrack(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	)}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	thumb := p.Cell(9, 0)
	track := p.Cell(9, 3)
	if thumb.Background == 0 || track.Background == 0 || thumb.Background == track.Background {
		t.Fatalf("default thumb/track backgrounds = %#v/%#v, want distinct non-zero colors", thumb.Background, track.Background)
	}
}

func TestScrollbarUsesFocusedStylesWhenScrollViewFocused(t *testing.T) {
	theme := DefaultTheme()
	theme.Scrollbar.Track = Style{Background: RGB(10, 10, 10)}
	theme.Scrollbar.FocusedTrack = Style{Background: RGB(20, 20, 20)}
	app := NewApp(Column(
		Button{Label: "before"},
		SizedBox{Width: 10, Height: 4, Child: Scrollbar{Child: ScrollView{Child: scrollViewLines(
			"one", "two", "three", "four", "five", "six", "seven", "eight",
		)}}},
	), WithTheme(theme))
	app.Pump(Size{Width: 10, Height: 5})

	p := NewPainter(Size{Width: 10, Height: 5})
	app.Paint(p)
	if got := p.Cell(9, 4).Background; got != theme.Scrollbar.Track.Background {
		t.Fatalf("unfocused track background = %#v, want %#v", got, theme.Scrollbar.Track.Background)
	}

	app.focusNext()
	app.Pump(Size{Width: 10, Height: 5})
	p = NewPainter(Size{Width: 10, Height: 5})
	app.Paint(p)
	if got := p.Cell(9, 4).Background; got != theme.Scrollbar.FocusedTrack.Background {
		t.Fatalf("focused track background = %#v, want %#v", got, theme.Scrollbar.FocusedTrack.Background)
	}
}

func TestScrollbarMovesThumbWithScrollOffset(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 3).Grapheme; got != " " {
		t.Fatalf("bottom thumb cell = %q, want filled space", got)
	}
}

func TestScrollbarPaintsHorizontalThumbForOverflow(t *testing.T) {
	thumbStyle := Style{Background: RGB(200, 200, 200)}
	trackStyle := Style{Background: RGB(30, 30, 30)}
	app := NewApp(SizedBox{Width: 4, Height: 2, Child: Scrollbar{
		Axis:       ScrollHorizontal,
		ThumbStyle: thumbStyle,
		TrackStyle: trackStyle,
		Child:      ScrollView{Axis: ScrollHorizontal, Child: Text{Value: "abcdefghij"}},
	}})
	app.Pump(Size{Width: 4, Height: 2})

	p := NewPainter(Size{Width: 4, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 1); got.Grapheme != "▄" || got.Foreground != thumbStyle.Background {
		t.Fatalf("left thumb cell = %#v, want lower-half thumb", got)
	}
	if got := p.Cell(3, 1); got.Grapheme != "▄" || got.Foreground != trackStyle.Background {
		t.Fatalf("right track cell = %#v, want lower-half track", got)
	}
}

func TestScrollbarHorizontalKeepsUpperBackground(t *testing.T) {
	baseStyle := Style{Background: RGB(9, 9, 9)}
	thumbStyle := Style{Background: RGB(200, 200, 200)}
	app := NewApp(SizedBox{Width: 4, Height: 2, Child: Scrollbar{
		Axis:       ScrollHorizontal,
		ThumbStyle: thumbStyle,
		Child: ScrollView{
			Axis: ScrollHorizontal,
			Child: ConstrainedBox{Constraints: Constraints{MinHeight: 2}, Child: DecoratedBox(
				Decoration{Style: baseStyle},
				Text{Value: "abcdefghij"},
			)},
		},
	}})
	app.Pump(Size{Width: 4, Height: 2})

	p := NewPainter(Size{Width: 4, Height: 2})
	app.Paint(p)
	if got := p.Cell(0, 1); got.Grapheme != "▄" || got.Foreground != thumbStyle.Background || got.Background != baseStyle.Background {
		t.Fatalf("horizontal scrollbar cell = %#v, want lower thumb over base background", got)
	}
}

func TestScrollbarClickHorizontalTrackPagesChild(t *testing.T) {
	app := NewApp(SizedBox{Width: 4, Height: 2, Child: Scrollbar{
		Axis:  ScrollHorizontal,
		Child: ScrollView{Axis: ScrollHorizontal, Child: Text{Value: "abcdefghij"}},
	}})
	app.Pump(Size{Width: 4, Height: 2})

	app.Send(Mouse{Col: 3, Row: 1, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 4, Height: 2})

	p := NewPainter(Size{Width: 4, Height: 2})
	app.Paint(p)
	if got := debugRenderedText(p); got[:4] != "efgh" {
		t.Fatalf("visible text after horizontal track click = %q, want efgh prefix", got)
	}
}

func TestScrollbarHidesWhenChildDoesNotOverflow(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines("one", "two")}})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(9, 0).Grapheme; got != "" {
		t.Fatalf("scrollbar cell without overflow = %q, want empty", got)
	}
}

func TestScrollbarThumbUsesViewportRatio(t *testing.T) {
	thumb := scrollbarThumb(ScrollVertical, ScrollMetrics{
		ScrollOffset:    2,
		MaxScrollOffset: 6,
		ViewportHeight:  4,
		ContentHeight:   10,
	})
	if math.Abs(thumb.Top-0.8) > 0.001 || math.Abs(thumb.Height-1.6) > 0.001 {
		t.Fatalf("thumb = top %.2f height %.2f, want top 0.80 height 1.60", thumb.Top, thumb.Height)
	}
}

func TestScrollbarClickTrackPagesChild(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})

	app.Send(Mouse{Col: 9, Row: 3, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "f" {
		t.Fatalf("first visible row after track click = %q, want five", got)
	}
}

func TestScrollbarClickTrackAboveThumbPagesUp(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})
	app.Send(Key{Keycode: KeyEnd})
	app.Pump(Size{Width: 10, Height: 4})

	app.Send(Mouse{Col: 9, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after track page up = %q, want one", got)
	}
}

func TestScrollbarDragThumbScrollsWithFractionalMouse(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})
	app.Send(Resize{Cols: 10, Rows: 4, XPixel: 100, YPixel: 400})

	app.Send(Mouse{Col: 9, Row: 0, XPixel: 95, YPixel: 50, Button: MouseLeftButton, EventType: EventPress})
	app.Send(Mouse{Col: 9, Row: 2, XPixel: 95, YPixel: 250, Button: MouseLeftButton, EventType: EventMotion})
	app.Send(Mouse{Col: 9, Row: 2, XPixel: 95, YPixel: 250, Button: MouseLeftButton, EventType: EventRelease})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "f" {
		t.Fatalf("first visible row after thumb drag = %q, want five", got)
	}
}

func TestScrollbarCapturedDragUsesVerticalPositionOutsideColumn(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})

	app.Send(Mouse{Col: 9, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	app.Send(Mouse{Col: 0, Row: 2, Button: MouseLeftButton, EventType: EventMotion})
	app.Send(Mouse{Col: 0, Row: 2, Button: MouseLeftButton, EventType: EventRelease})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "f" {
		t.Fatalf("first visible row after off-column thumb drag = %q, want five", got)
	}
}

func TestScrollbarCapturedReleaseOutsideStopsDrag(t *testing.T) {
	app := NewApp(Scrollbar{Child: ScrollView{Child: scrollViewLines(
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	)}})
	app.Pump(Size{Width: 10, Height: 4})

	app.Send(Mouse{Col: 9, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	app.Send(Mouse{Col: 9, Row: 2, Button: MouseLeftButton, EventType: EventRelease})
	app.Send(Mouse{Col: 9, Row: 3, Button: MouseLeftButton, EventType: EventMotion})
	app.Pump(Size{Width: 10, Height: 4})

	p := NewPainter(Size{Width: 10, Height: 4})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "o" {
		t.Fatalf("first visible row after released drag motion = %q, want one", got)
	}
}
