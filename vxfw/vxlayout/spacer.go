package vxlayout

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

// Spacer is a [vxfw.Widget] that takes up available space. It should be used in
// a [FlexItem] with a flex of at least 1.
// Use [NewSpacer] or [MustSpacer] for a [FlexItem] that can be passed directly to [FlexLayout]
type Spacer struct{}

// NewSpacer returns a FlexItem that fills available flex space based on the value of flex.
// It is an error to pass a flex less than 1.
func NewSpacer(flex uint8) (*FlexItem, error) {
	if flex < 1 {
		return nil, fmt.Errorf("spacer flex must be at least 1, got: %d", flex)
	}

	return &FlexItem{
		Widget: Spacer{},
		Flex:   flex,
	}, nil
}

// MustSpacer is like NewSpacer, but panics if flex is less than 1.
func MustSpacer(flex uint8) *FlexItem {
	w, err := NewSpacer(flex)
	if err != nil {
		panic(err)
	}
	return w
}

func (s Spacer) HandleEvent(_ vaxis.Event, _ vxfw.EventPhase) (vxfw.Command, error) { return nil, nil }
func (s Spacer) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	return vxfw.NewSurface(ctx.Min.Width, ctx.Min.Height, s), nil
}
