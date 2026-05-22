package ui

import "strconv"

type tableColumnKind int

const (
	tableColumnIntrinsic tableColumnKind = iota
	tableColumnFixed
	tableColumnFlex
)

// TableColumn describes how a Table column chooses its width.
type TableColumn struct {
	kind  tableColumnKind
	value int
}

// IntrinsicColumn sizes a column to the widest cell in that column.
func IntrinsicColumn() TableColumn {
	return TableColumn{kind: tableColumnIntrinsic}
}

// FixedColumn sizes a column to width cells.
func FixedColumn(width int) TableColumn {
	return TableColumn{kind: tableColumnFixed, value: width}
}

// FlexColumn gives a column a proportional share of remaining width.
func FlexColumn(flex int) TableColumn {
	return TableColumn{kind: tableColumnFlex, value: flex}
}

// TableRow is one row of widgets in a Table.
type TableRow struct {
	Children []Widget
}

// Table lays out widgets in rows and columns.
//
// Columns controls each column's width. When Columns is empty, the table infers
// a column for the widest row and sizes every column intrinsically.
type Table struct {
	Columns   []TableColumn
	ColumnGap int
	RowGap    int
	Rows      []TableRow
}

func (w Table) WidgetChildren() []Widget {
	children := make([]Widget, 0, tableChildCount(w.Rows))
	for _, row := range w.Rows {
		children = append(children, row.Children...)
	}
	return children
}

func (w Table) CreateRenderObject(BuildContext) RenderObject {
	return &renderTable{Columns: w.Columns, RowLengths: tableRowLengths(w.Rows), ColumnGap: w.ColumnGap, RowGap: w.RowGap}
}

func (w Table) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderTable)
	r.Columns = w.Columns
	r.RowLengths = tableRowLengths(w.Rows)
	r.ColumnGap = w.ColumnGap
	r.RowGap = w.RowGap
	r.MarkNeedsLayout()
}

func tableChildCount(rows []TableRow) int {
	n := 0
	for _, row := range rows {
		n += len(row.Children)
	}
	return n
}

func tableRowLengths(rows []TableRow) []int {
	lengths := make([]int, len(rows))
	for i, row := range rows {
		lengths[i] = len(row.Children)
	}
	return lengths
}

// TableParentData stores layout data for children of Table.
type TableParentData struct {
	Offset Offset
}

// RenderOffset returns the child's paint offset.
func (d TableParentData) RenderOffset() Offset {
	return d.Offset
}

type renderTable struct {
	MultiChildRenderObject
	Columns    []TableColumn
	RowLengths []int
	ColumnGap  int
	RowGap     int
}

func (r *renderTable) Layout(ctx LayoutContext, c Constraints) {
	size, _, _, offsets := r.layout(ctx, c, false)
	for i, child := range r.Children() {
		pd, _ := child.Base().ParentData().(TableParentData)
		pd.Offset = offsets[i]
		child.Base().SetParentData(pd)
	}
	r.SetSize(size)
}

func (r *renderTable) DryLayout(ctx LayoutContext, c Constraints) Size {
	size, _, _, _ := r.layout(ctx, c, true)
	return size
}

func (r *renderTable) layout(ctx LayoutContext, c Constraints, dry bool) (Size, []int, []int, []Offset) {
	columns := r.resolvedColumns()
	columnWidths := r.columnWidths(ctx, c, columns)
	rowHeights := make([]int, len(r.RowLengths))
	offsets := make([]Offset, len(r.Children()))
	childIndex := 0
	y := 0
	rowGap := max(0, r.RowGap)
	for rowIndex, rowLength := range r.RowLengths {
		x := 0
		for col := 0; col < rowLength; col++ {
			child := r.Children()[childIndex]
			if col < len(columnWidths) {
				s := r.layoutChild(ctx, child, tableCellConstraints(c, columnWidths[col]), dry)
				rowHeights[rowIndex] = max(rowHeights[rowIndex], s.Height)
				offsets[childIndex] = Offset{X: x, Y: y}
				x += columnWidths[col] + max(0, r.ColumnGap)
			}
			childIndex++
		}
		y += rowHeights[rowIndex]
		if rowIndex < len(r.RowLengths)-1 {
			y += rowGap
		}
	}
	width := tableUsedWidth(columnWidths, max(0, r.ColumnGap))
	return c.Constrain(Size{Width: width, Height: y}), columnWidths, rowHeights, offsets
}

