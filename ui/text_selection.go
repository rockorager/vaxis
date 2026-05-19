package ui

// TextSelection describes a directional text range from Base to Extent.
type TextSelection struct {
	Base   TextPosition
	Extent TextPosition
}

// NewTextSelection creates a selection from base to extent.
func NewTextSelection(base, extent TextPosition) TextSelection {
	return TextSelection{Base: base, Extent: extent}
}

// IsCollapsed reports whether the selection is empty.
func (s TextSelection) IsCollapsed() bool {
	return sameTextPosition(s.Base, s.Extent)
}

// Normalized returns the selection ordered from earlier to later position.
func (s TextSelection) Normalized() TextSelection {
	if compareTextPosition(s.Base, s.Extent) <= 0 {
		return s
	}
	return TextSelection{Base: s.Extent, Extent: s.Base}
}

// Contains reports whether pos is inside the selection.
func (s TextSelection) Contains(pos TextPosition) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, pos) <= 0 && compareTextPosition(pos, s.Extent) < 0
}

// IntersectsCell reports whether the selection overlaps cell.
func (s TextSelection) IntersectsCell(cell TextCell) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, cell.End()) < 0 && compareTextPosition(cell.Position, s.Extent) < 0
}

// ContainsLineBreak reports whether the selection covers line's trailing line break.
func (s TextSelection) ContainsLineBreak(line TextLine) bool {
	if s.IsCollapsed() {
		return false
	}
	s = s.Normalized()
	return compareTextPosition(s.Base, line.End) <= 0 && compareTextPosition(line.End, s.Extent) < 0
}
