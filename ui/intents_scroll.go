package ui

// ScrollDirection identifies the direction of a scroll command.
type ScrollDirection int

const (
	// ScrollBackward scrolls toward the start of the scroll axis.
	ScrollBackward ScrollDirection = iota
	// ScrollForward scrolls toward the end of the scroll axis.
	ScrollForward
)

// ScrollUnit identifies the unit for a scroll command.
type ScrollUnit int

const (
	// ScrollUnitLine scrolls by one row or column.
	ScrollUnitLine ScrollUnit = iota
	// ScrollUnitPage scrolls by one viewport.
	ScrollUnitPage
	// ScrollUnitEdge scrolls to the start or end.
	ScrollUnitEdge
)

const (
	// ScrollIntentType scrolls a viewport.
	ScrollIntentType IntentType = "vaxis.scroll"
)

// ScrollIntent scrolls a viewport along an axis.
type ScrollIntent struct {
	Axis      ScrollAxis
	Direction ScrollDirection
	Unit      ScrollUnit
}

func (ScrollIntent) IntentType() IntentType {
	return ScrollIntentType
}