func (r *renderTable) resolvedColumns() []TableColumn {
	cols := 0
	for _, n := range r.RowLengths {
		cols = max(cols, n)
	}
	if len(r.Columns) > 0 {
		columns := make([]TableColumn, max(len(r.Columns), cols))
		copy(columns, r.Columns)
		for i := len(r.Columns); i < len(columns); i++ {
			columns[i] = IntrinsicColumn()
		}
		return columns
	}
	columns := make([]TableColumn, cols)
	for i := range columns {
		columns[i] = IntrinsicColumn()
	}
	return columns
}

func (r *renderTable) columnWidths(ctx LayoutContext, c Constraints, columns []TableColumn) []int {
	widths := make([]int, len(columns))
	flexTotal := 0
	for i, col := range columns {
		switch col.kind {
		case tableColumnFixed:
			widths[i] = max(0, col.value)
		case tableColumnFlex:
			flexTotal += max(1, col.value)
		default:
			widths[i] = r.intrinsicColumnWidth(ctx, c, i)
		}
	}
	if c.MaxWidth == Unbounded || flexTotal == 0 {
		return widths
	}
	remaining := max(0, c.MaxWidth-tableUsedWidth(widths, max(0, r.ColumnGap)))
	remainingFlex := flexTotal
	for i, col := range columns {
		if col.kind != tableColumnFlex {
			continue
		}
		flex := max(1, col.value)
		share := 0
		if remainingFlex > 0 {
			share = remaining * flex / remainingFlex
		}
		widths[i] = share
		remaining -= share
		remainingFlex -= flex
	}
	return widths
}

func (r *renderTable) intrinsicColumnWidth(ctx LayoutContext, c Constraints, col int) int {
	width := 0
	childIndex := 0
	for _, rowLength := range r.RowLengths {
		for i := 0; i < rowLength; i++ {
			child := r.Children()[childIndex]
			if i == col {
				width = max(width, DryLayout(ctx, child, tableIntrinsicConstraints(c)).Width)
			}
			childIndex++
		}
	}
	return width
}

func tableIntrinsicConstraints(c Constraints) Constraints {
	return Constraints{MaxWidth: c.MaxWidth, MaxHeight: c.MaxHeight}
}

func tableCellConstraints(c Constraints, width int) Constraints {
	return Constraints{MinWidth: width, MaxWidth: width, MaxHeight: c.MaxHeight}
}

func tableUsedWidth(widths []int, gap int) int {
	used := 0
	for _, width := range widths {
		used += width
	}
	if len(widths) > 1 {
		used += gap * (len(widths) - 1)
	}
	return used
}

func (r *renderTable) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
	if dry {
		return DryLayout(ctx, child, c)
	}
	child.Layout(ctx, c)
	return child.Base().Size()
}

func (r *renderTable) Paint(p *Painter, off Offset) {
	for _, child := range r.Children() {
		pd, _ := child.Base().ParentData().(TableParentData)
		child.Paint(p, off.Add(pd.Offset))
	}
}

func (r *renderTable) HitTest(*HitTestResult, Point) bool {
	return false
}

// SliverTableBuilder lazily builds table rows for a CustomScrollView.
//
// It uses the same column sizing vocabulary as Table, but only materializes the
// visible rows plus Overscan. Row heights are measured as rows are laid out;
// unmeasured rows use EstimatedRowExtent.
type SliverTableBuilder struct {
	// Controller can be used to inspect and scroll this table by row index after
	// it is mounted in a CustomScrollView.
	Controller *SliverTableController
	// Columns controls each column's width. Missing columns are intrinsic.
	Columns []TableColumn
	// RowCount is the number of logical rows available from Builder.
	RowCount int
	// Builder returns the table row for row. It is only called for the active
	// visible range plus Overscan.
	Builder func(BuildContext, int) TableRow
	// EstimatedRowExtent is the height used for unmeasured rows. A zero or
	// negative value is treated as one row.
	EstimatedRowExtent int
	// Overscan builds this many extra rows before and after the viewport.
	Overscan int
}

