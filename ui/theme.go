package ui

import "context"

// Theme contains the default styles used by built-in widgets.
type Theme struct {
	// Text is the default style for Text and RichText.
	Text Style
	// Button contains defaults used by Button.
	Button ButtonTheme
	// ListTile contains defaults used by ListTile.
	ListTile ListTileTheme
	// SegmentedControl contains defaults used by SegmentedControl.
	SegmentedControl SegmentedControlTheme
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

// SegmentedControlTheme contains styling defaults for SegmentedControl.
type SegmentedControlTheme struct {
	// Normal is the default segment style.
	Normal Style
	// Focused is merged over the active segment while the control has focus.
	Focused Style
	// Hovered is merged over the segment under the mouse.
	Hovered Style
	// Selected is merged over the selected segment.
	Selected Style
	// Disabled is merged over disabled segments.
	Disabled Style
	// Separator is used for separators between segments.
	Separator Style
	// Mouse is the pointer shape used while hovering enabled segments.
	Mouse MouseShape
}

// ListTileTheme contains styling and sizing defaults for ListTile.
type ListTileTheme struct {
	// Normal is the default tile style.
	Normal Style
	// Focused is used while the tile has keyboard focus.
	Focused Style
	// Hovered is used while the mouse is over the tile.
	Hovered Style
	// Selected is used when the tile is selected.
	Selected Style
	// Disabled is merged over the tile style when disabled.
	Disabled Style
	// Padding is the default interior spacing.
	Padding Insets
	// Gap is the default spacing between leading, content, and trailing slots.
	Gap int
	// MinHeight is the minimum tile height.
	MinHeight int
	// Mouse is the pointer shape used while hovering an enabled tile.
	Mouse MouseShape
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
	// FocusedThumb paints the thumb when the wrapped scrollable has focus.
	FocusedThumb Style
	// FocusedTrack paints the track when the wrapped scrollable has focus.
	FocusedTrack Style
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
		ListTile: ListTileTheme{
			Normal:    Style{Foreground: RGB(238, 238, 238)},
			Focused:   Style{Foreground: RGB(238, 238, 238), Background: RGB(48, 48, 48)},
			Hovered:   Style{Foreground: RGB(238, 238, 238), Background: RGB(40, 40, 40)},
			Selected:  Style{Foreground: RGB(238, 238, 238), Background: RGB(36, 72, 120)},
			Disabled:  Style{Attribute: AttrDim},
			Padding:   Symmetric(1, 0),
			Gap:       1,
			MinHeight: 1,
			Mouse:     MouseShapeClickable,
		},
		SegmentedControl: SegmentedControlTheme{
			Normal:    Style{Foreground: RGB(238, 238, 238), Background: RGB(32, 32, 32)},
			Focused:   Style{UnderlineStyle: UnderlineSingle},
			Hovered:   Style{Background: RGB(48, 48, 48)},
			Selected:  Style{Foreground: RGB(238, 238, 238), Background: RGB(36, 72, 120)},
			Disabled:  Style{Attribute: AttrDim},
			Separator: Style{Foreground: RGB(128, 128, 128), Background: RGB(32, 32, 32)},
			Mouse:     MouseShapeClickable,
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
			Thumb:        Style{Background: RGB(128, 128, 128)},
			Track:        Style{Background: RGB(48, 48, 48)},
			FocusedThumb: Style{Background: RGB(170, 170, 170)},
			FocusedTrack: Style{Background: RGB(72, 72, 72)},
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
		theme.ListTile.Normal.Foreground = fg
		theme.ListTile.Focused.Foreground = fg
		theme.SegmentedControl.Normal.Foreground = fg
		theme.SegmentedControl.Selected.Foreground = fg
		theme.TextField.Normal.Foreground = fg
		theme.TextField.Focused.Foreground = fg
	}
	if bg != 0 {
		theme.Button.Pressed.Foreground = bg
	}
	if surface, ok := blendColor(bg, fg, 12); ok {
		theme.Button.Normal.Background = surface
		theme.Button.Focused.Background = surface
		theme.ListTile.Focused.Background = surface
		theme.SegmentedControl.Normal.Background = surface
		theme.SegmentedControl.Separator.Background = surface
		theme.TextField.Normal.Background = surface
	}
	if hovered, ok := blendColor(bg, fg, 18); ok {
		theme.Button.Hovered.Background = hovered
		theme.ListTile.Hovered.Background = hovered
		theme.SegmentedControl.Hovered.Background = hovered
		theme.TextField.Focused.Background = hovered
		theme.TextField.Placeholder.Background = hovered
	}
	if selected, ok := blendColor(bg, RGB(80, 150, 255), 30); ok {
		theme.ListTile.Selected.Background = selected
		theme.SegmentedControl.Selected.Background = selected
	}
	if selection, ok := blendColor(bg, RGB(80, 150, 255), 45); ok {
		theme.TextField.Selection.Background = selection
	}
	if track, ok := blendColor(bg, fg, 18); ok {
		theme.Scrollbar.Track.Background = track
		theme.Scrollbar.FocusedTrack.Background = track
	}
	if thumb, ok := blendColor(bg, fg, 45); ok {
		theme.Scrollbar.Thumb.Background = thumb
		theme.Scrollbar.FocusedThumb.Background = thumb
	}
	if focusedTrack, ok := blendColor(bg, fg, 28); ok {
		theme.Scrollbar.FocusedTrack.Background = focusedTrack
	}
	if focusedThumb, ok := blendColor(bg, fg, 62); ok {
		theme.Scrollbar.FocusedThumb.Background = focusedThumb
	}
	if fg != 0 {
		theme.Button.Hovered.Foreground = fg
		theme.ListTile.Hovered.Foreground = fg
		theme.ListTile.Selected.Foreground = fg
		theme.SegmentedControl.Hovered.Foreground = fg
		theme.SegmentedControl.Selected.Foreground = fg
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
