package vxlayout

import (
	"fmt"
	"math"
	"os"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// Flexible describes a [vxfw.Widget] that can take a flexible amount of space in a [Row] or
// [Column].
type Flexible interface {
	vxfw.Widget
	FlexFactor() uint16
	FlexLoose() bool
}

// Determines how children in a [Row] or [Column] are laid out on the cross axis.
// The default is [CrossAxisCenter], which centers children in the available cross axis space.
// Use [CrossAxisStart] to align children to the top of a [Row] or left of a [Column], and vice
// versa for [CrossAxisEnd].
// [CrossAxisStretch] will force all children to fill the maximum space on the cross axis.
type CrossAxisAlignment int

const (
	CrossAxisCenter CrossAxisAlignment = iota
	CrossAxisStart
	CrossAxisEnd
	CrossAxisStretch
)

// Determines how children in a [Row] or [Column] are laid out on the main axis.
// The default is [MainAxisStart], which places children at the left or top of a [Row] or [Column]
// respectively.
// Use [MainAxisCenter] to center children, or one of the Space variants to distribute the space
// elsewhere.
// Note that these options do nothing if one of the children has a flex factor > 0, as those
// children will take the available space.
type MainAxisAlignment int

const (
	MainAxisStart MainAxisAlignment = iota
	MainAxisEnd
	MainAxisCenter
	MainAxisSpaceBetween
	MainAxisSpaceAround
	MainAxisSpaceEvenly
)

type LayoutOptions struct {
	MainAxis  MainAxisAlignment
	CrossAxis CrossAxisAlignment

	// Gap controls how much space is placed between each child *before* the children are sized.
	Gap uint16
}

// Row returns a [vxfw.Widget] that lays out children horizontally.
func Row(children []vxfw.Widget, options LayoutOptions) vxfw.Widget {
	return &flex{children: children, options: options, direction: FlexHorizontal}
}

// Column returns a [vxfw.Widget] that lays out children vertically.
func Column(children []vxfw.Widget, options LayoutOptions) vxfw.Widget {
	return &flex{children: children, options: options, direction: FlexVertical}
}

type box struct {
	vxfw.Widget
	flex  uint16
	loose bool
}

var (
	_ vxfw.Widget = box{}
	_ Flexible    = box{}
)

func (b box) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	return b.Widget.Draw(ctx)
}

func (b box) FlexLoose() bool    { return b.loose }
func (b box) FlexFactor() uint16 { return b.flex }

// Expanded returns a [vxfw.Widget] that will expand in a flexible layout based on flex.
func Expanded(widget vxfw.Widget, flex uint16) vxfw.Widget {
	return box{Widget: widget, flex: flex}
}

// Flex returns a [vxfw.Widget] that will loosely flex in a flexible layout based on flex.
func Flex(widget vxfw.Widget, flex uint16) vxfw.Widget {
	return box{Widget: widget, flex: flex, loose: true}
}

type widgetFunc func(vxfw.DrawContext) (vxfw.Surface, error)

func (w widgetFunc) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	return w(ctx)
}

// Space returns a [vxfw.Widget] that will fill all available space in a flexible layout.
// Note that a Space(0) has no flex factor and will take ALL space that comes after it in the
// layout. If flex is > 0 (ie, Space(1)), Space will share remaining space proportionally with
// other [Flexible] widgets.
func Space(flex uint16) vxfw.Widget {
	return box{Widget: widgetFunc(func(ctx vxfw.DrawContext) (vxfw.Surface, error) {
		return vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, nil), nil
	}), flex: flex}
}

// Fill returns a [vxfw.Widget] that fills its space with the supplied cell.
// Note that Fill will take all available space. It's primarily useful to diagnose layouts, and
// will usually be contained in a [Flex]
func Fill(cell vaxis.Cell) vxfw.Widget {
	return widgetFunc(func(ctx vxfw.DrawContext) (vxfw.Surface, error) {
		surface := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, nil)
		surface.Fill(cell)
		return surface, nil
	})
}