func (w SliverTableBuilder) CreateState() State {
	return &sliverTableBuilderState{last: defaultSliverListBuilderInitialCount}
}

type sliverTableBuilderState struct {
	StateBase
	first         int
	last          int
	width         int
	extents       map[int]int
	intrinsic     []int
	pendingReveal pendingSliverReveal
}

func (s *sliverTableBuilderState) Build(ctx BuildContext) Widget {
	w := s.Widget().(SliverTableBuilder)
	s.attachController(w.Controller)
	count := max(0, w.RowCount)
	first := clampInt(s.first, 0, count)
	last := clampInt(s.last, first, count)
	children := make([]Widget, 0)
	rowLengths := make([]int, 0, max(0, last-first))
	if w.Builder != nil {
		for row := first; row < last; row++ {
			tableRow := w.Builder(ctx, row)
			rowLengths = append(rowLengths, len(tableRow.Children))
			for col, child := range tableRow.Children {
				if child == nil {
					child = SizedBox{Height: normalizeSliverEstimatedItemExtent(w.EstimatedRowExtent)}
				}
				children = append(children, sliverTableBuilderCell{
					Key:   KeyValue(strconv.Itoa(row) + ":" + strconv.Itoa(col)),
					Child: child,
				})
			}
		}
	}
	return sliverTableBuilderView{
		State:     s,
		Columns:   w.Columns,
		RowCount:  count,
		Estimate:  normalizeSliverEstimatedItemExtent(w.EstimatedRowExtent),
		Overscan:  max(0, w.Overscan),
		First:     first,
		Extents:   s.extentsForWidth(),
		Intrinsic: cloneIntSlice(s.intrinsic),
		Rows:      rowLengths,
		Children:  children,
	}
}

func (s *sliverTableBuilderState) DidUpdateWidget(old Widget) {
	next := s.Widget().(SliverTableBuilder).Controller
	prev := old.(SliverTableBuilder).Controller
	if prev != nil && prev != next {
		prev.detach(s)
	}
	s.attachController(next)
}

func (s *sliverTableBuilderState) Dispose() {
	if c := s.Widget().(SliverTableBuilder).Controller; c != nil {
		c.detach(s)
	}
}

func (s *sliverTableBuilderState) attachController(c *SliverTableController) {
	if c != nil {
		c.attach(s)
	}
}

func (s *sliverTableBuilderState) renderObject() *renderSliverTableBuilder {
	ro := s.Context().FindRenderObject()
	if r, ok := ro.(*renderSliverTableBuilder); ok {
		return r
	}
	return nil
}

func (s *sliverTableBuilderState) ScrollToRow(row int, align ScrollAlign) bool {
	if r := s.renderObject(); r != nil {
		return r.ScrollToRow(row, align)
	}
	return false
}

func (s *sliverTableBuilderState) RevealRow(row int) bool {
	return s.ScrollToRow(row, ScrollAlignNearest)
}

func (s *sliverTableBuilderState) OffsetForRow(row int) (int, bool) {
	if r := s.renderObject(); r != nil {
		return r.OffsetForRow(row)
	}
	return 0, false
}

func (s *sliverTableBuilderState) VisibleRange() (int, int, bool) {
	if r := s.renderObject(); r != nil {
		return r.VisibleRange()
	}
	return 0, 0, false
}

func (s *sliverTableBuilderState) CellAt(pt Point) (int, int, bool) {
	if r := s.renderObject(); r != nil {
		return r.CellAt(pt)
	}
	return 0, 0, false
}

func (s *sliverTableBuilderState) RowRect(row int) (Rect, bool) {
	if r := s.renderObject(); r != nil {
		return r.RowRect(row)
	}
	return Rect{}, false
}

func (s *sliverTableBuilderState) CellRect(row, col int) (Rect, bool) {
	if r := s.renderObject(); r != nil {
		return r.CellRect(row, col)
	}
	return Rect{}, false
}

func (s *sliverTableBuilderState) VisibleRows() []VisibleTableRow {
	if r := s.renderObject(); r != nil {
		return r.VisibleRows()
	}
	return nil
}

func (s *sliverTableBuilderState) extentsForWidth() map[int]int {
	if s.extents == nil {
		return nil
	}
	return s.extents
}

