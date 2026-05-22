package ui

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
