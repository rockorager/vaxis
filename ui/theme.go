package ui

import "context"

type Theme struct {
	Text   Style
	Button ButtonTheme
}

type ButtonTheme struct {
	Normal     Style
	Focused    Style
	Hovered    Style
	Pressed    Style
	Padding    Insets
	MinWidth   int
	Mouse      MouseShape
	FocusLeft  Character
	FocusRight Character
}

func DefaultTheme() Theme {
	return Theme{
		Text: Style{Foreground: RGB(238, 238, 238)},
		Button: ButtonTheme{
			Normal:     Style{Foreground: RGB(238, 238, 238), Background: RGB(48, 48, 48)},
			Focused:    Style{Foreground: RGB(238, 238, 238), Background: RGB(48, 48, 48)},
			Hovered:    Style{Foreground: RGB(238, 238, 238), Background: RGB(64, 64, 64)},
			Pressed:    Style{Foreground: RGB(0, 0, 0), Background: RGB(0, 255, 0)},
			Padding:    Symmetric(1, 0),
			MinWidth:   5,
			Mouse:      MouseShapeClickable,
			FocusLeft:  Character{Grapheme: "[", Width: 1},
			FocusRight: Character{Grapheme: "]", Width: 1},
		},
	}
}

type terminalColorQuerier interface {
	QueryForeground(context.Context) Color
	QueryBackground(context.Context) Color
}

func themeFromTerminal(ctx context.Context, q terminalColorQuerier) Theme {
	theme := DefaultTheme()
	if q == nil {
		return theme
	}
	fg := q.QueryForeground(ctx)
	bg := q.QueryBackground(ctx)
	if fg != 0 {
		theme.Text.Foreground = fg
		theme.Button.Normal.Foreground = fg
		theme.Button.Focused.Foreground = fg
	}
	if bg != 0 {
		theme.Button.Pressed.Foreground = bg
	}
	if surface, ok := blendColor(bg, fg, 12); ok {
		theme.Button.Normal.Background = surface
		theme.Button.Focused.Background = surface
	}
	if hovered, ok := blendColor(bg, fg, 18); ok {
		theme.Button.Hovered.Background = hovered
	}
	if fg != 0 {
		theme.Button.Hovered.Foreground = fg
	}
	if pressed, ok := blendColor(bg, fg, 25); ok {
		theme.Button.Pressed.Background = pressed
	}
	return theme
}

func blendColor(a, b Color, percentB int) (Color, bool) {
	ap := a.Params()
	bp := b.Params()
	if len(ap) != 3 || len(bp) != 3 {
		return 0, false
	}
	percentA := 100 - percentB
	return RGB(
		uint8((int(ap[0])*percentA+int(bp[0])*percentB)/100),
		uint8((int(ap[1])*percentA+int(bp[1])*percentB)/100),
		uint8((int(ap[2])*percentA+int(bp[2])*percentB)/100),
	), true
}
