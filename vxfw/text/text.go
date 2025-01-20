package text

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

type Text struct {
	// The content of the Text widget
	Content string

	// The style to draw the text as
	Style vaxis.Style

	// Whether to softwrap the text or not
	Softwrap bool
}

func New(content string) *Text {
	return &Text{
		Content:  content,
		Softwrap: true,
	}
}

// Noop for text
func (t *Text) HandleEvent(ev vaxis.Event, phase vxfw.EventPhase) (vxfw.Command, error) {
	return nil, nil
}

func (t *Text) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	chars := ctx.Characters(t.Content)
	cells := make([]vaxis.Cell, 0, len(chars))
	var w int
	for _, char := range chars {
		cell := vaxis.Cell{
			Character: char,
			Style:     t.Style,
		}
		cells = append(cells, cell)
		w += char.Width
	}

	return vxfw.Surface{
		Size:     vxfw.Size{Width: uint16(w), Height: 1},
		Widget:   t,
		Cursor:   nil,
		Buffer:   cells,
		Children: []vxfw.SubSurface{},
	}, nil
}
