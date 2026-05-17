package ui

type Theme struct {
	Text   Style
	Button ButtonTheme
}

type ButtonTheme struct {
	Normal  Style
	Focused Style
	Pressed Style
}

func DefaultTheme() Theme { return Theme{} }