// Constrained is a [vxfw.Widget] that constrains a widget by min and max size.
// Min and max are pointers to [vxfw.Size] so that the caller can indicate a lack of constraint.
// If either axis of size is 0, that axis is ignored for constraint purposes.
// Constrained can be useful for laying out a Row where you want to ensure a maximum height.
func Constrained(widget vxfw.Widget, minSize, maxSize *vxfw.Size) vxfw.Widget {
	return widgetFunc(func(ctx vxfw.DrawContext) (vxfw.Surface, error) {
		if minSize != nil {
			if minSize.Width > ctx.Min.Width {
				ctx.Min.Width = minSize.Width
			}
			if minSize.Height > ctx.Min.Height {
				ctx.Min.Height = minSize.Height
			}
		}
		if maxSize != nil {
			if maxSize.Width != 0 && maxSize.Width < ctx.Max.Width {
				ctx.Max.Width = maxSize.Width
			}
			if maxSize.Height != 0 && maxSize.Height < ctx.Max.Height {
				ctx.Max.Height = maxSize.Height
			}
		}

		return widget.Draw(ctx)
	})
}

// Sized is a [vxfw.Widget] that passes a fixed size to its child widget as long as size fits the
// incoming constraints. If either axis of size is 0, that size is ignored.
// This can be used to place a widget that normally cannot be unconstrained (such as an infinite
// list) into a flexible layout. See also [Limited].
// This is a shortcut for Constrained(widget, size, size)
func Sized(widget vxfw.Widget, size vxfw.Size) vxfw.Widget {
	return Constrained(widget, &size, &size)
}

// Limited is a [vxfw.Widget] that limits its child by size only if the incoming constraint is
// unlimited. If either axis of size is 0, that axis is ignored.
func Limited(widget vxfw.Widget, size vxfw.Size) vxfw.Widget {
	return widgetFunc(func(ctx vxfw.DrawContext) (vxfw.Surface, error) {
		if ctx.Max.HasUnboundedWidth() && size.Width > 0 {
			ctx.Max.Width = size.Width
		}
		if ctx.Max.HasUnboundedHeight() && size.Height > 0 {
			ctx.Max.Height = size.Height
		}

		return widget.Draw(ctx)
	})
}

type flex struct {
	children []vxfw.Widget
	options  LayoutOptions

	// TODO: rename FlexDirection to just Direction, as its useable in other layout types such as
	// Wrap
	direction FlexDirection
}

var _ vxfw.Widget = flex{}

