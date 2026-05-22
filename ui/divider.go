package ui

// Divider paints a one-cell horizontal or vertical separator.
//
// The zero value paints a horizontal divider using the box-drawing horizontal
// line. A vertical divider uses the box-drawing vertical line.
type Divider struct {
	// Axis controls the divider orientation.
	Axis Axis
	// Character overrides the line character.
	Character Character
	// Style overrides Theme foreground when non-zero fields are set.
	Style Style
}

func (w Divider) CreateRenderObject(ctx BuildContext) RenderObject {
	if w.Style == (Style{}) {
		w.Style = textStyle(MustDepend[Theme](ctx))
	}
	return &renderDivider{Axis: w.Axis, Character: dividerCharacter(w), Style: w.Style}
}

func (w Divider) UpdateRenderObject(ctx BuildContext, ro RenderObject) {
	if w.Style == (Style{}) {
		w.Style = textStyle(MustDepend[Theme](ctx))
	}
	r := ro.(*renderDivider)
	next := dividerCharacter(w)
	if r.Axis != w.Axis || r.Character != next || r.Style != w.Style {
		r.Axis = w.Axis
		r.Character = next
		r.Style = w.Style
		r.MarkNeedsLayout()
	}
}

type renderDivider struct {
	LeafRenderObject
	Axis      Axis
	Character Character
	Style     Style
}

func (r *renderDivider) Layout(_ LayoutContext, c Constraints) {
	r.SetSize(r.size(c))
}

func (r *renderDivider) DryLayout(_ LayoutContext, c Constraints) Size {
	return r.size(c)
}

func (r *renderDivider) size(c Constraints) Size {
	if r.Axis == Vertical {
		height := 1
		if c.HasBoundedHeight() {
			height = c.MaxHeight
		}
		return c.Constrain(Size{Width: 1, Height: height})
	}
	width := 1
	if c.HasBoundedWidth() {
		width = c.MaxWidth
	}
	return c.Constrain(Size{Width: width, Height: 1})
}

func (r *renderDivider) Paint(p *Painter, off Offset) {
	size := r.Size()
	cell := Cell{Character: r.Character, Style: r.Style}
	if r.Axis == Vertical {
		for y := 0; y < size.Height; y++ {
			p.DrawCell(Point{X: off.X, Y: off.Y + y}, cell)
		}
		return
	}
	for x := 0; x < size.Width; x++ {
		p.DrawCell(Point{X: off.X + x, Y: off.Y}, cell)
	}
}

func dividerCharacter(w Divider) Character {
	if w.Character != (Character{}) {
		return w.Character
	}
	if w.Axis == Vertical {
		return Character{Grapheme: "│", Width: 1}
	}
	return Character{Grapheme: "─", Width: 1}
}