func (s *sliverTableBuilderState) updateLayout(width, first, last int, measured map[int]int, intrinsic []int) {
	reset := s.width != width
	changed := reset || first != s.first || last != s.last || !intSlicesEqualPrefix(s.intrinsic, intrinsic)
	if !changed {
		for row, extent := range measured {
			if s.extents == nil || s.extents[row] != extent {
				changed = true
				break
			}
		}
	}
	if !changed {
		return
	}
	s.SetState(func() {
		if reset || s.extents == nil {
			s.width = width
			s.extents = make(map[int]int)
			s.intrinsic = nil
		}
		for row, extent := range measured {
			s.extents[row] = extent
		}
		s.intrinsic = mergeMaxIntSlices(s.intrinsic, intrinsic)
		s.first = first
		s.last = last
	})
}

func (s *sliverTableBuilderState) setPendingReveal(row int, align ScrollAlign, first, last int) {
	s.SetState(func() {
		s.pendingReveal = pendingSliverReveal{Index: row, Align: align, Active: true}
		s.first = first
		s.last = last
	})
}

func (s *sliverTableBuilderState) clearPendingReveal() {
	s.pendingReveal = pendingSliverReveal{}
}

type sliverTableBuilderCell struct {
	Key   KeyValue
	Child Widget
}

func (w sliverTableBuilderCell) WidgetKey() KeyValue {
	return w.Key
}

func (w sliverTableBuilderCell) Build(BuildContext) Widget {
	return w.Child
}

type sliverTableBuilderView struct {
	State     *sliverTableBuilderState
	Columns   []TableColumn
	RowCount  int
	Estimate  int
	Overscan  int
	First     int
	Extents   map[int]int
	Intrinsic []int
	Rows      []int
	Children  []Widget
}

func (w sliverTableBuilderView) WidgetChildren() []Widget {
	return w.Children
}

func (w sliverTableBuilderView) CreateRenderObject(BuildContext) RenderObject {
	return &renderSliverTableBuilder{State: w.State, Columns: w.Columns, RowCount: w.RowCount, Estimate: w.Estimate, Overscan: w.Overscan, First: w.First, Extents: w.Extents, Intrinsic: w.Intrinsic, RowLengths: w.Rows}
}

func (w sliverTableBuilderView) UpdateRenderObject(_ BuildContext, ro RenderObject) {
	r := ro.(*renderSliverTableBuilder)
	r.State = w.State
	r.Columns = w.Columns
	r.RowCount = w.RowCount
	r.Estimate = w.Estimate
	r.Overscan = w.Overscan
	r.First = w.First
	r.Extents = w.Extents
	r.Intrinsic = w.Intrinsic
	r.RowLengths = w.Rows
	r.MarkNeedsLayout()
}

type renderSliverTableBuilder struct {
	MultiChildRenderObject
	State             *sliverTableBuilderState
	Columns           []TableColumn
	RowCount          int
	Estimate          int
	Overscan          int
	First             int
	Extents           map[int]int
	Intrinsic         []int
	RowLengths        []int
	geometry          SliverGeometry
	childOffsets      []Offset
	tableColumnWidths []int
	rowHeights        map[int]int
	constraints       SliverConstraints
}

func (r *renderSliverTableBuilder) Layout(ctx LayoutContext, c Constraints) {
	r.LayoutSliver(ctx, SliverConstraints{ViewportWidth: c.MaxWidth, ViewportHeight: c.MaxHeight, RemainingPaintExtent: c.MaxHeight})
}

func (r *renderSliverTableBuilder) LayoutSliver(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	r.constraints = c
	return r.layoutVariable(ctx, c)
}

