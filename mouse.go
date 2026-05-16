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

	MouseWheelUp    MouseButton = 64
	MouseWheelDown  MouseButton = 65
	MouseWheelLeft  MouseButton = 66
	MouseWheelRight MouseButton = 67

	MouseButton8  MouseButton = 128
	MouseButton9  MouseButton = 129
	MouseButton10 MouseButton = 130
	MouseButton11 MouseButton = 131
)

// MouseShape is used with OSC 22 to change the shape of the mouse cursor
type MouseShape string

const (
	MouseShapeDefault          MouseShape = "default"
	MouseShapeContextMenu      MouseShape = "context-menu"
	MouseShapeTextInput        MouseShape = "text"
	MouseShapeVerticalText     MouseShape = "vertical-text"
	MouseShapeClickable        MouseShape = "pointer"
	MouseShapeHelp             MouseShape = "help"
	MouseShapeBusyBackground   MouseShape = "progress"
	MouseShapeBusy             MouseShape = "wait"
	MouseShapeAlias            MouseShape = "alias"
	MouseShapeCopy             MouseShape = "copy"
	MouseShapeMove             MouseShape = "move"
	MouseShapeNoDrop           MouseShape = "no-drop"
	MouseShapeNotAllowed       MouseShape = "not-allowed"
	MouseShapeGrab             MouseShape = "grab"
	MouseShapeGrabbing         MouseShape = "grabbing"
	MouseShapeAllScroll        MouseShape = "all-scroll"
	MouseShapeCrosshair        MouseShape = "crosshair"
	MouseShapeResizeColumn     MouseShape = "col-resize"
	MouseShapeResizeRow        MouseShape = "row-resize"
	MouseShapeResizeNorth      MouseShape = "n-resize"
	MouseShapeResizeEast       MouseShape = "e-resize"
	MouseShapeResizeSouth      MouseShape = "s-resize"
	MouseShapeResizeWest       MouseShape = "w-resize"
	MouseShapeResizeNorthEast  MouseShape = "ne-resize"
	MouseShapeResizeNorthWest  MouseShape = "nw-resize"
	MouseShapeResizeSouthEast  MouseShape = "se-resize"
	MouseShapeResizeSouthWest  MouseShape = "sw-resize"
	MouseShapeResizeHorizontal MouseShape = "ew-resize"
	MouseShapeResizeVertical   MouseShape = "ns-resize"
	MouseShapeResizeNESW       MouseShape = "nesw-resize"
	MouseShapeResizeNWSE       MouseShape = "nwse-resize"
	MouseShapeZoomIn           MouseShape = "zoom-in"
	MouseShapeZoomOut          MouseShape = "zoom-out"
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
	intermediates := seq.Intermediates()
	if len(intermediates) != 1 || intermediates[0] != '<' {
		log.Error("[CSI] unknown sequence: %s", seq)
		return mouse, false
	}

	if seq.NumParameters != 3 {
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
	button := seq.Param(0) & buttonBits
	mouse.Button = MouseButton(button)

	if seq.Param(0)&motion != 0 {
		mouse.EventType = EventMotion
	}

	if seq.Param(0)&mouseModShift != 0 {
		mouse.Modifiers |= ModShift
	}
	if seq.Param(0)&mouseModAlt != 0 {
		mouse.Modifiers |= ModAlt
	}
	if seq.Param(0)&mouseModCtrl != 0 {
		mouse.Modifiers |= ModCtrl
	}

	if enableSGRPixels {
		mouse.XPixel = seq.Param(1)
		mouse.YPixel = seq.Param(2)
		mouse.Col = pixelToCell(mouse.XPixel, ws.XPixel, ws.Cols)
		mouse.Row = pixelToCell(mouse.YPixel, ws.YPixel, ws.Rows)
	} else {
		mouse.Col = seq.Param(1) - 1
		mouse.Row = seq.Param(2) - 1
	}

	return mouse, true
}
