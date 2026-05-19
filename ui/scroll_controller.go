package ui

// ScrollController controls a mounted CustomScrollView.
//
// Methods return false when the controller is not attached to a mounted view or
// when the requested scroll command does not change the current offset. Metrics
// returns zero values until the controlled view has been laid out.
type ScrollController struct {
	target scrollOffsetController
	owner  any
}

func (c *ScrollController) attach(owner any) {
	c.owner = owner
	if target, ok := owner.(scrollOffsetController); ok {
		c.target = target
	}
}

func (c *ScrollController) detach(owner any) {
	if c.owner != owner {
		return
	}
	c.owner = nil
	c.target = nil
}

// Attached reports whether the controller is attached to a mounted view.
func (c *ScrollController) Attached() bool {
	return c != nil && c.target != nil
}

// ScrollByLines scrolls by line rows.
func (c *ScrollController) ScrollByLines(lines int) bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollByLines(lines)
}

// ScrollByPages scrolls by page viewports.
func (c *ScrollController) ScrollByPages(pages int) bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollByPages(pages)
}

// ScrollToOffset scrolls to row.
func (c *ScrollController) ScrollToOffset(row int) bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollToOffset(row)
}

// ScrollToStart scrolls to the first row.
func (c *ScrollController) ScrollToStart() bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollToStart()
}

// ScrollToEnd scrolls to the last valid row.
func (c *ScrollController) ScrollToEnd() bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollToEnd()
}

// Metrics returns the current scroll metrics.
func (c *ScrollController) Metrics() ScrollMetrics {
	if !c.Attached() {
		return ScrollMetrics{}
	}
	return c.target.ScrollMetrics()
}

// ScrollAlign controls how a scrolled-to item is placed in the viewport.
type ScrollAlign int

const (
	// ScrollAlignStart places the item at the top of the viewport.
	ScrollAlignStart ScrollAlign = iota
	// ScrollAlignCenter centers the item in the viewport.
	ScrollAlignCenter
	// ScrollAlignEnd places the item at the bottom of the viewport.
	ScrollAlignEnd
	// ScrollAlignNearest scrolls only enough to reveal the item.
	ScrollAlignNearest
)

type sliverListScrollTarget interface {
	ScrollToIndex(int, ScrollAlign) bool
	OffsetForIndex(int) (int, bool)
	VisibleRange() (int, int, bool)
}

// SliverListController controls a mounted SliverListBuilder by item index.
//
// The list must be mounted inside a CustomScrollView for ScrollToIndex to move
// the viewport. Variable-height lists use measured extents for rows that have
// been laid out and EstimatedItemExtent for unknown rows.
type SliverListController struct {
	target sliverListScrollTarget
	owner  any
}

func (c *SliverListController) attach(owner any) {
	c.owner = owner
	if target, ok := owner.(sliverListScrollTarget); ok {
		c.target = target
	}
}

func (c *SliverListController) detach(owner any) {
	if c.owner != owner {
		return
	}
	c.owner = nil
	c.target = nil
}

// Attached reports whether the controller is attached to a mounted list.
func (c *SliverListController) Attached() bool {
	return c != nil && c.target != nil
}

// ScrollToIndex scrolls the containing CustomScrollView to index.
func (c *SliverListController) ScrollToIndex(index int, align ScrollAlign) bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollToIndex(index, align)
}

// OffsetForIndex returns the list-local row offset for index.
func (c *SliverListController) OffsetForIndex(index int) (int, bool) {
	if !c.Attached() {
		return 0, false
	}
	return c.target.OffsetForIndex(index)
}

// VisibleRange returns the first and exclusive-last visible item indices.
func (c *SliverListController) VisibleRange() (int, int, bool) {
	if !c.Attached() {
		return 0, 0, false
	}
	return c.target.VisibleRange()
}
