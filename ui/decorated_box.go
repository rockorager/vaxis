package ui

type Decoration struct {
	Style  Style
	Fill   Character
	Border Border
}

type Border struct {
	Style                    Style
	Top, Right, Bottom, Left bool
	Chars                    BorderChars
}

func BorderAll(style Style) Border {
	return Border{Style: style, Top: true, Right: true, Bottom: true, Left: true}
}

func BorderLine(color Color) Border {
	return BorderAll(Style{Foreground: color})
}

type BorderChars struct {
	Horizontal, Vertical    Character
	TopLeft, TopRight       Character
	BottomLeft, BottomRight Character
}

type DecoratedBoxWidget struct {
	Decoration  Decoration
	ChildWidget Widget
}

func DecoratedBox(decoration Decoration, child Widget) Widget {
	return DecoratedBoxWidget{Decoration: decoration, ChildWidget: child}
}

func (w DecoratedBoxWidget) Child() Widget {
	return w.ChildWidget
}

func (w DecoratedBoxWidget) CreateRenderObject(ctx BuildContext) RenderObject {
	return &RenderDecoratedBox{Decoration: w.Decoration}
}

func (w DecoratedBoxWidget) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	r := ro.(*RenderDecoratedBox)
	if r.Decoration != w.Decoration {
		r.Decoration = w.Decoration
		r.MarkNeedsPaint()
	}
}

type RenderDecoratedBox struct {
	SingleChildRenderObject
	Decoration Decoration
}

func (r *RenderDecoratedBox) Layout(ctx LayoutContext, c Constraints) {
	r.SetSize(r.layout(ctx, c, false))
}

func (r *RenderDecoratedBox) DryLayout(ctx LayoutContext, c Constraints) Size {
	return r.layout(ctx, c, true)
}

func (r *RenderDecoratedBox) layout(ctx LayoutContext, c Constraints, dry bool) Size {
	if child := r.Child(); child != nil {
		if dry {
			return c.Constrain(DryLayout(ctx, child, c))
		}
		child.Layout(ctx, c)
		return c.Constrain(child.Base().Size())
	}
	return c.Constrain(Size{})
}

func (r *RenderDecoratedBox) Paint(p *Painter, off Offset) {
	size := r.Size()
	fill := r.Decoration.Fill
	if fill == (Character{}) {
		fill = Character{Grapheme: " ", Width: 1}
	}
	p.Fill(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, Cell{Character: fill, Style: r.Decoration.Style})
	r.paintBorder(p, off, size)
	if child := r.Child(); child != nil {
		child.Paint(p, off)
	}
}

func (r *RenderDecoratedBox) paintBorder(p *Painter, off Offset, size Size) {
	border := r.Decoration.Border
	if size.Width <= 0 || size.Height <= 0 || (!border.Top && !border.Right && !border.Bottom && !border.Left) {
		return
	}
	chars := border.Chars.withDefaults()
	horizontal := Cell{Character: chars.Horizontal, Style: border.Style}
	vertical := Cell{Character: chars.Vertical, Style: border.Style}
	if border.Top {
		for x := 0; x < size.Width; x++ {
			p.DrawCell(Point{X: off.X + x, Y: off.Y}, horizontal)
		}
	}
	if border.Bottom {
		for x := 0; x < size.Width; x++ {
			p.DrawCell(Point{X: off.X + x, Y: off.Y + size.Height - 1}, horizontal)
		}
	}
	if border.Left {
		for y := 0; y < size.Height; y++ {
			p.DrawCell(Point{X: off.X, Y: off.Y + y}, vertical)
		}
	}
	if border.Right {
		for y := 0; y < size.Height; y++ {
			p.DrawCell(Point{X: off.X + size.Width - 1, Y: off.Y + y}, vertical)
		}
	}
	if border.Top && border.Left {
		p.DrawCell(Point(off), Cell{Character: chars.TopLeft, Style: border.Style})
	}
	if border.Top && border.Right {
		p.DrawCell(Point{X: off.X + size.Width - 1, Y: off.Y}, Cell{Character: chars.TopRight, Style: border.Style})
	}
	if border.Bottom && border.Left {
		p.DrawCell(Point{X: off.X, Y: off.Y + size.Height - 1}, Cell{Character: chars.BottomLeft, Style: border.Style})
	}
	if border.Bottom && border.Right {
		p.DrawCell(Point{X: off.X + size.Width - 1, Y: off.Y + size.Height - 1}, Cell{Character: chars.BottomRight, Style: border.Style})
	}
}

func (c BorderChars) withDefaults() BorderChars {
	if c.Horizontal == (Character{}) {
		c.Horizontal = Character{Grapheme: "─", Width: 1}
	}
	if c.Vertical == (Character{}) {
		c.Vertical = Character{Grapheme: "│", Width: 1}
	}
	if c.TopLeft == (Character{}) {
		c.TopLeft = Character{Grapheme: "┌", Width: 1}
	}
	if c.TopRight == (Character{}) {
		c.TopRight = Character{Grapheme: "┐", Width: 1}
	}
	if c.BottomLeft == (Character{}) {
		c.BottomLeft = Character{Grapheme: "└", Width: 1}
	}
	if c.BottomRight == (Character{}) {
		c.BottomRight = Character{Grapheme: "┘", Width: 1}
	}
	return c
}

func (r *RenderDecoratedBox) HitTest(*HitTestResult, Point) bool {
	return false
}
