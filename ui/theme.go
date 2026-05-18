package ui

import "context"

type Theme struct {
	Text   Style
	Button ButtonTheme
}

type ButtonTheme struct {
	Normal  Style
	Focused Style
	Pressed Style
	Mouse   MouseShape
}

func DefaultTheme() Theme {
	return Theme{
		Text: Style{Foreground: RGB(238, 238, 238)},
		Button: ButtonTheme{
			Normal:  Style{Foreground: RGB(238, 238, 238), Background: RGB(48, 48, 48)},
			Focused: Style{Foreground: RGB(0, 0, 0), Background: RGB(0, 255, 255)},
			Pressed: Style{Foreground: RGB(0, 0, 0), Background: RGB(0, 255, 0)},
			Mouse:   MouseShapeClickable,
		},
	}
}

type terminalColorQuerier interface {
	QueryForeground(context.Context) Color
	QueryBackground(context.Context) Color
	QueryColor(context.Context, uint8) Color
}

func themeFromTerminal(ctx context.Context, q terminalColorQuerier) Theme {
	theme := DefaultTheme()
	if q == nil {
		return theme
	}
	fg := q.QueryForeground(ctx)
	bg := q.QueryBackground(ctx)
	colors := [7]Color{}
	for i := uint8(1); i <= 6; i++ {
		colors[i] = q.QueryColor(ctx, i)
	}
	if fg != 0 {
		theme.Text.Foreground = fg
		theme.Button.Normal.Foreground = fg
	}
	if bg != 0 {
		theme.Button.Focused.Foreground = bg
		theme.Button.Pressed.Foreground = bg
	}
	if colors[4] != 0 {
		theme.Button.Normal.Background = colors[4]
	}
	if colors[6] != 0 {
		theme.Button.Focused.Background = colors[6]
	}
	if colors[2] != 0 {
		theme.Button.Pressed.Background = colors[2]
	}
	return theme
}
