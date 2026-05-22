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

// ScrollPaneController controls a mounted ScrollPane.
//
// Methods return false when the controller is not attached to a mounted pane or
// when the requested scroll command does not change the current offset. Metrics
// returns zero values until the controlled pane has been laid out.
type ScrollPaneController struct {
	target scrollAxisOffsetController
	owner  any
}

func (c *ScrollPaneController) attach(owner any) {
	c.owner = owner
	if target, ok := owner.(scrollAxisOffsetController); ok {
		c.target = target
	}
}

func (c *ScrollPaneController) detach(owner any) {
	if c.owner != owner {
		return
	}
	c.owner = nil
	c.target = nil
}

// Attached reports whether the controller is attached to a mounted pane.
func (c *ScrollPaneController) Attached() bool {
	return c != nil && c.target != nil
}

// ScrollBy scrolls by columns and rows.
func (c *ScrollPaneController) ScrollBy(cols, rows int) bool {
	if !c.Attached() {
		return false
	}
	colChanged := c.target.ScrollByLinesAxis(ScrollHorizontal, cols)
	rowChanged := c.target.ScrollByLinesAxis(ScrollVertical, rows)
	return colChanged || rowChanged
}

// ScrollTo scrolls to column and row.
func (c *ScrollPaneController) ScrollTo(col, row int) bool {
	if !c.Attached() {
		return false
	}
	colChanged := c.target.ScrollToOffsetAxis(ScrollHorizontal, col)
	rowChanged := c.target.ScrollToOffsetAxis(ScrollVertical, row)
	return colChanged || rowChanged
}

// ScrollToStart scrolls both axes to the start.
func (c *ScrollPaneController) ScrollToStart() bool {
	if !c.Attached() {
		return false
	}
	colChanged := c.target.ScrollToStartAxis(ScrollHorizontal)
	rowChanged := c.target.ScrollToStartAxis(ScrollVertical)
	return colChanged || rowChanged
}

// ScrollToEnd scrolls both axes to the end.
func (c *ScrollPaneController) ScrollToEnd() bool {
	if !c.Attached() {
		return false
	}
	colChanged := c.target.ScrollToEndAxis(ScrollHorizontal)
	rowChanged := c.target.ScrollToEndAxis(ScrollVertical)
	return colChanged || rowChanged
}

// Metrics returns the current scroll metrics for axis.
func (c *ScrollPaneController) Metrics(axis ScrollAxis) ScrollMetrics {
	if !c.Attached() {
		return ScrollMetrics{}
	}
	return c.target.ScrollMetricsForAxis(axis)
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
	RevealIndex(int) bool
	OffsetForIndex(int) (int, bool)
	VisibleRange() (int, int, bool)
}

type sliverTableScrollTarget interface {
	ScrollToRow(int, ScrollAlign) bool
	RevealRow(int) bool
	OffsetForRow(int) (int, bool)
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

// RevealIndex scrolls only enough to reveal index.
func (c *SliverListController) RevealIndex(index int) bool {
	if !c.Attached() {
		return false
	}
	return c.target.RevealIndex(index)
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

// SliverTableController controls a mounted SliverTableBuilder by row index.
//
// The table must be mounted inside a CustomScrollView for ScrollToRow and
// RevealRow to move the viewport. Variable-height rows use measured extents for
// rows that have been laid out and EstimatedRowExtent for unknown rows.
type SliverTableController struct {
	target sliverTableScrollTarget
	owner  any
}

func (c *SliverTableController) attach(owner any) {
	c.owner = owner
	if target, ok := owner.(sliverTableScrollTarget); ok {
		c.target = target
	}
}

func (c *SliverTableController) detach(owner any) {
	if c.owner != owner {
		return
	}
	c.owner = nil
	c.target = nil
}

// Attached reports whether the controller is attached to a mounted table.
func (c *SliverTableController) Attached() bool {
	return c != nil && c.target != nil
}

// ScrollToRow scrolls the containing CustomScrollView to row.
func (c *SliverTableController) ScrollToRow(row int, align ScrollAlign) bool {
	if !c.Attached() {
		return false
	}
	return c.target.ScrollToRow(row, align)
}

// RevealRow scrolls only enough to reveal row.
func (c *SliverTableController) RevealRow(row int) bool {
	if !c.Attached() {
		return false
	}
	return c.target.RevealRow(row)
}

// OffsetForRow returns the table-local row offset for row.
func (c *SliverTableController) OffsetForRow(row int) (int, bool) {
	if !c.Attached() {
		return 0, false
	}
	return c.target.OffsetForRow(row)
}

// VisibleRange returns the first and exclusive-last visible row indices.
func (c *SliverTableController) VisibleRange() (int, int, bool) {
	if !c.Attached() {
		return 0, 0, false
	}
	return c.target.VisibleRange()
}