func (r *renderSliverTableBuilder) layoutVariable(ctx LayoutContext, c SliverConstraints) SliverGeometry {
	cachedExtents := cloneSliverExtentCache(r.Extents)
	resized := r.State != nil && r.State.width != 0 && r.State.width != c.ViewportWidth
	anchorExtents := cachedExtents
	if resized {
		anchorExtents = cloneSliverExtentCache(r.Extents)
		cachedExtents = nil
	}
	model := measuredSliverExtentModel{Count: r.RowCount, Estimate: r.Estimate, Extents: cloneSliverExtentCache(cachedExtents)}
	anchorModel := measuredSliverExtentModel{Count: r.RowCount, Estimate: r.Estimate, Extents: anchorExtents}
	first, last := model.VisibleRange(r.Overscan, c)
	anchorScrollOffset := max(c.ScrollOffset, c.ObscuredLeadingExtent)
	anchorRow := anchorModel.IndexForOffset(anchorScrollOffset)
	anchorOffset := anchorModel.OffsetForIndex(anchorRow)
	anchorDelta := anchorScrollOffset - anchorOffset
	if resized {
		paintExtent := max(0, min(c.ViewportHeight, c.RemainingPaintExtent))
		first = clampInt(anchorRow-r.Overscan, 0, model.ItemCount())
		last = clampInt(anchorRow+(paintExtent+model.EstimatedExtent()-1)/model.EstimatedExtent()+r.Overscan+1, first, model.ItemCount())
	}
	if r.State != nil && r.State.pendingReveal.Active {
		first, last = pendingSliverRevealRange(r.State.pendingReveal.Index, r.Overscan, model, c, first, last)
	}
	columnWidths, intrinsic := r.columnWidths(ctx, c)
	r.tableColumnWidths = columnWidths
	children := r.Children()
	measured := make(map[int]int, len(r.RowLengths))
	r.childOffsets = make([]Offset, len(children))
	r.rowHeights = make(map[int]int, len(r.RowLengths))
	childIndex := 0
	width := tableUsedWidth(columnWidths, 0)
	for rowOffset, rowLength := range r.RowLengths {
		row := r.First + rowOffset
		x := 0
		rowHeight := 0
		for col := 0; col < rowLength && childIndex < len(children); col++ {
			child := children[childIndex]
			if col < len(columnWidths) {
				s := r.layoutChild(ctx, child, tableCellConstraints(Constraints{MaxHeight: Unbounded}, columnWidths[col]), false)
				rowHeight = max(rowHeight, s.Height)
				r.childOffsets[childIndex] = Offset{X: x, Y: model.OffsetForIndex(row)}
				x += columnWidths[col]
			}
			childIndex++
		}
		measured[row] = rowHeight
		r.rowHeights[row] = rowHeight
		model.Update(row, rowHeight)
	}
	scrollExtent := model.ScrollExtent()
	correction := 0
	if anchorScrollOffset > 0 && anchorRow < model.ItemCount() {
		correction = model.OffsetForIndex(anchorRow) + anchorDelta - anchorScrollOffset
	}
	if r.State != nil && r.State.pendingReveal.Active {
		pendingCorrection, done := pendingSliverRevealCorrection(r.State.pendingReveal, model, c)
		correction = pendingCorrection
		if done {
			r.State.clearPendingReveal()
		}
	}
	r.Extents = model.Extents
	if r.State != nil {
		r.State.updateLayout(c.ViewportWidth, first, last, measured, intrinsic)
	}
	r.SetSize(Size{Width: max(width, c.ViewportWidth), Height: scrollExtent})
	r.geometry = SliverGeometry{ScrollExtent: scrollExtent, PaintExtent: visibleSliverExtent(c, scrollExtent), ScrollOffsetCorrection: correction}
	return r.geometry
}

func (r *renderSliverTableBuilder) columnWidths(ctx LayoutContext, c SliverConstraints) ([]int, []int) {
	columns := r.resolvedColumns()
	widths := make([]int, len(columns))
	intrinsic := cloneIntSlice(r.Intrinsic)
	if len(intrinsic) < len(columns) {
		intrinsic = append(intrinsic, make([]int, len(columns)-len(intrinsic))...)
	}
	flexTotal := 0
	for i, col := range columns {
		switch col.kind {
		case tableColumnFixed:
			widths[i] = max(0, col.value)
		case tableColumnFlex:
			flexTotal += max(1, col.value)
		default:
			intrinsic[i] = max(intrinsic[i], r.intrinsicColumnWidth(ctx, c, i))
			widths[i] = intrinsic[i]
		}
	}
	if c.ViewportWidth == Unbounded || flexTotal == 0 {
		return widths, intrinsic
	}
	remaining := max(0, c.ViewportWidth-tableUsedWidth(widths, 0))
	remainingFlex := flexTotal
	for i, col := range columns {
		if col.kind != tableColumnFlex {
			continue
		}
		flex := max(1, col.value)
		share := 0
		if remainingFlex > 0 {
			share = remaining * flex / remainingFlex
		}
		widths[i] = share
		remaining -= share
		remainingFlex -= flex
	}
	return widths, intrinsic
}

