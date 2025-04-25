package vaxis

import (
	"git.sr.ht/~rockorager/vaxis/ansi"
	"git.sr.ht/~rockorager/vaxis/log"
)

// Mouse is a mouse event
type Mouse struct {
	Button    MouseButton
	Row       int
	Col       int
	EventType EventType
	Modifiers ModifierMask
	XPixel    int
	YPixel    int
}

// MouseButton represents a mouse button
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

// MouseShape is used with OSC 22 to change the shape of the mouse cursor
type MouseShape string

const (
	MouseShapeDefault          MouseShape = "default"
	MouseShapeTextInput        MouseShape = "text"
	MouseShapeClickable        MouseShape = "pointer"
	MouseShapeHelp             MouseShape = "help"
	MouseShapeBusyBackground   MouseShape = "progress"
	MouseShapeBusy             MouseShape = "wait"
	MouseShapeResizeHorizontal MouseShape = "ew-resize"
	MouseShapeResizeVertical   MouseShape = "ns-resize"
	// The thick plus sign cursor that's typically used in spread-sheet applications to select cells.
	MouseShapeCell MouseShape = "cell"
)

const (
	motion        = 0b00100000
	buttonBits    = 0b11000011
	mouseModShift = 0b00000100
	mouseModAlt   = 0b00001000
	mouseModCtrl  = 0b00010000
)

func pixelToCell(px, length, cells int) int {
	if length > 0 {
		return px * cells / length
	}
	return 0
}

func parseMouseEvent(seq ansi.CSI, ws Resize, enableSGRPixels bool) (Mouse, bool) {
	mouse := Mouse{}
	if len(seq.Intermediate) != 1 && seq.Intermediate[0] != '<' {
		log.Error("[CSI] unknown sequence: %s", seq)
		return mouse, false
	}

	if len(seq.Parameters) != 3 {
		log.Error("[CSI] unknown sequence: %s", seq)
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

	if enableSGRPixels {
		mouse.XPixel = seq.Parameters[1][0]
		mouse.YPixel = seq.Parameters[2][0]
		mouse.Col = pixelToCell(mouse.XPixel, ws.XPixel, ws.Cols)
		mouse.Row = pixelToCell(mouse.YPixel, ws.YPixel, ws.Rows)
	} else {
		mouse.Col = seq.Parameters[1][0] - 1
		mouse.Row = seq.Parameters[2][0] - 1
	}

	return mouse, true
}
