package ui

import "context"

// Theme contains the default styles used by built-in widgets.
type Theme struct {
	// Text is the default style for Text and RichText.
	Text Style
	// Button contains defaults used by Button.
	Button ButtonTheme
	// TextField contains defaults used by TextField and TextArea.
	TextField TextFieldTheme
	// Scrollbar contains defaults used by Scrollbar.
	Scrollbar ScrollbarTheme
}

// ButtonTheme contains styling and sizing defaults for Button.
type ButtonTheme struct {
	// Normal is the default button style.
	Normal Style
	// Focused is used while the button has keyboard focus.
	Focused Style
	// Hovered is used while the mouse is over the button.
	Hovered Style
	// Pressed is reserved for pressed button states.
	Pressed Style
	// Padding is the default interior spacing.
	Padding Insets
	// MinWidth is the default minimum button width.
	MinWidth int
	// Mouse is the pointer shape used while hovering the button.
	Mouse MouseShape
	// FocusLeft is painted before the label while focused.
	FocusLeft Character
	// FocusRight is painted after the label while focused.
	FocusRight Character
}

// TextFieldTheme contains styling and sizing defaults for TextField and TextArea.
type TextFieldTheme struct {
	// Normal is the default text input style.
	Normal Style
	// Focused is used while the input has keyboard focus.
	Focused Style
	// Placeholder is merged over the current input style for placeholder text.
	Placeholder Style
	// Cursor is reserved for software cursor painting.
	Cursor Style
	// Selection is merged over selected text.
	Selection Style
	// Padding is the default interior spacing.
	Padding Insets
	// MinWidth is the default minimum input width.
	MinWidth int
}

// ScrollbarTheme contains styling defaults for Scrollbar.
type ScrollbarTheme struct {
	// Thumb paints the draggable scrollbar thumb.
	Thumb Style
	// Track paints the scrollbar track.
	Track Style
}

// DefaultTheme returns the built-in fallback theme.
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
			Selection:   Style{Background: RGB(48, 96, 160)},
			Padding:     Symmetric(1, 0),
			MinWidth:    10,
		},
		Scrollbar: ScrollbarTheme{
			Thumb: Style{Background: RGB(128, 128, 128)},
			Track: Style{Background: RGB(48, 48, 48)},
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
	if selection, ok := blendColor(bg, RGB(80, 150, 255), 45); ok {
		theme.TextField.Selection.Background = selection
	}
	if track, ok := blendColor(bg, fg, 18); ok {
		theme.Scrollbar.Track.Background = track
	}
	if thumb, ok := blendColor(bg, fg, 45); ok {
		theme.Scrollbar.Thumb.Background = thumb
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
