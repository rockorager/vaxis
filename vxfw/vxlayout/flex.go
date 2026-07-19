package vxlayout

import (
	"math"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

type FlexDirection int

const (
	FlexHorizontal FlexDirection = iota
	FlexVertical
)

type Alignment int

const (
	AlignStart Alignment = iota
	AlignCenter
	AlignEnd
)

// FlexItem is a [vxfw.Widget] used in a [FlexLayout]
type FlexItem struct {
	vxfw.Widget

	// Flex determines how much space the widget takes in the layout.
	// Flex of 0 means the widget will be its inherent size.
	// Remaining space is divided proportionally to all FlexItems with Flex > 0
	Flex uint8

	// Tight determines if the inherent size is ignored and is treated as purely
	// proportional.
	Tight bool

	// Justify controls how the widget is aligned within its allocated space.
	Justify Alignment
}

// unboundedContext takes a [vxfw.DrawContext] and returns a new one, with the main (flex) axis
// unbound.
// This context is used during the first layout pass to calculate inherent sizes of flex items.
func (f FlexDirection) unboundedContext(ctx vxfw.DrawContext) (out vxfw.DrawContext) {
	out = vxfw.DrawContext(ctx)
	switch f {
	case FlexHorizontal:
		out.Max.Width = math.MaxUint16
	case FlexVertical:
		out.Max.Height = math.MaxUint16
	}
	return
}

// flexContext takes a [vxfw.DrawContext] and a size and returns a new DrawContext that
// clamps the main axis to size
func (f FlexDirection) flexContext(ctx vxfw.DrawContext, size uint16) vxfw.DrawContext {
	out := vxfw.DrawContext(ctx)
	switch f {
	case FlexHorizontal:
		out.Min.Width = size
		out.Max.Width = size
	case FlexVertical:
		out.Min.Height = size
		out.Max.Height = size
	}
	return out
}

// flexAlignContext returns a context that caps the main axis to size without
// forcing the child to fill it. Used for per-item alignment.
//
// The main axis Min is reset to 0: ctx may carry a Min inherited from this FlexLayout's own
// incoming context (e.g. when the layout itself sits in an AlignStart flex slot of an outer
// layout, which sets Min=Max via flexContext). Left in place, that Min would leak into a child's
// slot and could force it to grow past size (widgets like text.Text honor Min over Max), pushing
// it out of its allocated slot and into its siblings.
func (f FlexDirection) flexAlignContext(ctx vxfw.DrawContext, size uint16) vxfw.DrawContext {
	out := vxfw.DrawContext(ctx)
	switch f {
	case FlexHorizontal:
		out.Min.Width = 0
		out.Max.Width = size
	case FlexVertical:
		out.Min.Height = 0
		out.Max.Height = size
	}
	return out
}

// mainAxis takes a size and returns the main axis depending on the direction
func (f FlexDirection) mainAxis(size vxfw.Size) (main uint16) {
	switch f {
	case FlexHorizontal:
		main = size.Width
	case FlexVertical:
		main = size.Height
	}
	return
}

// max takes a size and a constraint and returns the max of size or constraint depending on the
// direction
func (f FlexDirection) max(size vxfw.Size, constraint uint16) uint16 {
	if f == FlexHorizontal && size.Height > constraint {
		return size.Height
	} else if f == FlexVertical && size.Width > constraint {
		return size.Width
	}

	return constraint
}

// size takes main and cross axis sizes a [vxfw.Size] depending on the direction
func (f FlexDirection) size(main, cross uint16) (out vxfw.Size) {
	switch f {
	case FlexHorizontal:
		out.Width, out.Height = main, cross
	case FlexVertical:
		out.Height, out.Width = main, cross
	}
	return
}

// flexOrigin takes a [vxfw.RelativePoint] and returns a new one where the Row or Col is adjusted
// by offset based on the direction
func (f FlexDirection) flexOrigin(origin vxfw.RelativePoint, offset int) (p vxfw.RelativePoint) {
	p.Col, p.Row = origin.Col, origin.Row
	switch f {
	case FlexHorizontal:
		p.Col += offset
	case FlexVertical:
		p.Row += offset
	}
	return
}

type FlexLayout struct {
	Children  []*FlexItem
	Direction FlexDirection
}

var _ vxfw.Widget = (*FlexLayout)(nil)

func (w *FlexLayout) HandleEvent(_ vaxis.Event, _ vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (w *FlexLayout) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	// the accumulated size of all children in the first pass, this is their inherent size
	first_pass_size := uint16(0)

	// number of "flex units" opted in by the flex items
	total_flex := uint16(0)

	sizes := make([]uint16, len(w.Children))

	// First pass: layout
	// The layout pass draws each child assuming it had the full ctx to draw into to determine
	// what size it would be if it was not sharing the space.
	// We can use this information, plus the number of flex units, to determine how to distribute
	// the space.
	unboundedContext := w.Direction.unboundedContext(ctx)

	// This flex should absorb any leftover space if there is a non-flexed
	// trailing widget incoming, with the size specifying the distribution
	lastFlexIdx := -1
	var trailingFixedSize uint16

	// Iterate over each child, draw it, and measure the flex direction
	for i, child := range w.Children {
		total_flex += uint16(child.Flex)
		if child.Flex > 0 {
			lastFlexIdx = i
			trailingFixedSize = 0
			if child.Tight {
				continue
			}
		}

		surface, err := child.Draw(unboundedContext)
		if err != nil {
			return vxfw.Surface{}, err
		}

		inherent_size := w.Direction.mainAxis(surface.Size)
		first_pass_size += inherent_size
		sizes[i] = inherent_size
		if child.Flex == 0 {
			trailingFixedSize += inherent_size
		}
	}

	children := make([]vxfw.SubSurface, len(w.Children))

	// Accumulated offset for positioning (based on allocated slot sizes)
	accumulated := uint16(0)

	// The total size we can flex into along the main axis
	max_flex_axis := w.Direction.mainAxis(ctx.Max)

	// The largest size we've seen in the cross axis
	max_cross_axis := uint16(0)

	// Extra space left over that needs to be distributed
	remaining := max_flex_axis - first_pass_size + trailingFixedSize

	for i, child := range w.Children {
		inherent_size := sizes[i]
		child_size := uint16(0)

		if child.Flex == 0 {
			child_size = inherent_size
		} else if i == lastFlexIdx {
			child_size = max_flex_axis - accumulated - trailingFixedSize
		} else if child.Tight {
			child_size = (remaining * uint16(child.Flex)) / total_flex
		} else {
			child_size = inherent_size + (remaining*uint16(child.Flex))/total_flex
		}

		childctx := w.Direction.flexContext(ctx, child_size)
		if child.Justify != AlignStart {
			childctx = w.Direction.flexAlignContext(ctx, child_size)
		}

		surface, err := child.Draw(childctx)
		if err != nil {
			return vxfw.Surface{}, err
		}

		// Compute alignment offset within the allocated slot
		var slotOffset uint16
		if child.Justify != AlignStart {
			actual_main := w.Direction.mainAxis(surface.Size)
			if actual_main < child_size {
				slotOffset = child_size - actual_main
				if child.Justify == AlignCenter {
					slotOffset /= 2
				}
			}
		}

		// flex the origin by the accumulated size of the second pass
		origin := w.Direction.flexOrigin(vxfw.RelativePoint{}, int(accumulated+slotOffset))

		children[i] = vxfw.SubSurface{
			Origin:  origin,
			Surface: surface,
			ZIndex:  0,
		}

		// track the max of the cross axis
		max_cross_axis = w.Direction.max(surface.Size, max_cross_axis)
		accumulated += child_size
	}

	return vxfw.Surface{
		Size:     w.Direction.size(accumulated, max_cross_axis),
		Widget:   w,
		Buffer:   nil,
		Children: children,
	}, nil
}
