package vaxis

import (
	"git.sr.ht/~rockorager/vaxis/ansi"
)

type Mouse struct {
	Button    MouseButton
	Row       int
	Col       int
	EventType EventType
	Modifiers ModifierMask
}

type MouseButton int

const (
	MouseLeftButton MouseButton = iota
	MouseMiddleButton
	MouseRightButton
	MouseNoButton

	MouseWheelUp   MouseButton = 64
	MouseWheelDown MouseButton = 65

	MouseButton8  MouseButton = 128
	MouseButton9  MouseButton = 129
	MouseButton10 MouseButton = 130
	MouseButton11 MouseButton = 131
)

const (
	motion        = 0b00100000
	buttonBits    = 0b11000011
	mouseModShift = 0b00000100
	mouseModAlt   = 0b00001000
	mouseModCtrl  = 0b00010000
)

func parseMouseEvent(seq ansi.CSI) (Mouse, bool) {
	mouse := Mouse{}
	if len(seq.Intermediate) != 1 && seq.Intermediate[0] != '<' {
		Logger.Error("[CSI] unknown sequence", "sequence", seq)
		return mouse, false
	}

	if len(seq.Parameters) != 3 {
		Logger.Error("[CSI] unknown sequence", "sequence", seq)
		return mouse, false
	}

	switch seq.Final {
	case 'M':
		mouse.EventType = EventPress
	case 'm':
		mouse.EventType = EventRelease
	}

	// buttons are encoded with the high two and low two bits
	button := seq.Parameters[0][0] & buttonBits
	mouse.Button = MouseButton(button)

	if seq.Parameters[0][0]&motion != 0 {
		mouse.EventType = EventMotion
	}

	if seq.Parameters[0][0]&mouseModShift != 0 {
		mouse.Modifiers |= ModShift
	}
	if seq.Parameters[0][0]&mouseModAlt != 0 {
		mouse.Modifiers |= ModAlt
	}
	if seq.Parameters[0][0]&mouseModCtrl != 0 {
		mouse.Modifiers |= ModCtrl
	}

	mouse.Col = seq.Parameters[1][0] - 1
	mouse.Row = seq.Parameters[2][0] - 1

	return mouse, true
}
