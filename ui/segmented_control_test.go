package ui

import "testing"

func TestSegmentedControlPaintsSegments(t *testing.T) {
	app := NewApp(SegmentedControl[string]{
		Value: "cozy",
		Segments: []SegmentedItem[string]{
			{Value: "compact", Label: "Compact"},
			{Value: "cozy", Label: "Cozy"},
			{Value: "wide", Label: "Wide"},
		},
	})
	app.Pump(Size{Width: 40, Height: 1})

	p := NewPainter(Size{Width: 40, Height: 1})
	app.Paint(p)
	if got := debugRenderedText(p); got != " Compact │ Cozy │ Wide" {
		t.Fatalf("rendered control = %q, want segments", got)
	}
}

func TestSegmentedControlKeyboardSelectsEnabledSegments(t *testing.T) {
	var selected string
	app := NewApp(SegmentedControl[string]{
		Value: "compact",
		Segments: []SegmentedItem[string]{
			{Value: "compact", Label: "Compact"},
			{Value: "disabled", Label: "Disabled", Disabled: true},
			{Value: "wide", Label: "Wide"},
		},
		OnChanged: func(ctx EventContext, value string) {
			selected = value
		},
	})
	app.Pump(Size{Width: 40, Height: 1})

	app.focusNext()
	app.Send(Key{Keycode: KeyRight})
	app.Send(Key{Keycode: '\r'})
	if selected != "wide" {
		t.Fatalf("selected = %q, want wide", selected)
	}
}

func TestSegmentedControlMouseSelectsSegment(t *testing.T) {
	var selected string
	app := NewApp(SegmentedControl[string]{
		Value: "compact",
		Segments: []SegmentedItem[string]{
			{Value: "compact", Label: "Compact"},
			{Value: "cozy", Label: "Cozy"},
			{Value: "wide", Label: "Wide"},
		},
		OnChanged: func(ctx EventContext, value string) {
			selected = value
		},
	})
	app.Pump(Size{Width: 40, Height: 1})

	app.Send(Mouse{Col: 11, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if selected != "cozy" {
		t.Fatalf("selected = %q, want cozy", selected)
	}
}

func TestSegmentedControlStylesStates(t *testing.T) {
	theme := DefaultTheme()
	theme.SegmentedControl.Selected = Style{Background: RGB(1, 1, 1)}
	theme.SegmentedControl.Focused = Style{UnderlineStyle: UnderlineSingle}
	theme.SegmentedControl.Disabled = Style{Attribute: AttrDim}
	app := NewApp(SegmentedControl[string]{
		Value: "cozy",
		Segments: []SegmentedItem[string]{
			{Value: "compact", Label: "Compact"},
			{Value: "cozy", Label: "Cozy"},
			{Value: "disabled", Label: "Disabled", Disabled: true},
		},
		OnChanged: func(EventContext, string) {
		},
	}, WithTheme(theme))
	app.Pump(Size{Width: 40, Height: 1})
	app.focusNext()
	app.Pump(Size{Width: 40, Height: 1})

	p := NewPainter(Size{Width: 40, Height: 1})
	app.Paint(p)
	if got := p.Cell(11, 0).Background; got != theme.SegmentedControl.Selected.Background {
		t.Fatalf("selected background = %#v, want %#v", got, theme.SegmentedControl.Selected.Background)
	}
	if got := p.Cell(11, 0).UnderlineStyle; got != UnderlineSingle {
		t.Fatalf("focused underline = %#v, want single", got)
	}
	if got := p.Cell(18, 0).Attribute; got&AttrDim == 0 {
		t.Fatalf("disabled attribute = %#v, want dim", got)
	}
}

func TestSegmentedControlDisabledDoesNotFocusOrActivate(t *testing.T) {
	activated := false
	app := NewApp(Column(
		SegmentedControl[string]{
			Value:    "compact",
			Disabled: true,
			Segments: []SegmentedItem[string]{
				{Value: "compact", Label: "Compact"},
				{Value: "cozy", Label: "Cozy"},
			},
			OnChanged: func(EventContext, string) {
				activated = true
			},
		},
		Button{Label: "Next", OnPressed: func(EventContext) {
			activated = true
		}},
	))
	app.Pump(Size{Width: 40, Height: 2})

	app.focusNext()
	app.Send(Mouse{Col: 1, Row: 0, Button: MouseLeftButton, EventType: EventPress})
	if activated {
		t.Fatal("disabled segmented control activated")
	}

	app.Send(Key{Keycode: '\r'})
	if !activated {
		t.Fatal("focus did not skip disabled segmented control to activate next button")
	}
}
