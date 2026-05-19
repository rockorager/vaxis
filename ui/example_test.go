package ui_test

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis/ui"
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
