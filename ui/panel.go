package ui

type PanelStyle struct {
	Background Color
	Border     Border
	Padding    Insets
}

func Panel(style PanelStyle, child Widget) Widget {
	if style.Padding != (Insets{}) {
		child = Padding(style.Padding, child)
	}
	return DecoratedBox(Decoration{
		Style:  Style{Background: style.Background},
		Border: style.Border,
	}, child)
}
