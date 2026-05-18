package ui

type TextSelection struct {
	Base   TextPosition
	Extent TextPosition
}

func NewTextSelection(base, extent TextPosition) TextSelection {
	return TextSelection{Base: base, Extent: extent}
}

func (s TextSelection) IsCollapsed() bool {
	return sameTextPosition(s.Base, s.Extent)
}

func (s TextSelection) Normalized() TextSelection {
	if compareTextPosition(s.Base, s.Extent) <= 0 {
		return s
	}
	return TextSelection{Base: s.Extent, Extent: s.Base}
}

func (s TextSelection) Contains(pos TextPosition) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, pos) <= 0 && compareTextPosition(pos, s.Extent) < 0
}

func (s TextSelection) IntersectsCell(cell TextCell) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, cell.End()) < 0 && compareTextPosition(cell.Position, s.Extent) < 0
}

func (s TextSelection) ContainsLineBreak(line TextLine) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, line.End) <= 0 && compareTextPosition(line.End, s.Extent) < 0
}