func (r *renderSliverTableBuilder) resolvedColumns() []TableColumn {
	cols := 0
	for _, n := range r.RowLengths {
		cols = max(cols, n)
	}
	if len(r.Columns) > 0 {
		columns := make([]TableColumn, max(len(r.Columns), cols))
		copy(columns, r.Columns)
		for i := len(r.Columns); i < len(columns); i++ {
			columns[i] = IntrinsicColumn()
		}
		return columns
	}
	columns := make([]TableColumn, cols)
	for i := range columns {
		columns[i] = IntrinsicColumn()
	}
	return columns
}

func (r *renderSliverTableBuilder) intrinsicColumnWidth(ctx LayoutContext, c SliverConstraints, col int) int {
	width := 0
	childIndex := 0
	for _, rowLength := range r.RowLengths {
		for i := 0; i < rowLength && childIndex < len(r.Children()); i++ {
			child := r.Children()[childIndex]
			if i == col {
				width = max(width, DryLayout(ctx, child, Constraints{MaxWidth: c.ViewportWidth, MaxHeight: Unbounded}).Width)
			}
			childIndex++
		}
	}
	return width
}

func (r *renderSliverTableBuilder) layoutChild(ctx LayoutContext, child RenderObject, c Constraints, dry bool) Size {
	if dry {
		return DryLayout(ctx, child, c)
	}
	child.Layout(ctx, c)
	return child.Base().Size()
}

func (r *renderSliverTableBuilder) Paint(p *Painter, off Offset) {
	r.PaintSliver(p, off)
}

func (r *renderSliverTableBuilder) PaintSliver(p *Painter, off Offset) {
	for i, child := range r.Children() {
		if i >= len(r.childOffsets) {
			continue
		}
		child.Paint(p, off.Add(r.childOffsets[i]))
	}
}

func (r *renderSliverTableBuilder) HitTest(*HitTestResult, Point) bool {
	return false
}

func (r *renderSliverTableBuilder) ChildOffset(child RenderObject) Offset {
	for i, candidate := range r.Children() {
		if candidate == child && i < len(r.childOffsets) {
			return r.childOffsets[i]
		}
	}
	return Offset{}
}

func (r *renderSliverTableBuilder) SelectionChildOffset(child RenderObject) Offset {
	return r.ChildOffset(child)
}

func (r *renderSliverTableBuilder) SelectionSize() Size {
	return r.Size()
}

func (r *renderSliverTableBuilder) ScrollToRow(row int, align ScrollAlign) bool {
	parent, ok := r.Base().parent.(*renderCustomScrollView)
	if !ok {
		return false
	}
	offset, ok := r.OffsetForRow(row)
	if !ok {
		return false
	}
	if r.State != nil {
		first, last := pendingSliverRevealRange(row, r.Overscan, r.extentModel(), r.constraints, r.First, r.First+len(r.RowLengths))
		r.State.setPendingReveal(row, align, first, last)
	}
	extent := r.extentForRow(row)
	target := parent.SelectionChildOffset(r).Y + offset
	metrics := parent.ScrollMetrics()
	switch align {
	case ScrollAlignCenter:
		target += extent/2 - metrics.ViewportHeight/2
	case ScrollAlignEnd:
		target += extent - metrics.ViewportHeight
	case ScrollAlignNearest:
		current := metrics.ScrollOffset
		if target >= current && target+extent <= current+metrics.ViewportHeight {
			return false
		}
		if target >= current {
			target += extent - metrics.ViewportHeight
		}
	}
	return parent.ScrollToOffset(target)
}

func (r *renderSliverTableBuilder) RevealRow(row int) bool {
	return r.ScrollToRow(row, ScrollAlignNearest)
}

func (r *renderSliverTableBuilder) OffsetForRow(row int) (int, bool) {
	if row < 0 || row >= max(0, r.RowCount) {
		return 0, false
	}
	return r.extentModel().OffsetForIndex(row), true
}

func (r *renderSliverTableBuilder) VisibleRange() (int, int, bool) {
	if max(0, r.RowCount) == 0 {
		return 0, 0, true
	}
	first, last := r.extentModel().VisibleRange(0, r.constraints)
	return first, last, true
}

