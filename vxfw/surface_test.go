package vxfw_test

import (
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
)

func TestWriteCellBoundsCheck(t *testing.T) {
	// Create a surface where Size claims more cells than Buffer actually has.
	// This simulates the bug where Size and Buffer get out of sync.
	s := vxfw.Surface{
		Size: vxfw.Size{
			Width:  100,
			Height: 100,
		},
		Buffer: make([]vaxis.Cell, 50), // Only 50 cells, but Size claims 10000
	}

	cell := vaxis.Cell{
		Character: vaxis.Character{
			Grapheme: "x",
			Width:    1,
		},
	}

	// This should not panic even though the index would be out of bounds
	// if we only checked against Size.
	s.WriteCell(99, 99, cell) // Would calculate index 9999, but Buffer only has 50 cells

	// Also test that valid writes still work
	s.WriteCell(0, 0, cell)
	if s.Buffer[0].Character.Grapheme != "x" {
		t.Errorf("expected cell at (0,0) to be written, got %q", s.Buffer[0].Character.Grapheme)
	}
}
