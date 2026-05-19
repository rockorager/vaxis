package ui

import "testing"

func TestStackPaintsChildrenInOrder(t *testing.T) {
	app := NewApp(Stack{Children: []Widget{
		Text{Value: "a"},
		Text{Value: "b"},
	}})
	app.Pump(Size{Width: 1, Height: 1})

	p := NewPainter(Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Grapheme; got != "b" {
		t.Fatalf("painted cell = %q, want top child b", got)
	}
}

func TestStackPositionsChildren(t *testing.T) {
	app := NewApp(Stack{Alignment: TopLeft, Children: []Widget{
		SizedBox{Width: 6, Height: 3},
		Positioned{Left: 2, Top: 1, Child: Text{Value: "x"}},
	}})
	app.Pump(Size{Width: 6, Height: 3})

	p := NewPainter(Size{Width: 6, Height: 3})
	app.Paint(p)
	if got := p.Cell(2, 1).Grapheme; got != "x" {
		t.Fatalf("positioned cell = %q, want x", got)
	}
}

func TestStackAlignsNonPositionedChildren(t *testing.T) {
	app := NewApp(Stack{Alignment: BottomRight, Children: []Widget{
		SizedBox{Width: 5, Height: 3},
		Text{Value: "x"},
	}})
	app.Pump(Size{Width: 5, Height: 3})

	p := NewPainter(Size{Width: 5, Height: 3})
	app.Paint(p)
	if got := p.Cell(4, 2).Grapheme; got != "x" {
		t.Fatalf("aligned cell = %q, want x", got)
	}
}

func TestStackDryLayoutMatchesLayout(t *testing.T) {
	ro := (&Stack{Children: []Widget{
		SizedBox{Width: 5, Height: 2},
		Positioned{Left: 8, Top: 4, Child: SizedBox{Width: 1, Height: 1}},
	}}).CreateRenderObject(BuildContext{})
	ro.(*renderStack).SetChildren([]RenderObject{
		(&SizedBox{Width: 5, Height: 2}).CreateRenderObject(BuildContext{}),
		(&SizedBox{Width: 1, Height: 1}).CreateRenderObject(BuildContext{}),
	})
	ro.(*renderStack).Children()[1].Base().SetParentData(StackParentData{Positioned: true, Left: 8, Top: 4})

	constraints := Constraints{MaxWidth: 20, MaxHeight: 10}
	dry := DryLayout(LayoutContext{}, ro, constraints)
	ro.Layout(LayoutContext{}, constraints)
	if got := ro.Base().Size(); got != dry {
		t.Fatalf("layout size = %#v, want dry %#v", got, dry)
	}
}

func TestStackHitTestsTopChildFirst(t *testing.T) {
	var pressed string
	app := NewApp(Stack{Children: []Widget{
		Button{Label: "bottom", OnPressed: func(EventContext) { pressed = "bottom" }},
		Button{Label: "top", OnPressed: func(EventContext) { pressed = "top" }},
	}})
	app.Pump(Size{Width: 10, Height: 1})

	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if pressed != "top" {
		t.Fatalf("pressed = %q, want top", pressed)
	}
}
