package ui

type Painter struct {
	size  Size
	cells []Cell
	clips []Rect
}

func NewPainter(size Size) *Painter {
	return &Painter{size: size, cells: make([]Cell, size.Width*size.Height), clips: []Rect{{Width: size.Width, Height: size.Height}}}
}

func (p *Painter) Size() Size {
	return p.size
}

func (p *Painter) Cells() []Cell {
	return p.cells
}

func (p *Painter) Cell(x, y int) Cell {
	if x < 0 || y < 0 || x >= p.size.Width || y >= p.size.Height {
		return Cell{}
	}
	return p.cells[y*p.size.Width+x]
}

func (p *Painter) DrawCell(pt Point, cell Cell) {
	if !p.inClip(pt) || pt.X < 0 || pt.Y < 0 || pt.X >= p.size.Width || pt.Y >= p.size.Height {
		return
	}
	p.cells[pt.Y*p.size.Width+pt.X] = cell
}

func (p *Painter) DrawText(off Offset, s string, style Style) {
	x := off.X
	for _, ch := range vaxisCharacters(s) {
		pt := Point{X: x, Y: off.Y}
		cell := Cell{Character: ch, Style: style}
		if p.inBounds(pt) {
			cell.Style = mergeStyle(p.Cell(pt.X, pt.Y).Style, style)
		}
		p.DrawCell(pt, cell)
		x += ch.Width
	}
}

func (p *Painter) Fill(r Rect, cell Cell) {
	for y := r.Y; y < r.Y+r.Height; y++ {
		for x := r.X; x < r.X+r.Width; x++ {
			p.DrawCell(Point{x, y}, cell)
		}
	}
}

func (p *Painter) PushClip(r Rect) {
	p.clips = append(p.clips, r)
}

func (p *Painter) PopClip() {
	if len(p.clips) > 1 {
		p.clips = p.clips[:len(p.clips)-1]
	}
}

func (p *Painter) inClip(pt Point) bool {
	c := p.clips[len(p.clips)-1]
	return pt.X >= c.X && pt.X < c.X+c.Width && pt.Y >= c.Y && pt.Y < c.Y+c.Height
}

func (p *Painter) inBounds(pt Point) bool {
	return pt.X >= 0 && pt.Y >= 0 && pt.X < p.size.Width && pt.Y < p.size.Height
}

func mergeStyle(base, over Style) Style {
	if over.Hyperlink != "" {
		base.Hyperlink = over.Hyperlink
	}
	if over.HyperlinkParams != "" {
		base.HyperlinkParams = over.HyperlinkParams
	}
	if over.Foreground != 0 {
		base.Foreground = over.Foreground
	}
	if over.Background != 0 {
		base.Background = over.Background
	}
	if over.UnderlineColor != 0 {
		base.UnderlineColor = over.UnderlineColor
	}
	if over.UnderlineStyle != 0 {
		base.UnderlineStyle = over.UnderlineStyle
	}
	if over.Attribute != 0 {
		base.Attribute = over.Attribute
	}
	return base
}
