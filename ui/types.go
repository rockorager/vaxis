// Package ui is a Flutter-inspired UI framework for Vaxis.
package ui

import (
	"math"

	"git.sr.ht/~rockorager/vaxis"
)

type Cell = vaxis.Cell
type Character = vaxis.Character
type Style = vaxis.Style
type Color = vaxis.Color
type AttributeMask = vaxis.AttributeMask
type Segment = vaxis.Segment

type Event = vaxis.Event
type Key = vaxis.Key
type Mouse = vaxis.Mouse
type MouseButton = vaxis.MouseButton
type FocusIn = vaxis.FocusIn
type FocusOut = vaxis.FocusOut
type Resize = vaxis.Resize
type Redraw = vaxis.Redraw
type SyncFunc = vaxis.SyncFunc
type MouseShape = vaxis.MouseShape
type Image = vaxis.Image

func RGB(r, g, b uint8) Color { return vaxis.RGBColor(r, g, b) }

const (
	MouseLeftButton   = vaxis.MouseLeftButton
	MouseMiddleButton = vaxis.MouseMiddleButton
	MouseRightButton  = vaxis.MouseRightButton
	EventPress        = vaxis.EventPress
	EventMotion       = vaxis.EventMotion
	KeyBackspace      = vaxis.KeyBackspace
	KeyDelete         = vaxis.KeyDelete
	KeyLeft           = vaxis.KeyLeft
	KeyRight          = vaxis.KeyRight
	KeyHome           = vaxis.KeyHome
	KeyEnd            = vaxis.KeyEnd
)

const (
	AttrNone          = vaxis.AttrNone
	AttrBold          = vaxis.AttrBold
	AttrDim           = vaxis.AttrDim
	AttrItalic        = vaxis.AttrItalic
	AttrBlink         = vaxis.AttrBlink
	AttrReverse       = vaxis.AttrReverse
	AttrInvisible     = vaxis.AttrInvisible
	AttrStrikethrough = vaxis.AttrStrikethrough
	AttrOverline      = vaxis.AttrOverline
)

const (
	MouseShapeDefault          = vaxis.MouseShapeDefault
	MouseShapeContextMenu      = vaxis.MouseShapeContextMenu
	MouseShapeTextInput        = vaxis.MouseShapeTextInput
	MouseShapeVerticalText     = vaxis.MouseShapeVerticalText
	MouseShapeClickable        = vaxis.MouseShapeClickable
	MouseShapeHelp             = vaxis.MouseShapeHelp
	MouseShapeBusyBackground   = vaxis.MouseShapeBusyBackground
	MouseShapeBusy             = vaxis.MouseShapeBusy
	MouseShapeAlias            = vaxis.MouseShapeAlias
	MouseShapeCopy             = vaxis.MouseShapeCopy
	MouseShapeMove             = vaxis.MouseShapeMove
	MouseShapeNoDrop           = vaxis.MouseShapeNoDrop
	MouseShapeNotAllowed       = vaxis.MouseShapeNotAllowed
	MouseShapeGrab             = vaxis.MouseShapeGrab
	MouseShapeGrabbing         = vaxis.MouseShapeGrabbing
	MouseShapeAllScroll        = vaxis.MouseShapeAllScroll
	MouseShapeCrosshair        = vaxis.MouseShapeCrosshair
	MouseShapeResizeColumn     = vaxis.MouseShapeResizeColumn
	MouseShapeResizeRow        = vaxis.MouseShapeResizeRow
	MouseShapeResizeNorth      = vaxis.MouseShapeResizeNorth
	MouseShapeResizeEast       = vaxis.MouseShapeResizeEast
	MouseShapeResizeSouth      = vaxis.MouseShapeResizeSouth
	MouseShapeResizeWest       = vaxis.MouseShapeResizeWest
	MouseShapeResizeNorthEast  = vaxis.MouseShapeResizeNorthEast
	MouseShapeResizeNorthWest  = vaxis.MouseShapeResizeNorthWest
	MouseShapeResizeSouthEast  = vaxis.MouseShapeResizeSouthEast
	MouseShapeResizeSouthWest  = vaxis.MouseShapeResizeSouthWest
	MouseShapeResizeHorizontal = vaxis.MouseShapeResizeHorizontal
	MouseShapeResizeVertical   = vaxis.MouseShapeResizeVertical
	MouseShapeResizeNESW       = vaxis.MouseShapeResizeNESW
	MouseShapeResizeNWSE       = vaxis.MouseShapeResizeNWSE
	MouseShapeZoomIn           = vaxis.MouseShapeZoomIn
	MouseShapeZoomOut          = vaxis.MouseShapeZoomOut
	MouseShapeCell             = vaxis.MouseShapeCell
)

type Widget = any

type KeyValue string

type Keyed interface {
	WidgetKey() KeyValue
}

type Point struct{ X, Y int }
type Offset struct{ X, Y int }
type Size struct{ Width, Height int }
type Rect struct{ X, Y, Width, Height int }

func (o Offset) Add(other Offset) Offset { return Offset{X: o.X + other.X, Y: o.Y + other.Y} }

const Unbounded = math.MaxInt

type Constraints struct {
	MinWidth, MaxWidth   int
	MinHeight, MaxHeight int
}

func Tight(size Size) Constraints {
	return Constraints{MinWidth: size.Width, MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height}
}

func Loose(size Size) Constraints {
	return Constraints{MaxWidth: size.Width, MaxHeight: size.Height}
}

func (c Constraints) HasBoundedWidth() bool  { return c.MaxWidth != Unbounded }
func (c Constraints) HasBoundedHeight() bool { return c.MaxHeight != Unbounded }

func (c Constraints) Constrain(size Size) Size {
	if size.Width < c.MinWidth {
		size.Width = c.MinWidth
	}
	if size.Height < c.MinHeight {
		size.Height = c.MinHeight
	}
	if c.HasBoundedWidth() && size.Width > c.MaxWidth {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() && size.Height > c.MaxHeight {
		size.Height = c.MaxHeight
	}
	return size
}

func (c Constraints) Enforce(other Constraints) Constraints {
	return Constraints{
		MinWidth:  clamp(c.MinWidth, other.MinWidth, other.MaxWidth),
		MaxWidth:  clamp(c.MaxWidth, other.MinWidth, other.MaxWidth),
		MinHeight: clamp(c.MinHeight, other.MinHeight, other.MaxHeight),
		MaxHeight: clamp(c.MaxHeight, other.MinHeight, other.MaxHeight),
	}
}

func (c Constraints) Deflate(in Insets) Constraints {
	dw, dh := in.Left+in.Right, in.Top+in.Bottom
	maxW, maxH := c.MaxWidth, c.MaxHeight
	if maxW != Unbounded {
		maxW = max(0, maxW-dw)
	}
	if maxH != Unbounded {
		maxH = max(0, maxH-dh)
	}
	return Constraints{MinWidth: max(0, c.MinWidth-dw), MaxWidth: maxW, MinHeight: max(0, c.MinHeight-dh), MaxHeight: maxH}
}

type Insets struct{ Top, Right, Bottom, Left int }

func All(v int) Insets { return Insets{Top: v, Right: v, Bottom: v, Left: v} }
func Symmetric(horizontal, vertical int) Insets {
	return Insets{Top: vertical, Bottom: vertical, Left: horizontal, Right: horizontal}
}

func (s Size) Inflate(in Insets) Size {
	return Size{Width: s.Width + in.Left + in.Right, Height: s.Height + in.Top + in.Bottom}
}

func clamp(v, minValue, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if maxValue != Unbounded && v > maxValue {
		return maxValue
	}
	return v
}
