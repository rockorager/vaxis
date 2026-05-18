package ui

import "context"

type Theme struct {
	Text      Style
	Button    ButtonTheme
	TextField TextFieldTheme
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

type TextFieldTheme struct {
	Normal      Style
	Focused     Style
	Placeholder Style
	Cursor      Style
	Padding     Insets
	MinWidth    int
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
		TextField: TextFieldTheme{
			Normal:      Style{Foreground: RGB(238, 238, 238), Background: RGB(32, 32, 32)},
			Focused:     Style{Foreground: RGB(238, 238, 238), Background: RGB(48, 48, 48)},
			Placeholder: Style{Foreground: RGB(128, 128, 128), Background: RGB(32, 32, 32)},
			Cursor:      Style{Foreground: RGB(0, 0, 0), Background: RGB(238, 238, 238)},
			Padding:     Symmetric(1, 0),
			MinWidth:    10,
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
		theme.TextField.Normal.Foreground = fg
		theme.TextField.Focused.Foreground = fg
	}
	if bg != 0 {
		theme.Button.Pressed.Foreground = bg
	}
	if surface, ok := blendColor(bg, fg, 12); ok {
		theme.Button.Normal.Background = surface
		theme.Button.Focused.Background = surface
		theme.TextField.Normal.Background = surface
	}
	if hovered, ok := blendColor(bg, fg, 18); ok {
		theme.Button.Hovered.Background = hovered
		theme.TextField.Focused.Background = hovered
		theme.TextField.Placeholder.Background = hovered
	}
	if fg != 0 {
		theme.Button.Hovered.Foreground = fg
	}
	if pressed, ok := blendColor(bg, fg, 25); ok {
		theme.Button.Pressed.Background = pressed
	}
	if placeholder, ok := blendColor(bg, fg, 50); ok {
		theme.TextField.Placeholder.Foreground = placeholder
	}
	if fg != 0 && bg != 0 {
		theme.TextField.Cursor.Foreground = bg
		theme.TextField.Cursor.Background = fg
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
