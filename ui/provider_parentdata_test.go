package ui

import "testing"

type buildMutatingWidget struct{ State **buildMutatingState }

func (w buildMutatingWidget) CreateState() State {
	s := &buildMutatingState{}
	*w.State = s
	return s
}

type buildMutatingState struct {
	StateBase
	mutateDuringBuild bool
}

func (s *buildMutatingState) Build(BuildContext) Widget {
	if s.mutateDuringBuild {
		s.MarkNeedsBuild()
	}
	return Text{Value: "x"}
}

func TestMarkNeedsBuildDuringBuildPanics(t *testing.T) {
	var state *buildMutatingState
	app := NewApp(buildMutatingWidget{State: &state})
	app.Pump(Size{Width: 1, Height: 1})
	state.mutateDuringBuild = true
	state.MarkNeedsBuild()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MarkNeedsBuild during build did not panic")
		}
	}()
	app.Pump(Size{Width: 1, Height: 1})
}

type providedValueConsumer struct{ Seen *[]string }

func (w providedValueConsumer) CreateState() State {
	return &providedValueState{seen: w.Seen}
}

type providedValueState struct {
	StateBase
	seen *[]string
}

func (s *providedValueState) Build(ctx BuildContext) Widget {
	*s.seen = append(*s.seen, MustDepend[string](ctx))
	return Text{Value: MustDepend[string](ctx)}
}

func TestProviderUsesNearestProvider(t *testing.T) {
	var seen []string
	app := NewApp(Provider[string]{Value: "outer", ChildWidget: Provider[string]{Value: "inner", ChildWidget: providedValueConsumer{Seen: &seen}}})
	app.Pump(Size{Width: 10, Height: 1})
	if len(seen) != 1 || seen[0] != "inner" {
		t.Fatalf("seen = %#v, want inner", seen)
	}
}

func TestProviderSuppressedNotificationStillUpdatesValueForLaterBuild(t *testing.T) {
	var seen []string
	app := NewApp(Provider[string]{Value: "one", ShouldNotify: func(string, string) bool { return false }, ChildWidget: providedValueConsumer{Seen: &seen}})
	app.Pump(Size{Width: 10, Height: 1})
	state := findState[*providedValueState](app.build.Root())
	app.UpdateRoot(Provider[string]{Value: "two", ShouldNotify: func(string, string) bool { return false }, ChildWidget: providedValueConsumer{Seen: &seen}})
	app.Pump(Size{Width: 10, Height: 1})
	if len(seen) != 1 {
		t.Fatalf("suppressed notification rebuilt consumer; seen = %#v", seen)
	}
	state.MarkNeedsBuild()
	app.Pump(Size{Width: 10, Height: 1})
	if got := seen[len(seen)-1]; got != "two" {
		t.Fatalf("rebuilt consumer saw %q, want two", got)
	}
}

func TestExpandedParentDataChangeMarksParentLayoutDirty(t *testing.T) {
	app := NewApp(Row(ExpandedWidget{Flex: 1, ChildWidget: Text{Value: "x"}}))
	app.Pump(Size{Width: 10, Height: 1})
	row := findRenderObject(app.build.Root()).(*RenderFlex)
	if row.NeedsLayout() {
		t.Fatal("row should be clean after pump")
	}
	app.UpdateRoot(Row(ExpandedWidget{Flex: 2, ChildWidget: Text{Value: "x"}}))
	if !row.NeedsLayout() {
		t.Fatal("changing flex parent data should mark parent layout dirty")
	}
}
