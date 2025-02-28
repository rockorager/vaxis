package button

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"git.sr.ht/~rockorager/vaxis/vxfw/center"
	"git.sr.ht/~rockorager/vaxis/vxfw/text"
)

type Button struct {
	Label   string
	Style   StyleSet
	OnClick func() (vxfw.Command, error)

	mouseDown bool
	hover     bool
	focused   bool
}

type StyleSet struct {
	Default   vaxis.Style
	MouseDown vaxis.Style
	Hover     vaxis.Style
	Focus     vaxis.Style
}

func New(label string, onClick func() (vxfw.Command, error)) *Button {
	ss := StyleSet{
		Default: vaxis.Style{
			Attribute: vaxis.AttrReverse,
		},
		MouseDown: vaxis.Style{
			Foreground: vaxis.IndexColor(4),
			Attribute:  vaxis.AttrReverse,
		},
		Hover: vaxis.Style{
			Foreground: vaxis.IndexColor(3),
			Attribute:  vaxis.AttrReverse,
		},
		Focus: vaxis.Style{
			Foreground: vaxis.IndexColor(5),
			Attribute:  vaxis.AttrReverse,
		},
	}
	return &Button{
		Label:   label,
		Style:   ss,
		OnClick: onClick,
	}
}

func (b *Button) HandleEvent(ev vaxis.Event, ph vxfw.EventPhase) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.EventType == vaxis.EventRelease {
			return nil, nil
		}
		if ev.Matches(vaxis.KeyEnter) {
			return b.OnClick()
		}
	case vaxis.Mouse:
		b.hover = true
		if b.mouseDown && ev.EventType == vaxis.EventRelease {
			b.mouseDown = false
			return b.OnClick()
		}
		if ev.EventType == vaxis.EventPress && ev.Button == vaxis.MouseLeftButton {
			b.mouseDown = true
			return vxfw.ConsumeAndRedraw(), nil
		}
	case vxfw.MouseEnter:
		b.hover = true
		cmd := []vxfw.Command{
			vxfw.SetMouseShapeCmd(vaxis.MouseShapeClickable),
			vxfw.RedrawCmd{},
			vxfw.ConsumeEventCmd{},
		}
		return cmd, nil
	case vxfw.MouseLeave:
		b.hover = false
		b.mouseDown = false
		cmd := []vxfw.Command{
			vxfw.SetMouseShapeCmd(vaxis.MouseShapeDefault),
			vxfw.RedrawCmd{},
			vxfw.ConsumeEventCmd{},
		}
		return cmd, nil
	case vaxis.FocusIn:
		b.focused = true
		return vxfw.ConsumeAndRedraw(), nil
	case vaxis.FocusOut:
		b.focused = false
		b.mouseDown = false
		return vxfw.ConsumeAndRedraw(), nil
	}
	return nil, nil
}

func (b *Button) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if ctx.Max.HasUnboundedHeight() || ctx.Max.HasUnboundedWidth() {
		panic("Button must have bounded constraints")
	}
	var style vaxis.Style
	switch {
	case b.mouseDown:
		style = b.Style.MouseDown
	case b.hover:
		style = b.Style.Hover
	case b.focused:
		style = b.Style.Focus
	default:
		style = b.Style.Default
	}

	l := text.New(b.Label)
	l.Style = style

	center := center.Center{Child: l}
	s, err := center.Draw(ctx)
	if err != nil {
		return vxfw.Surface{}, err
	}
	// Rewrite the widget of this surface. We don't really care about the
	// Center widget anyways, it's just for layout
	s.Widget = b
	s.Fill(style)
	return s, nil
}

// Verify we meet the Widget interface
var _ vxfw.Widget = &Button{}