func (r *renderSliverTableBuilder) CellAt(pt Point) (int, int, bool) {
	parent, ok := r.Base().parent.(*renderCustomScrollView)
	if !ok || pt.X < 0 || pt.Y < 0 || pt.X >= parent.Size().Width || pt.Y >= parent.Size().Height {
		return 0, 0, false
	}
	baseY := parent.SelectionChildOffset(r).Y
	localY := parent.ScrollMetrics().ScrollOffset + pt.Y - baseY
	if localY < 0 || localY >= r.geometry.ScrollExtent {
		return 0, 0, false
	}
	row := r.extentModel().IndexForOffset(localY)
	if row < 0 || row >= r.RowCount {
		return 0, 0, false
	}
	col, ok := r.columnAt(pt.X)
	if !ok || col >= r.rowLength(row) {
		return 0, 0, false
	}
	return row, col, true
}

func (r *renderSliverTableBuilder) RowRect(row int) (Rect, bool) {
	parent, ok := r.Base().parent.(*renderCustomScrollView)
	if !ok || row < 0 || row >= r.RowCount {
		return Rect{}, false
	}
	y := parent.SelectionChildOffset(r).Y + r.extentModel().OffsetForIndex(row) - parent.ScrollMetrics().ScrollOffset
	return Rect{X: 0, Y: y, Width: r.Size().Width, Height: r.extentForRow(row)}, true
}

func (r *renderSliverTableBuilder) CellRect(row, col int) (Rect, bool) {
	if col < 0 || col >= len(r.tableColumnWidths) || col >= r.rowLength(row) {
		return Rect{}, false
	}
	rowRect, ok := r.RowRect(row)
	if !ok {
		return Rect{}, false
	}
	return Rect{X: r.columnOffset(col), Y: rowRect.Y, Width: r.tableColumnWidths[col], Height: rowRect.Height}, true
}

func (r *renderSliverTableBuilder) VisibleRows() []VisibleTableRow {
	first, last, ok := r.VisibleRange()
	if !ok || first == last {
		return nil
	}
	rows := make([]VisibleTableRow, 0, last-first)
	for row := first; row < last; row++ {
		rect, ok := r.RowRect(row)
		if !ok || rect.Y+rect.Height <= 0 || rect.Y >= r.constraints.ViewportHeight {
			continue
		}
		rows = append(rows, VisibleTableRow{Row: row, Rect: rect})
	}
	return rows
}

func (r *renderSliverTableBuilder) extentForRow(row int) int {
	return r.extentModel().ExtentForIndex(row)
}

func (r *renderSliverTableBuilder) extentModel() sliverExtentModel {
	return measuredSliverExtentModel{Count: r.RowCount, Estimate: r.Estimate, Extents: r.Extents}
}

func (r *renderSliverTableBuilder) columnAt(x int) (int, bool) {
	if x < 0 {
		return 0, false
	}
	left := 0
	for col, width := range r.tableColumnWidths {
		if x >= left && x < left+width {
			return col, true
		}
		left += width
	}
	return 0, false
}

func (r *renderSliverTableBuilder) columnOffset(col int) int {
	off := 0
	for i := 0; i < col && i < len(r.tableColumnWidths); i++ {
		off += r.tableColumnWidths[i]
	}
	return off
}

func (r *renderSliverTableBuilder) rowLength(row int) int {
	if row < r.First || row >= r.First+len(r.RowLengths) {
		return 0
	}
	return r.RowLengths[row-r.First]
}

func cloneIntSlice(v []int) []int {
	if v == nil {
		return nil
	}
	out := make([]int, len(v))
	copy(out, v)
	return out
}

func mergeMaxIntSlices(a, b []int) []int {
	out := cloneIntSlice(a)
	if len(out) < len(b) {
		out = append(out, make([]int, len(b)-len(out))...)
	}
	for i, v := range b {
		out[i] = max(out[i], v)
	}
	return out
}

func intSlicesEqualPrefix(a, b []int) bool {
	merged := mergeMaxIntSlices(a, b)
	if len(merged) != len(a) {
		return false
	}
	for i := range merged {
		if merged[i] != a[i] {
			return false
		}
	}
	return true
}