func (f flex) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	fmt.Fprintf(
		os.Stderr,
		"flex.render: cons.w=%d, cons.h=%d, gap=%d, starting_used=%d, direction=%d, children=%d\n",
		ctx.Max.Width, ctx.Max.Height,
		f.options.Gap,
		f.options.Gap*uint16(len(f.children)-1),
		f.direction,
		len(f.children),
	)

	surfaces := make([]vxfw.Surface, len(f.children))
	var used_space uint16
	var flex_units uint16
	var max_cross_axis uint16

	// First, claim space for our gap
	used_space = f.options.Gap * uint16(len(f.children)-1)

	// Next, lay out non-flexible children and determine how much space we're using and how many
	// flex units to distribute the remaining space
	for i, child := range f.children {
		if c, ok := child.(Flexible); ok {
			// If the flex factor is 0, this is the same as being intrinsically sized
			factor := c.FlexFactor()
			if factor > 0 {
				flex_units += factor
				continue
			}
		}

		surface, err := child.Draw(instrinsicConstraint(ctx, f.direction, f.options.CrossAxis))
		if err != nil {
			return vxfw.Surface{}, err
		}

		surfaces[i] = surface
		used_space += f.direction.mainAxis(surface.Size)
		max_cross_axis = f.direction.maxCrossAxis(surface.Size, max_cross_axis)
	}

	// Now we can distribute the remaining space to flexible children based on their flex factor
	remaining := f.direction.mainAxis(ctx.Max) - used_space

	for i, child := range f.children {
		c, ok := child.(Flexible)
		// Non-flexible children, or children with a flex factor of 0, were laid out in the
		// first pass above.
		if !ok || c.FlexFactor() == 0 {
			continue
		}

		size := uint16(0)

		if i == len(f.children)-1 {
			// last child gets all of the remaining space
			size = remaining
		} else {
			// otherwise, size is based on the flex factor
			size = (remaining * c.FlexFactor()) / flex_units
		}

		// If c is FlexLoose, we loosen the minimum constraint to 0
		// Otherwise (the default), the child must take a tight constraint
		cons := flexibleConstraint(ctx, f.direction, f.options.CrossAxis, size)
		if c.FlexLoose() {
			cons = loosenConstraint(cons, f.direction)
		} else {
			cons = tightenConstraint(cons, f.direction)
		}

		fmt.Fprintf(
			os.Stderr,
			"  flex.render(flexible): child=%d, loose=%t, flex=%d, size=%d\n",
			i, c.FlexLoose(), c.FlexFactor(), size,
		)

		surface, err := c.Draw(cons)
		if err != nil {
			return vxfw.Surface{}, err
		}

		surfaces[i] = surface
		// remaining -= f.direction.mainAxis(surface.Size)
		used_space += f.direction.mainAxis(surface.Size)
		max_cross_axis = f.direction.maxCrossAxis(surface.Size, max_cross_axis)
	}

	// We have all of our surfaces, we know our constraints, it's time to finalize the layout.
	// Each child is placed within the parent surface based on the layout options.
	// TODO: Implement MainAxisSize option which allows the main axis to take min (size of all
	// children) or max. For now, we'll always use the max.
	// Note that even if we implement min main axis size, it becomes irrelevant if any of the
	// children are flexible (and not loose)
	size := f.direction.size(f.direction.mainAxis(ctx.Max), max_cross_axis)
	surface := vxfw.Surface{
		Size:     size,
		Children: make([]vxfw.SubSurface, len(surfaces)),
	}

	fmt.Fprintf(os.Stderr, "flex.render: surface.size=%d\n", f.direction.mainAxis(size))

	/*
		Distribution.

		First, apply f.options.Gap *between* each child (ie, increase offset)
		If main axis start, no other adjustments
		if main axis end, offset starts at remaining
		If main axis center, offset starts at remaining / 2
		If main axis space between, gap increases by remaining / (len(children)-1)
		If main axis space around, gap increases by remaining / len(children), offset starts at gap / 2
		If main axis space evenly, gap increases by remaining / len(children)+1, offset starts at gap

		Cross offset is simpler:

		if cross axis start, offset is 0
		if cross axis end, offset is max cross - child cross
		if cross axis center, offset is (max cross - child cross) / 2
		if cross axis stretch, offset is 0 (child is already tight in the cross axis)

		TODO: Should children sized 0 on the main axis (such as a Space with 0 flex factor) use the
		gap? Currently, they do. But it's also probably a bug to have a Space(0) in a flex layout.
	*/

	// reset remaining space to remove what was used
	// TODO: unify f.direction.mainAxis(ctx.Max) with above calls, this represents the main axis
	// extent of the layout and only changes if (and when) MainAxisSize option is implemented.
	remaining = f.direction.mainAxis(ctx.Max) - used_space
	var offset, gap uint16
	var nchildren uint16 = uint16(len(f.children))

	gap = f.options.Gap

	switch f.options.MainAxis {
	case MainAxisEnd:
		offset = remaining
	case MainAxisCenter:
		offset = remaining / 2
	case MainAxisSpaceBetween:
		// Place all remaining space between the children
		gap += remaining / (nchildren - 1)
	case MainAxisSpaceAround:
		// Place all remaining space between children, with half that space on each end
		chunk := remaining / nchildren
		gap += chunk
		offset = chunk / 2
	case MainAxisSpaceEvenly:
		// Place all remaining space between, before, and after children equally
		chunk := remaining / (nchildren + 1)
		gap += chunk
		offset = chunk
	}

	fmt.Fprintf(os.Stderr, "flex.render: offset=%d, gap=%d->%d, remaining=%d\n", offset, f.options.Gap, gap, remaining)

	// Iterate over children, applying spacing and distribution options
	var cross uint16
	for i, child := range surfaces {
		// If this is not the first child, add gap to offset
		if i > 0 {
			offset += gap
		}

		cross = 0
		switch f.options.CrossAxis {
		case CrossAxisEnd:
			cross = max_cross_axis - f.direction.crossAxis(child.Size)
		case CrossAxisCenter:
			cross = (max_cross_axis - f.direction.crossAxis(child.Size)) / 2
		}

		origin := flexOrigin(f.direction, offset, cross)
		surface.Children[i] = vxfw.SubSurface{
			Origin:  origin,
			Surface: child,
		}
		old_offset := offset
		offset += f.direction.mainAxis(child.Size)

		fmt.Fprintf(
			os.Stderr,
			" flex.render: child=%d, size=%d, gap=%d, offset=%d->%d\n",
			i,
			f.direction.mainAxis(child.Size),
			gap,
			old_offset,
			offset,
		)

	}

	return surface, nil
}

