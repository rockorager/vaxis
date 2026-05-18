package list

import (
	"math"
	"slices"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// BuilderFunc is a function which takes
type BuilderFunc func(i uint, cursor uint) vxfw.Widget

// Dynamic is a dynamic list. It does not contain a slice of Widgets, but
// instead obtains them from a BuilderFunc
type Dynamic struct {
	// Builder is the function Dynamic calls to get a widget at a location
	Builder BuilderFunc
	// DrawCursor indicates if the view should draw it's own cursor
	DrawCursor bool
	// DisableEventHandlers prevents the widget from handling key or mouse
	// events. Set this to true to use custom event handlers
	DisableEventHandlers bool

	// Distance between each list item
	Gap int

	cursor uint
	scroll scroll
}

// scroll state
type scroll struct {
	// index of the top widget in the viewport
	top uint
	// line offset within the top widget. This is the number of lines
	// scrolled into the widget. A widget of height=4 with 1 row showing
	// would have this set to -3
	offset int
	// pending is the pending scroll amount
	pending int
	// wantsCursor is true if we need to ensure the cursor is in view
	wantsCursor bool
}

func (d *Dynamic) SetCursor(c uint) {
	d.cursor = c
	d.ensureScroll()
}

// SetPendingScroll sets a pending scroll amount, by lines. Positive numbers
// indicate a scroll down
func (d *Dynamic) SetPendingScroll(lines int) {
	d.scroll.pending = lines
}

func (d *Dynamic) CaptureEvent(ev vaxis.Event) (vxfw.Command, error) {
	if d.DisableEventHandlers {
		return nil, nil
	}

	// We capture key events
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.Matches('j') || ev.Matches(vaxis.KeyDown) {
			cmd := d.NextItem()
			if cmd == nil {
				return nil, nil
			}
			return vxfw.ConsumeAndRedraw(), nil
		}
		if ev.Matches('k') || ev.Matches(vaxis.KeyUp) {
			cmd := d.PrevItem()
			if cmd == nil {
				return nil, nil
			}
			return vxfw.ConsumeAndRedraw(), nil
		}
	}
	return nil, nil
}

// Cursor returns the index of the cursor
func (d *Dynamic) Cursor() uint {
	return d.cursor
}

// Offset returns the rendered offset of the list
func (d *Dynamic) Offset() int {
	return d.scroll.offset
}

func (d *Dynamic) HandleEvent(ev vaxis.Event, ph vxfw.EventPhase) (vxfw.Command, error) {
	if d.DisableEventHandlers {
		return nil, nil
	}
	switch ev := ev.(type) {
	case vaxis.Mouse:
		switch ev.Button {
		case vaxis.MouseWheelDown:
			d.scroll.pending += 3
			return vxfw.ConsumeAndRedraw(), nil
		case vaxis.MouseWheelUp:
			d.scroll.pending -= 3
			return vxfw.ConsumeAndRedraw(), nil
		}
	}
	return nil, nil
}

