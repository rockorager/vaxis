package ui

// Overlay paints entries above a stable child subtree.
//
// Overlay is useful for app-level surfaces such as dialogs, command palettes,
// menus, and other popups. It keeps the child as the first child of an
// always-present Stack so showing or hiding entries does not change the root
// shape of the application body.
type Overlay struct {
	// Child is the base subtree painted below all entries.
	Child Widget
	// Entries are painted in order above Child.
	Entries []OverlayEntry
}

func (w Overlay) Build(BuildContext) Widget {
	children := []Widget{w.Child}
	for _, entry := range w.Entries {
		if entry.Modal {
			barrier := entry.Barrier
			if barrier == nil {
				barrier = ModalBarrier{}
			}
			children = append(children, barrier)
		}
		child := entry.Child
		if child == nil {
			continue
		}
		if entry.Alignment != (Alignment{}) {
			child = Align{Alignment: entry.Alignment, Child: child}
		}
		children = append(children, child)
	}
	return Stack{Alignment: CenterAlign, Children: children}
}

// OverlayEntry describes one overlay surface.
type OverlayEntry struct {
	// Child is painted for this overlay entry.
	Child Widget
	// Modal inserts a barrier behind Child.
	Modal bool
	// Barrier overrides the default ModalBarrier when Modal is true.
	Barrier Widget
	// Alignment wraps Child in Align when non-zero.
	Alignment Alignment
}
