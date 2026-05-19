package ui

import (
	"context"
	"testing"
)

type fakeColorQuerier struct {
	foreground Color
	background Color
}

func (q *fakeColorQuerier) QueryForeground(context.Context) Color {
	return q.foreground
}

func (q *fakeColorQuerier) QueryBackground(context.Context) Color {
	return q.background
}

func TestThemeFromTerminalDerivesRGBTheme(t *testing.T) {
	q := &fakeColorQuerier{
		foreground: RGB(200, 210, 220),
		background: RGB(10, 20, 30),
	}
	theme := themeFromTerminal(context.Background(), q)
	if got := theme.Text.Foreground; got != q.foreground {
		t.Fatalf("text foreground = %#v, want terminal foreground %#v", got, q.foreground)
	}
	if got := theme.Button.Focused.Foreground; got != q.foreground {
		t.Fatalf("focused button foreground = %#v, want terminal foreground", got)
	}
	if got := theme.Button.Normal.Background; got != RGB(32, 42, 52) {
		t.Fatalf("normal button background = %#v, want blended surface", got)
	}
	if got := theme.Button.Focused.Background; got != RGB(32, 42, 52) {
		t.Fatalf("focused button background = %#v, want blended surface", got)
	}
	if got := theme.Button.Hovered.Background; got != RGB(44, 54, 64) {
		t.Fatalf("hovered button background = %#v, want blended hover surface", got)
	}
	if got := theme.Button.Pressed.Background; got != RGB(57, 67, 77) {
		t.Fatalf("pressed button background = %#v, want blended pressed background", got)
	}
	if got := theme.Scrollbar.Track.Background; got != RGB(44, 54, 64) {
		t.Fatalf("scrollbar track background = %#v, want blended track background", got)
	}
	if got := theme.Scrollbar.Thumb.Background; got != RGB(95, 105, 115) {
		t.Fatalf("scrollbar thumb background = %#v, want blended thumb background", got)
	}
}

func TestThemeFromTerminalFallsBackForMissingColors(t *testing.T) {
	fallback := DefaultTheme()
	theme := themeFromTerminal(context.Background(), &fakeColorQuerier{})
	if theme != fallback {
		t.Fatalf("theme = %#v, want fallback %#v", theme, fallback)
	}
}
