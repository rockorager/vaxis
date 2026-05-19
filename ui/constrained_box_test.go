package ui

import "testing"

func TestConstraintsEnforceCombinesConstraints(t *testing.T) {
	got := (Constraints{MinWidth: 2, MaxWidth: 20, MinHeight: 1, MaxHeight: 10}).Enforce(Constraints{MinWidth: 5, MaxWidth: 12, MinHeight: 3, MaxHeight: 8})
	want := Constraints{MinWidth: 5, MaxWidth: 12, MinHeight: 3, MaxHeight: 8}
	if got != want {
		t.Fatalf("enforced constraints = %#v, want %#v", got, want)
	}
}

func TestConstrainedBoxAppliesMinimumConstraints(t *testing.T) {
	child := &recordingRenderObject{desired: Size{Width: 1, Height: 1}}
	app := NewApp(Align{Alignment: TopLeft, Child: ConstrainedBox{
		Constraints: Constraints{MinWidth: 4, MinHeight: 3},
		Child:       recordingWidget{RO: child},
	}})
	app.Pump(Size{Width: 10, Height: 5})
	if child.Size() != (Size{Width: 4, Height: 3}) {
		t.Fatalf("child size = %#v, want 4x3", child.Size())
	}
	ro := findRenderObject(app.build.Root()).(*renderAlign).Child().(*renderConstrainedBox)
	if ro.Size() != (Size{Width: 4, Height: 3}) {
		t.Fatalf("box size = %#v, want 4x3", ro.Size())
	}
}

func TestConstrainedBoxAppliesMaximumConstraints(t *testing.T) {
	child := &recordingRenderObject{desired: Size{Width: 10, Height: 5}}
	app := NewApp(Align{Alignment: TopLeft, Child: ConstrainedBox{
		Constraints: Constraints{MaxWidth: 4, MaxHeight: 2},
		Child:       recordingWidget{RO: child},
	}})
	app.Pump(Size{Width: 10, Height: 5})
	if child.Size() != (Size{Width: 4, Height: 2}) {
		t.Fatalf("child size = %#v, want 4x2", child.Size())
	}
}

func TestConstrainedBoxRespectsParentConstraints(t *testing.T) {
	child := &recordingRenderObject{desired: Size{Width: 10, Height: 1}}
	app := NewApp(Align{Alignment: TopLeft, Child: ConstrainedBox{
		Constraints: Constraints{MinWidth: 8, MaxWidth: 20},
		Child:       recordingWidget{RO: child},
	}})
	app.Pump(Size{Width: 5, Height: 1})
	if child.Size().Width != 5 {
		t.Fatalf("child width = %d, want parent max 5", child.Size().Width)
	}
}

func TestConstrainedBoxMarksLayoutDirtyWhenConstraintsChange(t *testing.T) {
	app := NewApp(ConstrainedBox{Constraints: Constraints{MinWidth: 2}, Child: Text{Value: "x"}})
	app.Pump(Size{Width: 10, Height: 1})
	ro := findRenderObject(app.build.Root()).(*renderConstrainedBox)
	app.UpdateRoot(ConstrainedBox{Constraints: Constraints{MinWidth: 4}, Child: Text{Value: "x"}})
	if !ro.NeedsLayout() {
		t.Fatal("constraint update should mark layout dirty")
	}
}
