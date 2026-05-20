package ui

import "git.sr.ht/~rockorager/vaxis"

func handleScrollKey(key Key, controller scrollOffsetController) EventResult {
	if keyIsRelease(key) || controller == nil {
		return EventIgnored
	}
	switch {
	case key.Keycode == KeyUp || key.MatchString("k"):
		controller.ScrollByLines(-1)
		return EventHandled
	case key.Keycode == KeyDown || key.MatchString("j"):
		controller.ScrollByLines(1)
		return EventHandled
	case key.Keycode == KeyPgUp || keyIsShiftSpace(key):
		controller.ScrollByPages(-1)
		return EventHandled
	case key.Keycode == KeyPgDown || keyIsSpace(key):
		controller.ScrollByPages(1)
		return EventHandled
	case key.Keycode == KeyHome:
		controller.ScrollToStart()
		return EventHandled
	case key.Keycode == KeyEnd:
		controller.ScrollToEnd()
		return EventHandled
	default:
		return EventIgnored
	}
}

func handleHorizontalScrollKey(key Key, controller scrollOffsetController) EventResult {
	if keyIsRelease(key) || controller == nil {
		return EventIgnored
	}
	switch {
	case key.Keycode == KeyLeft || key.MatchString("h"):
		controller.ScrollByLines(-1)
		return EventHandled
	case key.Keycode == KeyRight || key.MatchString("l"):
		controller.ScrollByLines(1)
		return EventHandled
	case key.Keycode == KeyPgUp || keyIsShiftSpace(key):
		controller.ScrollByPages(-1)
		return EventHandled
	case key.Keycode == KeyPgDown || keyIsSpace(key):
		controller.ScrollByPages(1)
		return EventHandled
	case key.Keycode == KeyHome:
		controller.ScrollToStart()
		return EventHandled
	case key.Keycode == KeyEnd:
		controller.ScrollToEnd()
		return EventHandled
	default:
		return EventIgnored
	}
}

func keyIsSpace(key Key) bool {
	return key.Keycode == vaxis.KeySpace && key.Modifiers&vaxis.ModShift == 0
}

func keyIsShiftSpace(key Key) bool {
	return key.Keycode == vaxis.KeySpace && key.Modifiers&vaxis.ModShift != 0
}
