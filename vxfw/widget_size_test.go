package vxfw_test // _test package to avoid import cycles

import (
	"strings"
	"testing"

	"go.rockorager.dev/vaxis"
	"go.rockorager.dev/vaxis/vxfw"
	"go.rockorager.dev/vaxis/vxfw/button"
	"go.rockorager.dev/vaxis/vxfw/center"
	"go.rockorager.dev/vaxis/vxfw/list"
	"go.rockorager.dev/vaxis/vxfw/richtext"
	"go.rockorager.dev/vaxis/vxfw/text"
)

func TestWidgetConstraints(t *testing.T) {
	lt := func(t *testing.T, lhs, rhs uint16, dim string) {
		t.Helper()

		if lhs <= rhs {
			return
		}

		t.Fail()
		t.Logf("%s check failed, %d is not less than %d", dim, lhs, rhs)
	}

	ctx := vxfw.DrawContext{
		Min:        vxfw.Size{Width: 4, Height: 4},
		Max:        vxfw.Size{Width: 16, Height: 16},
		Characters: vaxis.Characters,
	}

	short := "_"
	long := strings.Repeat(short, 256)

	testcases := []struct {
		name   string
		widget vxfw.Widget
	}{{
		"text",
		text.New(short),
	}, {
		"text-long",
		text.New(long),
	}, {
		"richtext",
		richtext.New([]vaxis.Segment{{Text: short}}),
	}, {
		"richtext-long",
		richtext.New([]vaxis.Segment{{Text: long}}),
	}, {
		"center",
		&center.Center{Child: text.New(short)},
	}, {
		"center-long",
		&center.Center{Child: text.New(long)},
	}, {
		"button",
		&button.Button{Label: short},
	}, {
		"button-long",
		&button.Button{Label: long},
	}, {
		"list",
		&list.Dynamic{
			Builder: func(i, _ uint) vxfw.Widget {
				if i == 1 {
					return text.New(short)
				}
				return nil
			},
		},
	}, {
		"list-long",
		&list.Dynamic{
			Builder: func(i, _ uint) vxfw.Widget {
				return text.New(long)
			},
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			surface, err := tc.widget.Draw(ctx)
			if err != nil {
				t.Fatalf("unexpected error calling Draw: %v", err)
			}

			sz := surface.Size
			min := ctx.Min
			max := ctx.Max

			lt(t, min.Width, sz.Width, "min-width")
			lt(t, min.Height, sz.Height, "min-height")

			lt(t, sz.Width, max.Width, "max-width")
			lt(t, sz.Height, max.Height, "max-height")
		})
	}
}
