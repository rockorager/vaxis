package ui_test

import (
	"fmt"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/ui"
	"git.sr.ht/~rockorager/vaxis/ui/uitest"
)

func ExampleRun() {
	_ = ui.Run(ui.Center(ui.Text{Value: "hello"}))
}

func ExampleTextField() {
	value := ""
	field := ui.TextField{
		Value:       value,
		Placeholder: "Name",
		OnChanged: func(ctx ui.EventContext, next string) {
			value = next
		},
	}

	_ = field
}

type exampleSaveIntent struct{}

func (exampleSaveIntent) IntentType() ui.IntentType {
	return "example.save"
}

func ExampleShortcuts() {
	saved := false
	app := ui.NewApp(ui.Actions{
		Bindings: map[ui.IntentType]ui.ActionFunc{
			exampleSaveIntent{}.IntentType(): func(ctx ui.EventContext, intent ui.Intent) ui.EventResult {
				saved = true
				return ui.EventHandled
			},
		},
		Child: ui.Shortcuts{
			Bindings: ui.ShortcutMap{"s": exampleSaveIntent{}},
			Child:    ui.Button{Label: "save"},
		},
	})
	app.Pump(ui.Size{Width: 10, Height: 1})
	app.Send(vaxis.Key{Text: "s", Keycode: 's'})

	fmt.Println(saved)
	// Output: true
}

func ExampleWithShortcuts() {
	pressed := ""
	app := ui.NewApp(ui.Row(
		ui.Button{Label: "one", OnPressed: func(ctx ui.EventContext) { pressed = "one" }},
		ui.Button{Label: "two", OnPressed: func(ctx ui.EventContext) { pressed = "two" }},
	), ui.WithShortcuts(ui.ShortcutMap{
		"x": ui.NextFocusIntent{},
	}))
	app.Pump(ui.Size{Width: 20, Height: 1})
	app.Send(vaxis.Key{Text: "x", Keycode: 'x'})
	app.Send(vaxis.Key{Keycode: vaxis.KeyEnter})

	fmt.Println(pressed)
	// Output: two
}

func ExampleTextBuffer() {
	buffer := ui.NewTextBuffer("hello")
	buffer.SetCursorOffset(buffer.Len())
	buffer.Insert(", world")

	fmt.Println(buffer.Text())
	// Output: hello, world
}

func ExampleLayoutText() {
	layout := ui.LayoutText(
		[]ui.TextSpan{{Text: "hello world"}},
		ui.Constraints{MaxWidth: 5, MaxHeight: ui.Unbounded},
		ui.TextLayoutOptions{SoftWrap: true},
	)

	fmt.Println(layout.Size)
	// Output: {5 2}
}

func ExampleFloatTween() {
	tween := ui.FloatTween{Begin: 10, End: 20}

	fmt.Println(tween.At(ui.EaseInOut(0.5)))
	// Output: 15
}

type nameForm struct{}

func (nameForm) CreateState() ui.State {
	return &nameFormState{}
}

type nameFormState struct {
	ui.StateBase
	name string
}

func (s *nameFormState) Build(ui.BuildContext) ui.Widget {
	return ui.Column(
		ui.TextField{
			Value:       s.name,
			Placeholder: "Name",
			OnChanged: func(ctx ui.EventContext, next string) {
				s.SetState(func() { s.name = next })
			},
		},
		ui.Text{Value: "Hello, " + s.name},
	)
}

func ExampleStatefulWidget() {
	app := uitest.New(nameForm{})
	app.Pump(20, 2)
	app.Key("A")
	app.Pump(20, 2)

	fmt.Println(app.Contains("Hello, A"))
	// Output: true
}

type animatedLabel struct {
	Controller **ui.AnimationController
}

func (w animatedLabel) CreateState() ui.State {
	return &animatedLabelState{controller: w.Controller}
}

type animatedLabelState struct {
	ui.StateBase
	controller **ui.AnimationController
}

func (s *animatedLabelState) InitState() {
	controller := s.NewAnimation(ui.AnimationOptions{
		Duration: time.Second,
		Curve:    ui.EaseInOut,
	})
	controller.ForwardAt(time.Unix(0, 0))
	*s.controller = controller
}

func (s *animatedLabelState) Build(ui.BuildContext) ui.Widget {
	return ui.Text{Value: fmt.Sprintf("%.2f", (*s.controller).Value())}
}

func ExampleStateBase_NewAnimation() {
	var controller *ui.AnimationController
	app := ui.NewApp(animatedLabel{Controller: &controller})
	app.Pump(ui.Size{Width: 4, Height: 1})

	fmt.Println(controller.Running())
	// Output: true
}