func (d *Dynamic) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if ctx.Max.HasUnboundedHeight() || ctx.Max.HasUnboundedWidth() {
		panic("Dynamic cannot have unbounded height or width")
	}

	s := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, d)

	// Accumulated height is the accumulated height we have drawn. We
	// initialize it to the scroll offset + any pending scroll we have. We
	// negate it so that it is lines *above* the viewport.
	//
	ah := -(d.scroll.offset + d.scroll.pending)

	// Now we can reset pending
	d.scroll.pending = 0

	// When ah > 0 and we are on the top widget, it means we have an upward
	// scroll which consumed all of the scroll offset. We can't go up any
	// more so we set some state here
	if ah > 0 && d.scroll.top == 0 {
		ah = 0
		d.scroll.offset = 0
	}

	// Set initial index for drawing downard. We set this before inserting
	// children because we might alter the top state based on accumulated
	// height
	i := d.scroll.top

	// When ah > 0, we will start drawing our previous "top" widget at a
	// positive offset. We need to insert widgets before it so we will do so
	// here
	if ah > 0 {
		err := d.insertChildren(ctx, &s, ah)
		if err != nil {
			return s, err
		}
		// Get the last child so we can set our accumulated height
		last := s.Children[len(s.Children)-1]
		ah = last.Origin.Row + int(last.Surface.Size.Height)
	}

	var colOffset int
	if d.DrawCursor {
		colOffset = 2
	}

	// Loop through widgets to draw
	for {
		// Get the widget at this index
		ch := d.Builder(i, d.cursor)
		// If we don't have one, we are done
		if ch == nil {
			break
		}
		// Increment the index
		i += 1

		chCtx := ctx.WithConstraints(vxfw.Size{}, vxfw.Size{
			Width:  ctx.Max.Width - uint16(colOffset),
			Height: math.MaxUint16,
		})

		// Draw the child
		chS, err := ch.Draw(chCtx)
		if err != nil {
			return s, err
		}
		// Add it to the parent. The accumulated height is the current
		// row we are drawing on
		s.AddChild(colOffset, ah, chS)

		// Add our height to accumulated height
		ah += int(chS.Size.Height) + d.Gap

		// If we need to draw the cursor, keep going
		if d.scroll.wantsCursor && i <= d.cursor {
			continue
		}
		// If we have accumulated enough height, we are done
		if ah >= int(ctx.Max.Height) {
			break
		}
	}

	var totalHeight uint16
	for _, ch := range s.Children {
		totalHeight += ch.Surface.Size.Height
	}
	if d.Gap > 0 && len(s.Children) > 1 {
		// Add gap for between each child
		totalHeight += uint16((len(s.Children) - 1) * d.Gap)
	}

	if d.DrawCursor {
		var row uint16
		// Set the entire gutter to a blank cell
		for ; row < s.Size.Height; row += 1 {
			s.WriteCell(0, row, vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: " ",
					Width:    1,
				},
			})
			s.WriteCell(1, row, vaxis.Cell{
				Character: vaxis.Character{
					Grapheme: " ",
					Width:    1,
				},
			})
		}

		// Get the index of the cursored widget in our child list
		idx := d.cursor - d.scroll.top

		// If our cursor is within the list, we draw a cursor next to it
		if int(idx) < len(s.Children) {
			ch := s.Children[idx]
			// Create a surface for the cursor
			cur := vxfw.NewSurface(ctx.Max.Width, ch.Surface.Size.Height, ch.Surface.Widget)

			// Draw the cursor glyph
			var curRow uint16
			for ; curRow < ch.Surface.Size.Height; curRow += 1 {
				cur.WriteCell(0, curRow, vaxis.Cell{
					Character: vaxis.Character{
						Grapheme: "▐",
						Width:    1,
					},
				})
			}
			// Add the cursored widget as a child of the cursor
			// surface
			cur.AddChild(colOffset, 0, ch.Surface)
			ss := vxfw.NewSubSurface(0, ch.Origin.Row, cur)
			// Reassign the SubSurface as this one
			s.Children[idx] = ss
		}
	}

	// If we want the cursor, we check that the cursored widget is in view.
	// We position it so that it is fully in view, and if it is too large
	// then the top portion of it is in view
	if d.scroll.wantsCursor {
		idx := d.cursor - d.scroll.top
		// Guaranteed we have drawn enough children from above
		if int(idx) < len(s.Children) {
			ch := s.Children[idx]

			// Define the bottom row
			bRow := ch.Origin.Row + int(ch.Surface.Size.Height)

			// The bottom row is beyond the height, adjust all the children
			// so that the bottom of the cursored widget is at the bottom of
			// the screen
			if bRow > int(ctx.Max.Height) {
				adj := int(ctx.Max.Height) - bRow
				for i, ch := range s.Children {
					ch.Origin.Row += adj
					s.Children[i] = ch
				}
			}
			d.scroll.wantsCursor = false

		}
	}

	// Reset origins and state based on actual draw
	for i, ch := range s.Children {
		if ch.Origin.Row <= 0 &&
			ch.Origin.Row+int(ch.Surface.Size.Height) > 0 {
			d.scroll.top += uint(i)
			d.scroll.offset = -ch.Origin.Row
		}
	}

	return s, nil
}

// Inserts children until h < 0
func (d *Dynamic) insertChildren(ctx vxfw.DrawContext, p *vxfw.Surface, ah int) error {
	// We'll start at the widget before the top widget
	d.scroll.top -= 1

	var colOffset int
	if d.DrawCursor {
		colOffset = 2
	}

	for ah > 0 {
		chCtx := ctx.WithMax(vxfw.Size{
			Width:  ctx.Max.Width - uint16(colOffset),
			Height: math.MaxUint16,
		})
		ch := d.Builder(d.scroll.top, d.cursor)
		// Break if we don't have a widget, really this should never
		// happen
		if ch == nil {
			break
		}

		s, err := ch.Draw(chCtx)
		if err != nil {
			return err
		}
		// Subtract the height of this surface and add it to the parent
		ah -= int(s.Size.Height)
		ss := vxfw.NewSubSurface(colOffset, ah, s)
		p.Children = slices.Insert(p.Children, 0, ss)

		if d.scroll.top == 0 {
			break
		}

		// Decrease the top widget index
		d.scroll.top -= 1
	}

	// Our ah is now the offset into the top widget
	d.scroll.offset = ah

	// We reached the top widget but are below row 0. Reset the
	if d.scroll.top == 0 && ah > 0 {
		d.scroll.offset = 0
		var row uint16
		for i, ch := range p.Children {
			ch.Origin.Row = int(row)
			p.Children[i] = ch
			row += ch.Surface.Size.Height
		}
		return nil
	}

	return nil
}

func (d *Dynamic) NextItem() vxfw.Command {
	// Check if we have another item
	w := d.Builder(d.cursor+1, d.cursor)
	if w == nil {
		return nil
	}
	d.cursor += 1
	d.ensureScroll()
	return vxfw.RedrawCmd{}
}

func (d *Dynamic) PrevItem() vxfw.Command {
	if d.cursor == 0 {
		return nil
	}
	w := d.Builder(d.cursor-1, d.cursor)
	if w == nil {
		return nil
	}
	d.cursor -= 1
	d.ensureScroll()
	return vxfw.RedrawCmd{}
}

func (d *Dynamic) ensureScroll() {
	if d.cursor > d.scroll.top {
		d.scroll.wantsCursor = true
		return
	}
	d.scroll.top = d.cursor
	d.scroll.offset = 0
}

var _ vxfw.Widget = &Dynamic{}
