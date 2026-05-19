package ui

import (
	"math"

	"git.sr.ht/~rockorager/vaxis"
)

type (
	// Cell aliases vaxis.Cell for convenience in ui code.
	Cell = vaxis.Cell
	// Character aliases vaxis.Character for convenience in ui code.
	Character = vaxis.Character
	// Style aliases vaxis.Style for convenience in ui code.
	Style = vaxis.Style
	// Color aliases vaxis.Color for convenience in ui code.
	Color = vaxis.Color
	// AttributeMask aliases vaxis.AttributeMask for convenience in ui code.
	AttributeMask = vaxis.AttributeMask
	// Segment aliases vaxis.Segment for convenience in ui code.
	Segment = vaxis.Segment
)

type (
	// Event aliases vaxis.Event.
	Event = vaxis.Event
	// Key aliases vaxis.Key.
	Key = vaxis.Key
	// Mouse aliases vaxis.Mouse.
	Mouse = vaxis.Mouse
	// MouseButton aliases vaxis.MouseButton.
	MouseButton = vaxis.MouseButton
	// FocusIn aliases vaxis.FocusIn.
	FocusIn = vaxis.FocusIn
	// FocusOut aliases vaxis.FocusOut.
	FocusOut = vaxis.FocusOut
	// Resize aliases vaxis.Resize.
	Resize = vaxis.Resize
	// Redraw aliases vaxis.Redraw.
	Redraw = vaxis.Redraw
	// SyncFunc aliases vaxis.SyncFunc.
	SyncFunc = vaxis.SyncFunc
	// MouseShape aliases vaxis.MouseShape.
	MouseShape = vaxis.MouseShape
	// Image aliases vaxis.Image.
	Image = vaxis.Image
	// CursorStyle aliases vaxis.CursorStyle.
	CursorStyle = vaxis.CursorStyle
)

// RGB returns a 24-bit RGB color.
func RGB(r, g, b uint8) Color {
	return vaxis.RGBColor(r, g, b)
}

const (
	// MouseLeftButton aliases vaxis.MouseLeftButton.
	MouseNoButton     = vaxis.MouseNoButton
	MouseLeftButton   = vaxis.MouseLeftButton
	MouseMiddleButton = vaxis.MouseMiddleButton
	MouseRightButton  = vaxis.MouseRightButton
	MouseWheelUp      = vaxis.MouseWheelUp
	MouseWheelDown    = vaxis.MouseWheelDown
	// EventPress aliases vaxis.EventPress.
	EventPress   = vaxis.EventPress
	EventRelease = vaxis.EventRelease
	EventMotion  = vaxis.EventMotion
	// KeyBackspace aliases vaxis.KeyBackspace.
	KeyBackspace = vaxis.KeyBackspace
	KeyDelete    = vaxis.KeyDelete
	KeyLeft      = vaxis.KeyLeft
	KeyUp        = vaxis.KeyUp
	KeyRight     = vaxis.KeyRight
	KeyDown      = vaxis.KeyDown
	KeyPgDown    = vaxis.KeyPgDown
	KeyPgUp      = vaxis.KeyPgUp
	KeyHome      = vaxis.KeyHome
	KeyEnd       = vaxis.KeyEnd
)

const (
	// AttrNone aliases vaxis.AttrNone.
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
	// MouseShapeDefault aliases vaxis.MouseShapeDefault.
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

const (
	// CursorDefault aliases vaxis.CursorDefault.
	CursorDefault           = vaxis.CursorDefault
	CursorBlockBlinking     = vaxis.CursorBlockBlinking
	CursorBlock             = vaxis.CursorBlock
	CursorUnderlineBlinking = vaxis.CursorUnderlineBlinking
	CursorUnderline         = vaxis.CursorUnderline
	CursorBeamBlinking      = vaxis.CursorBeamBlinking
	CursorBeam              = vaxis.CursorBeam
)

// Widget is any value that implements one of the widget interfaces.
type Widget = any

// KeyValue identifies a widget across rebuilds.
type KeyValue string

// Keyed gives a widget a stable identity among siblings of the same type.
type Keyed interface {
	WidgetKey() KeyValue
}

type (
	// Point is an absolute cell coordinate.
	Point struct{ X, Y int }
	// Offset is a relative cell coordinate.
	Offset struct{ X, Y int }
	// Size is a width and height in terminal cells.
	Size struct{ Width, Height int }
	// Rect is a rectangular cell region.
	Rect struct{ X, Y, Width, Height int }
)

// Add returns the sum of two offsets.
func (o Offset) Add(other Offset) Offset {
	return Offset{X: o.X + other.X, Y: o.Y + other.Y}
}

// Unbounded marks an unconstrained maximum size.
const Unbounded = math.MaxInt

// Constraints describes minimum and maximum sizes for layout.
type Constraints struct {
	// MinWidth is the smallest width a render object may choose.
	MinWidth int
	// MaxWidth is the largest width a render object may choose, or Unbounded.
	MaxWidth int
	// MinHeight is the smallest height a render object may choose.
	MinHeight int
	// MaxHeight is the largest height a render object may choose, or Unbounded.
	MaxHeight int
}

// Tight returns constraints that force exactly size.
func Tight(size Size) Constraints {
	return Constraints{MinWidth: size.Width, MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height}
}

// Loose returns constraints bounded by size with zero minimums.
func Loose(size Size) Constraints {
	return Constraints{MaxWidth: size.Width, MaxHeight: size.Height}
}

// HasBoundedWidth reports whether MaxWidth is finite.
func (c Constraints) HasBoundedWidth() bool {
	return c.MaxWidth != Unbounded
}

// HasBoundedHeight reports whether MaxHeight is finite.
func (c Constraints) HasBoundedHeight() bool {
	return c.MaxHeight != Unbounded
}

// Constrain clamps size into the constraint range.
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

// Enforce clamps c so it also satisfies other.
func (c Constraints) Enforce(other Constraints) Constraints {
	return Constraints{
		MinWidth:  clamp(c.MinWidth, other.MinWidth, other.MaxWidth),
		MaxWidth:  clamp(c.MaxWidth, other.MinWidth, other.MaxWidth),
		MinHeight: clamp(c.MinHeight, other.MinHeight, other.MaxHeight),
		MaxHeight: clamp(c.MaxHeight, other.MinHeight, other.MaxHeight),
	}
}

// Deflate subtracts insets from the constraint space.
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

// Insets describes top, right, bottom, and left padding.
type Insets struct {
	Top, Right, Bottom, Left int
}

// All returns equal insets on every side.
func All(v int) Insets {
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// Symmetric returns insets with shared horizontal and vertical values.
func Symmetric(horizontal, vertical int) Insets {
	return Insets{Top: vertical, Bottom: vertical, Left: horizontal, Right: horizontal}
}

// Inflate adds insets to a size.
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
