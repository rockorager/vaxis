package ui

import (
	"context"
	"testing"
)

type fakeColorQuerier struct {
	foreground Color
	background Color
	colors     map[uint8]Color
	queried    []uint8
}

func (q *fakeColorQuerier) QueryForeground(context.Context) Color { return q.foreground }
func (q *fakeColorQuerier) QueryBackground(context.Context) Color { return q.background }
func (q *fakeColorQuerier) QueryColor(_ context.Context, index uint8) Color {
	q.queried = append(q.queried, index)
	return q.colors[index]
}

func TestThemeFromTerminalQueriesExpectedColors(t *testing.T) {
	q := &fakeColorQuerier{
		foreground: RGB(1, 2, 3),
		background: RGB(4, 5, 6),
		colors: map[uint8]Color{
			2: RGB(20, 21, 22),
			4: RGB(40, 41, 42),
			6: RGB(60, 61, 62),
		},
	}
	theme := themeFromTerminal(context.Background(), q)
	if got, want := q.queried, []uint8{1, 2, 3, 4, 5, 6}; len(got) != len(want) {
		t.Fatalf("queried color count = %d, want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("queried colors = %v, want %v", got, want)
			}
		}
	}
	if got := theme.Text.Foreground; got != q.foreground {
		t.Fatalf("text foreground = %#v, want terminal foreground %#v", got, q.foreground)
	}
	if got := theme.Button.Focused.Foreground; got != q.background {
		t.Fatalf("focused button foreground = %#v, want terminal background %#v", got, q.background)
	}
	if got := theme.Button.Normal.Background; got != q.colors[4] {
		t.Fatalf("normal button background = %#v, want terminal index 4 %#v", got, q.colors[4])
	}
	if got := theme.Button.Focused.Background; got != q.colors[6] {
		t.Fatalf("focused button background = %#v, want terminal index 6 %#v", got, q.colors[6])
	}
	if got := theme.Button.Pressed.Background; got != q.colors[2] {
		t.Fatalf("pressed button background = %#v, want terminal index 2 %#v", got, q.colors[2])
	}
}

func TestThemeFromTerminalFallsBackForMissingColors(t *testing.T) {
	fallback := DefaultTheme()
	theme := themeFromTerminal(context.Background(), &fakeColorQuerier{colors: map[uint8]Color{}})
	if theme != fallback {
		t.Fatalf("theme = %#v, want fallback %#v", theme, fallback)
	}
}
