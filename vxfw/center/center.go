package center

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// Center draws the child centered within the space given to Center
type Center struct {
	Child vxfw.Widget
}

func (c *Center) HandleEvent(ev vaxis.Event, ph vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (c *Center) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if ctx.Max.HasUnboundedHeight() || ctx.Max.HasUnboundedWidth() {
		panic("Center must have bounded constraints")
	}
	chCtx := vxfw.DrawContext{
		Max:        ctx.Max,
		Characters: ctx.Characters,
	}
	chS, err := c.Child.Draw(chCtx)
	if err != nil {
		return vxfw.Surface{}, nil
	}
	// Create the surface for center
	s := vxfw.NewSurface(ctx.Max.Width, ctx.Max.Height, c)
	offX := (ctx.Max.Width - chS.Size.Width) / 2
	offY := (ctx.Max.Height - chS.Size.Height) / 2
	s.AddChild(int(offX), int(offY), chS)
	return s, err
}

// Verify we meet the Widget interface
var _ vxfw.Widget = &Center{}
