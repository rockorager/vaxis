package ui

import "git.sr.ht/~rockorager/vaxis"

type scrollAxisController interface {
	ScrollByLinesAxis(ScrollAxis, int) bool
	ScrollByPagesAxis(ScrollAxis, int) bool
	ScrollToStartAxis(ScrollAxis) bool
	ScrollToEndAxis(ScrollAxis) bool
}

func scrollDefaultActions(controller scrollAxisController, child Widget) Widget {
	return DefaultActions{
		Bindings: map[IntentType]ActionFunc{
			ScrollIntentType: func(ctx EventContext, intent Intent) EventResult {
				scroll, ok := intent.(ScrollIntent)
				if !ok {
					return EventIgnored
				}
				return invokeScrollIntent(controller, scroll)
			},
		},
		Child: child,
	}
}

func invokeScrollIntent(controller scrollAxisController, intent ScrollIntent) EventResult {
	if controller == nil {
		return EventIgnored
	}
	sign := 1
	if intent.Direction == ScrollBackward {
		sign = -1
	}
	switch intent.Unit {
	case ScrollUnitLine:
		controller.ScrollByLinesAxis(intent.Axis, sign)
	case ScrollUnitPage:
		controller.ScrollByPagesAxis(intent.Axis, sign)
	case ScrollUnitEdge:
		if intent.Direction == ScrollBackward {
			controller.ScrollToStartAxis(intent.Axis)
		} else {
			controller.ScrollToEndAxis(intent.Axis)
		}
	default:
		return EventIgnored
	}
	return EventHandled
}

func handleScrollKey(key Key, controller scrollOffsetController) EventResult {
	if keyIsRelease(key) || controller == nil {
		return EventIgnored
	}
	intent, ok := scrollKeyIntent(key, ScrollVertical)
	if !ok {
		return EventIgnored
	}
	return invokeScrollOffsetIntent(controller, intent)
}

func handleHorizontalScrollKey(key Key, controller scrollOffsetController) EventResult {
	if keyIsRelease(key) || controller == nil {
		return EventIgnored
	}
	intent, ok := scrollKeyIntent(key, ScrollHorizontal)
	if !ok {
		return EventIgnored
	}
	return invokeScrollOffsetIntent(controller, intent)
}

func invokeScrollOffsetIntent(controller scrollOffsetController, intent ScrollIntent) EventResult {
	if controller == nil {
		return EventIgnored
	}
	sign := 1
	if intent.Direction == ScrollBackward {
		sign = -1
	}
	switch intent.Unit {
	case ScrollUnitLine:
		controller.ScrollByLines(sign)
	case ScrollUnitPage:
		controller.ScrollByPages(sign)
	case ScrollUnitEdge:
		if intent.Direction == ScrollBackward {
			controller.ScrollToStart()
		} else {
			controller.ScrollToEnd()
		}
	default:
		return EventIgnored
	}
	return EventHandled
}

func handleScrollKeyForAxis(ctx EventContext, key Key, axis ScrollAxis) EventResult {
	if keyIsRelease(key) {
		return EventIgnored
	}
	intent, ok := scrollKeyIntent(key, axis)
	if !ok {
		return EventIgnored
	}
	return ctx.Invoke(intent)
}

func handleScrollKeyWithInvoke(ctx EventContext, key Key) EventResult {
	return handleScrollKeyForAxis(ctx, key, ScrollVertical)
}

func handleScrollPaneKey(key Key, controller scrollAxisController) EventResult {
	if keyIsRelease(key) || controller == nil {
		return EventIgnored
	}
	intent, ok := scrollPaneKeyIntent(key)
	if !ok {
		return EventIgnored
	}
	return invokeScrollIntent(controller, intent)
}

func handleScrollPaneKeyWithInvoke(ctx EventContext, key Key) EventResult {
	if keyIsRelease(key) {
		return EventIgnored
	}
	intent, ok := scrollPaneKeyIntent(key)
	if !ok {
		return EventIgnored
	}
	return ctx.Invoke(intent)
}

func scrollKeyIntent(key Key, axis ScrollAxis) (ScrollIntent, bool) {
	backwardKey := key.Keycode == KeyUp || key.MatchString("k")
	forwardKey := key.Keycode == KeyDown || key.MatchString("j")
	if axis == ScrollHorizontal {
		backwardKey = key.Keycode == KeyLeft || key.MatchString("h")
		forwardKey = key.Keycode == KeyRight || key.MatchString("l")
	}
	switch {
	case backwardKey:
		return ScrollIntent{Axis: axis, Direction: ScrollBackward, Unit: ScrollUnitLine}, true
	case forwardKey:
		return ScrollIntent{Axis: axis, Direction: ScrollForward, Unit: ScrollUnitLine}, true
	case key.Keycode == KeyPgUp || keyIsShiftSpace(key):
		return ScrollIntent{Axis: axis, Direction: ScrollBackward, Unit: ScrollUnitPage}, true
	case key.Keycode == KeyPgDown || keyIsSpace(key):
		return ScrollIntent{Axis: axis, Direction: ScrollForward, Unit: ScrollUnitPage}, true
	case key.Keycode == KeyHome:
		return ScrollIntent{Axis: axis, Direction: ScrollBackward, Unit: ScrollUnitEdge}, true
	case key.Keycode == KeyEnd:
		return ScrollIntent{Axis: axis, Direction: ScrollForward, Unit: ScrollUnitEdge}, true
	default:
		return ScrollIntent{}, false
	}
}

func scrollPaneKeyIntent(key Key) (ScrollIntent, bool) {
	switch {
	case key.Keycode == KeyUp || key.MatchString("k"):
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollBackward, Unit: ScrollUnitLine}, true
	case key.Keycode == KeyDown || key.MatchString("j"):
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollForward, Unit: ScrollUnitLine}, true
	case key.Keycode == KeyLeft || key.MatchString("h"):
		return ScrollIntent{Axis: ScrollHorizontal, Direction: ScrollBackward, Unit: ScrollUnitLine}, true
	case key.Keycode == KeyRight || key.MatchString("l"):
		return ScrollIntent{Axis: ScrollHorizontal, Direction: ScrollForward, Unit: ScrollUnitLine}, true
	case key.Keycode == KeyPgUp || keyIsShiftSpace(key):
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollBackward, Unit: ScrollUnitPage}, true
	case key.Keycode == KeyPgDown || keyIsSpace(key):
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollForward, Unit: ScrollUnitPage}, true
	case key.Keycode == KeyHome:
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollBackward, Unit: ScrollUnitEdge}, true
	case key.Keycode == KeyEnd:
		return ScrollIntent{Axis: ScrollVertical, Direction: ScrollForward, Unit: ScrollUnitEdge}, true
	default:
		return ScrollIntent{}, false
	}
}

func keyIsSpace(key Key) bool {
	return key.Keycode == vaxis.KeySpace && key.Modifiers&vaxis.ModShift == 0
}

func keyIsShiftSpace(key Key) bool {
	return key.Keycode == vaxis.KeySpace && key.Modifiers&vaxis.ModShift != 0
}
