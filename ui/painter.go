package ui

// Painter records terminal cells and cursor state for a frame.
type Painter struct {
	size   Size
	cells  []Cell
	clips  []Rect
	cursor *CursorState
}

// CursorState describes the terminal cursor requested during paint.
type CursorState struct {
	Col   int
	Row   int
	Shape CursorStyle
}

// NewPainter creates a painter with a blank cell buffer of size.
func NewPainter(size Size) *Painter {
	return &Painter{size: size, cells: make([]Cell, size.Width*size.Height), clips: []Rect{{Width: size.Width, Height: size.Height}}}
}

// Size returns the painter's cell buffer size.
func (p *Painter) Size() Size {
	return p.size
}

// Cells returns the painter's backing cell buffer.
func (p *Painter) Cells() []Cell {
	return p.cells
}

func (p *Painter) clone() *Painter {
	if p == nil {
		return nil
	}
	clone := &Painter{
		size:  p.size,
		cells: append([]Cell(nil), p.cells...),
		clips: append([]Rect(nil), p.clips...),
	}
	if p.cursor != nil {
		cursor := *p.cursor
		clone.cursor = &cursor
	}
	return clone
}

// Cell returns the cell at x,y or an empty cell when out of bounds.
func (p *Painter) Cell(x, y int) Cell {
	if x < 0 || y < 0 || x >= p.size.Width || y >= p.size.Height {
		return Cell{}
	}
	return p.cells[y*p.size.Width+x]
}

// ShowCursor records a visible cursor if the position is in bounds and unclipped.
func (p *Painter) ShowCursor(col, row int, shape CursorStyle) {
	pt := Point{X: col, Y: row}
	if !p.inClip(pt) || !p.inBounds(pt) {
		p.cursor = nil
		return
	}
	p.cursor = &CursorState{Col: col, Row: row, Shape: shape}
}

// HideCursor clears any requested cursor.
func (p *Painter) HideCursor() {
	p.cursor = nil
}

// Cursor returns the requested cursor state.
func (p *Painter) Cursor() (CursorState, bool) {
	if p.cursor == nil {
		return CursorState{}, false
	}
	return *p.cursor, true
}

// DrawCell writes a single cell when pt is in bounds and unclipped.
func (p *Painter) DrawCell(pt Point, cell Cell) {
	if !p.inClip(pt) || pt.X < 0 || pt.Y < 0 || pt.X >= p.size.Width || pt.Y >= p.size.Height {
		return
	}
	p.cells[pt.Y*p.size.Width+pt.X] = cell
}

// DrawText writes s at off using style.
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

// Fill writes cell into every visible cell of r.
func (p *Painter) Fill(r Rect, cell Cell) {
	for y := r.Y; y < r.Y+r.Height; y++ {
		for x := r.X; x < r.X+r.Width; x++ {
			p.DrawCell(Point{x, y}, cell)
		}
	}
}

// Scrim blends every visible cell in r toward color by opacity.
//
// Opacity ranges from 0, no effect, to 255, replace RGB colors with color.
// Non-RGB colors are left unchanged because their terminal palette values are
// not known to the painter.
func (p *Painter) Scrim(r Rect, color Color, opacity uint8) {
	if opacity == 0 {
		return
	}
	for y := r.Y; y < r.Y+r.Height; y++ {
		for x := r.X; x < r.X+r.Width; x++ {
			pt := Point{x, y}
			if !p.inClip(pt) || !p.inBounds(pt) {
				continue
			}
			cell := p.Cell(x, y)
			cell.Foreground = scrimColor(cell.Foreground, color, opacity)
			cell.Background = scrimColor(cell.Background, color, opacity)
			cell.UnderlineColor = scrimColor(cell.UnderlineColor, color, opacity)
			p.cells[y*p.size.Width+x] = cell
		}
	}
}

// PushClip restricts subsequent drawing to r.
func (p *Painter) PushClip(r Rect) {
	p.clips = append(p.clips, intersectRect(p.clips[len(p.clips)-1], r))
}

// PopClip restores the previous clip rectangle.
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

func intersectRect(a, b Rect) Rect {
	x0 := max(a.X, b.X)
	y0 := max(a.Y, b.Y)
	x1 := min(a.X+a.Width, b.X+b.Width)
	y1 := min(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: x0, Y: y0, Width: max(0, x1-x0), Height: max(0, y1-y0)}
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

func scrimColor(src, dst Color, opacity uint8) Color {
	if src == 0 || dst == 0 || opacity == 0 {
		return src
	}
	s := src.Params()
	d := dst.Params()
	if len(s) != 3 || len(d) != 3 {
		return src
	}
	return RGB(
		blendChannel(s[0], d[0], opacity),
		blendChannel(s[1], d[1], opacity),
		blendChannel(s[2], d[2], opacity),
	)
}

func blendChannel(src, dst, opacity uint8) uint8 {
	return uint8((int(src)*(255-int(opacity)) + int(dst)*int(opacity)) / 255)
}
