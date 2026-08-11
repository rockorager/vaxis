package vxfw

import (
	"fmt"
	"testing"

	"go.rockorager.dev/vaxis"
)

func TestRenderClipsNegativeChildOrigin(t *testing.T) {
	for _, ancestorRender := range []bool{false, true} {
		t.Run(fmt.Sprintf("ancestor render %t", ancestorRender), func(t *testing.T) {
			var renderWin vaxis.Window
			grandchild := NewSurface(8, 7, nil)
			grandchild.Render = func(win vaxis.Window) {
				renderWin = win
			}

			parent := NewSurface(2, 2, nil)
			if ancestorRender {
				parent.Render = func(vaxis.Window) {}
			}
			parent.AddChild(-3, -3, grandchild)
			root := NewSurface(10, 10, nil)
			root.AddChild(5, 4, parent)

			root.render(vaxis.Window{Width: 10, Height: 10}, nil)

			if col, row := renderWin.Origin(); col != 2 || row != 1 {
				t.Fatalf("render window origin = %d,%d, want 2,1", col, row)
			}
			if width, height := renderWin.Size(); width != 5 || height != 5 {
				t.Fatalf("render window size = %dx%d, want 5x5", width, height)
			}
			if renderWin.Parent == nil {
				t.Fatal("render window has no clipping parent")
			}
			if col, row := renderWin.Parent.Origin(); col != 5 || row != 4 {
				t.Fatalf("clip window origin = %d,%d, want 5,4", col, row)
			}
			if width, height := renderWin.Parent.Size(); width != 2 || height != 2 {
				t.Fatalf("clip window size = %dx%d, want 2x2", width, height)
			}
		})
	}
}

func TestRenderSkipsFullyClippedSubtree(t *testing.T) {
	called := false
	grandchild := NewSurface(5, 1, nil)
	grandchild.Render = func(vaxis.Window) {
		called = true
	}

	parent := NewSurface(5, 1, nil)
	parent.AddChild(-15, 0, grandchild)
	root := NewSurface(10, 1, nil)
	root.AddChild(20, 0, parent)

	root.render(vaxis.Window{Width: 10, Height: 1}, nil)

	if called {
		t.Fatal("render called for a fully clipped subtree")
	}
}

func TestRootRenderUsesSurfaceSize(t *testing.T) {
	var width, height int
	s := NewSurface(2, 1, nil)
	s.Render = func(win vaxis.Window) {
		width, height = win.Size()
	}

	s.render(vaxis.Window{Width: 10, Height: 5}, nil)

	if width != 2 || height != 1 {
		t.Fatalf("render window size = %dx%d, want 2x1", width, height)
	}
}