// instrinsicConstraint takes a [vxfw.DrawContext] and returns a new one with the main axis
// unbound, and the cross axis adjusted based on the crossalign.
// This constraint is used to compute instrinsic sizes of non-[Flexible] children in the first
// layout pass.
func instrinsicConstraint(ctx vxfw.DrawContext, direction FlexDirection, crossalign CrossAxisAlignment) (out vxfw.DrawContext) {
	out = vxfw.DrawContext(ctx)
	switch direction {
	case FlexHorizontal:
		out.Max.Width = math.MaxUint16
		if crossalign == CrossAxisStretch {
			out.Min.Height = out.Max.Height
		}
	case FlexVertical:
		out.Max.Height = math.MaxUint16
		if crossalign == CrossAxisStretch {
			out.Min.Width = out.Max.Width
		}
	}
	return
}

// flexibleConstraint is like [instrinsicConstraint] but sets the main axis to the specified size.
// This is used to layout a Flexible child after determining its portion of the available space.
func flexibleConstraint(ctx vxfw.DrawContext, direction FlexDirection, crossalign CrossAxisAlignment, size uint16) (out vxfw.DrawContext) {
	out = vxfw.DrawContext(ctx)
	switch direction {
	case FlexHorizontal:
		out.Max.Width = size
		if crossalign == CrossAxisStretch {
			out.Min.Height = out.Max.Height
		}
	case FlexVertical:
		out.Max.Height = size
		if crossalign == CrossAxisStretch {
			out.Min.Width = out.Max.Width
		}
	}
	return
}

// loosenConstraint takes a ctx and loosens the main axis
func loosenConstraint(ctx vxfw.DrawContext, direction FlexDirection) (out vxfw.DrawContext) {
	out = vxfw.DrawContext(ctx)
	switch direction {
	case FlexHorizontal:
		out.Min.Width = 0
	case FlexVertical:
		out.Min.Height = 0
	}
	return
}

// tightenConstraint is the opposite of loosenConstraint, it forces a tight constraint on the main
// axis
func tightenConstraint(ctx vxfw.DrawContext, direction FlexDirection) (out vxfw.DrawContext) {
	out = vxfw.DrawContext(ctx)
	switch direction {
	case FlexHorizontal:
		out.Min.Width = out.Max.Width
	case FlexVertical:
		out.Min.Height = out.Max.Height
	}
	return
}

// flexOrigin takes main and cross and returns a [vxfw.RelativePoint]
func flexOrigin(direction FlexDirection, main, cross uint16) (p vxfw.RelativePoint) {
	switch direction {
	case FlexHorizontal:
		p.Col = int(main)
		p.Row = int(cross)
	case FlexVertical:
		p.Row = int(main)
		p.Col = int(cross)
	}
	return
}
