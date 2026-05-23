package ui

import (
	"fmt"
	"sort"
	"strings"
)

const (
	defaultFuzzySelectWidth          = 54
	defaultFuzzySelectMaxVisibleRows = 5
	fuzzySelectTopDivisor            = 4

	fuzzySelectSelectionIntentType IntentType = "ui.fuzzy-select.selection"
)

// FuzzySelectRowStyle controls how result rows are laid out.
type FuzzySelectRowStyle int

const (
	// FuzzySelectTwoLine shows title and description rows. This is the default.
	FuzzySelectTwoLine FuzzySelectRowStyle = iota
	// FuzzySelectOneLine shows only title text and uses one terminal row per item.
	FuzzySelectOneLine
)

// FuzzySelectFilter filters and ranks fuzzy select items for query.
type FuzzySelectFilter[T any] func(query string, items []T, item FuzzySelectItemFunc[T]) []T

// FuzzySelectItemFunc converts an item into searchable and renderable row data.
type FuzzySelectItemFunc[T any] func(T) FuzzySelectItem

// FuzzySelectItem describes one selectable fuzzy select row.
type FuzzySelectItem struct {
	// Title is the primary row text.
	Title string
	// Description is optional secondary row text.
	Description string
	// Aliases are additional strings matched by the default fuzzy filter.
	Aliases []string
	// Leading is painted before the title content when non-nil.
	Leading Widget
	// Trailing is painted at the end of the row when non-nil.
	Trailing Widget
	// Disabled prevents activation when true.
	Disabled bool
}

// FuzzySelect shows a searchable list of items in a floating panel.
type FuzzySelect[T any] struct {
	// Items are filtered, displayed, and activated by the picker.
	Items []T
	// Item converts an item into searchable and renderable row data.
	Item FuzzySelectItemFunc[T]
	// Filter filters and ranks Items for the current query. When nil,
	// DefaultFuzzySelectFilter is used.
	Filter FuzzySelectFilter[T]
	// Placeholder is shown in the search field when the query is empty.
	Placeholder string
	// EmptyText is shown when no items match the query.
	EmptyText string
	// Width is the panel content width when greater than zero.
	Width int
	// MaxVisibleRows limits visible result rows before scrolling.
	MaxVisibleRows int
	// RowStyle controls whether items render as one-line or two-line rows.
	RowStyle FuzzySelectRowStyle
	// OnDismiss is called when Escape is pressed.
	OnDismiss VoidCallback
	// OnSelected is called when an item is activated.
	OnSelected func(EventContext, T)
}

func (w FuzzySelect[T]) CreateState() State {
	return &fuzzySelectState[T]{}
}

// CommandPaletteFilter filters and ranks command palette items for query.
type CommandPaletteFilter func(query string, items []CommandPaletteItem) []CommandPaletteItem

// CommandPaletteSelectedCallback receives a selected command palette item.
type CommandPaletteSelectedCallback func(EventContext, CommandPaletteItem)

// CommandPaletteItem describes one selectable command palette row.
type CommandPaletteItem struct {
	// Title is the primary row text.
	Title string
	// Description is optional secondary row text.
	Description string
	// Aliases are additional strings matched by the default fuzzy filter.
	Aliases []string
	// Leading is painted before the title content when non-nil.
	Leading Widget
	// Trailing is painted at the end of the row when non-nil.
	Trailing Widget
	// Disabled prevents activation when true.
	Disabled bool
	// OnSelected is called when this item is activated.
	OnSelected VoidCallback
}

// CommandPalette shows a searchable list of commands in a floating panel.
type CommandPalette struct {
	// Items are filtered, displayed, and activated by the palette.
	Items []CommandPaletteItem
	// Filter filters and ranks Items for the current query. When nil,
	// DefaultCommandPaletteFilter is used.
	Filter CommandPaletteFilter
	// Placeholder is shown in the search field when the query is empty.
	Placeholder string
	// EmptyText is shown when no items match the query.
	EmptyText string
	// Width is the panel content width when greater than zero.
	Width int
	// MaxVisibleRows limits visible result rows before scrolling.
	MaxVisibleRows int
	// OnDismiss is called when Escape is pressed.
	OnDismiss VoidCallback
	// OnSelected is called after the selected item's OnSelected callback.
	OnSelected CommandPaletteSelectedCallback
}

func (w CommandPalette) Build(BuildContext) Widget {
	placeholder := w.Placeholder
	if placeholder == "" {
		placeholder = "Search commands…"
	}
	emptyText := w.EmptyText
	if emptyText == "" {
		emptyText = "No matching commands"
	}
	return FuzzySelect[CommandPaletteItem]{
		Items:          w.Items,
		Item:           commandPaletteSelectItem,
		Filter:         commandPaletteSelectFilter(w.Filter),
		Placeholder:    placeholder,
		EmptyText:      emptyText,
		Width:          w.Width,
		MaxVisibleRows: w.MaxVisibleRows,
		OnDismiss:      w.OnDismiss,
		OnSelected: func(ctx EventContext, item CommandPaletteItem) {
			if item.OnSelected != nil {
				item.OnSelected(ctx)
			}
			if w.OnSelected != nil {
				w.OnSelected(ctx, item)
			}
		},
	}
}

type fuzzySelectState[T any] struct {
	StateBase
	query          string
	selected       int
	listController ScrollPaneController
}

func (s *fuzzySelectState[T]) Build(ctx BuildContext) Widget {
	w := s.Widget().(FuzzySelect[T])
	items := fuzzySelectFilteredItems(w, s.query)
	selected := s.selected
	if len(items) > 0 {
		selected = clampInt(selected, 0, len(items)-1)
	}
	return s.view(ctx, w, items, selected)
}

func (s *fuzzySelectState[T]) view(ctx BuildContext, w FuzzySelect[T], items []T, selected int) Widget {
	theme := MustDepend[Theme](ctx)
	width := fuzzySelectWidth(w.Width)
	rowHeight := fuzzySelectRowHeight(w.RowStyle)
	placeholder := w.Placeholder
	if placeholder == "" {
		placeholder = "Search…"
	}
	children := []Widget{
		TextField{
			Value:       s.query,
			Placeholder: placeholder,
			MinWidth:    width,
			OnChanged:   s.setQuery,
			OnSubmitted: func(ctx EventContext, _ string) { s.run(ctx, items, selected) },
		},
		SizedBox{Height: 1},
	}
	if len(items) == 0 {
		empty := w.EmptyText
		if empty == "" {
			empty = "No matching items"
		}
		children = append(children, fuzzySelectList(&s.listController, width, 1, []Widget{
			Text{Value: empty, Style: Style{Foreground: theme.MutedForeground}},
		}))
	} else {
		rows := make([]Widget, 0, len(items))
		for i, item := range items {
			index := i
			item := item
			row := fuzzySelectItem(w, item)
			isSelected := index == selected
			rows = append(rows, fuzzySelectRow(theme, row, isSelected, rowHeight, func(ctx EventContext) {
				s.run(ctx, []T{item}, 0)
			}))
		}
		listHeight := min(len(items)*rowHeight, fuzzySelectMaxVisibleRows(w.MaxVisibleRows)*rowHeight)
		children = append(children, fuzzySelectList(&s.listController, width, listHeight, rows))
	}
	panelStyle := Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered}
	return Actions{
		Bindings: map[IntentType]ActionFunc{
			DismissIntentType: func(ctx EventContext, intent Intent) EventResult {
				if w.OnDismiss != nil {
					w.OnDismiss(ctx)
				}
				return EventHandled
			},
			fuzzySelectSelectionIntentType: func(ctx EventContext, intent Intent) EventResult {
				s.moveSelection(intent.(fuzzySelectSelectionIntent).Delta)
				return EventHandled
			},
		},
		Child: Shortcuts{
			Bindings: ShortcutMap{
				"Down":   fuzzySelectSelectionIntent{Delta: 1},
				"Ctrl+n": fuzzySelectSelectionIntent{Delta: 1},
				"Up":     fuzzySelectSelectionIntent{Delta: -1},
				"Ctrl+p": fuzzySelectSelectionIntent{Delta: -1},
			},
			Child: fuzzySelectPositioner{Child: FocusScope{Trap: true, AutoFocus: true, Child: DecoratedBox(
				Decoration{Style: panelStyle},
				Padding(Symmetric(2, 1), ConstrainedBox{
					Constraints: Constraints{MinWidth: width, MaxWidth: width},
					Child:       Flex{Axis: Vertical, MainAxisSize: MainAxisSizeMin, CrossAxisAlignment: CrossAxisStretch, Children: children},
				}),
			)}},
		},
	}
}

func fuzzySelectRow(theme Theme, row FuzzySelectItem, selected bool, rowHeight int, onPressed VoidCallback) Widget {
	title := Text{Value: row.Title, Style: fuzzySelectPrimaryTextStyle(theme, selected), MaxLines: 1, Overflow: TextOverflowEllipsis}
	tile := ListTile{
		Leading:   row.Leading,
		Trailing:  row.Trailing,
		Title:     title,
		Selected:  selected,
		Disabled:  row.Disabled,
		OnPressed: onPressed,
		MinHeight: rowHeight,
	}
	if rowHeight > 1 {
		tile.Subtitle = Text{Value: row.Description, Style: fuzzySelectSecondaryTextStyle(theme, selected), MaxLines: 1, Overflow: TextOverflowEllipsis}
	}
	return tile
}

func (s *fuzzySelectState[T]) setQuery(_ EventContext, query string) {
	s.SetState(func() {
		s.query = query
		s.selected = 0
	})
	s.listController.ScrollToStart()
}

func (s *fuzzySelectState[T]) moveSelection(delta int) {
	items := fuzzySelectFilteredItems(s.Widget().(FuzzySelect[T]), s.query)
	if len(items) == 0 {
		return
	}
	next := clampInt(s.selected+delta, 0, len(items)-1)
	s.SetState(func() { s.selected = next })
	s.revealSelection(next)
}

func (s *fuzzySelectState[T]) revealSelection(index int) {
	metrics := s.listController.Metrics(ScrollVertical)
	if metrics.ViewportHeight == 0 {
		return
	}
	rowHeight := fuzzySelectRowHeight(s.Widget().(FuzzySelect[T]).RowStyle)
	top := index * rowHeight
	bottom := top + rowHeight
	visibleTop := metrics.ScrollOffset
	visibleBottom := metrics.ScrollOffset + metrics.ViewportHeight
	switch {
	case top < visibleTop:
		s.listController.ScrollTo(0, top)
	case bottom > visibleBottom:
		s.listController.ScrollTo(0, bottom-metrics.ViewportHeight)
	}
}

func (s *fuzzySelectState[T]) run(ctx EventContext, items []T, index int) {
	if len(items) == 0 {
		return
	}
	w := s.Widget().(FuzzySelect[T])
	item := items[clampInt(index, 0, len(items)-1)]
	if fuzzySelectItem(w, item).Disabled {
		return
	}
	if w.OnDismiss != nil {
		w.OnDismiss(ctx)
	}
	if w.OnSelected != nil {
		w.OnSelected(ctx, item)
	}
}

type fuzzySelectSelectionIntent struct{ Delta int }

func (i fuzzySelectSelectionIntent) IntentType() IntentType {
	return fuzzySelectSelectionIntentType
}

func commandPaletteSelectItem(item CommandPaletteItem) FuzzySelectItem {
	return FuzzySelectItem{
		Title:       item.Title,
		Description: item.Description,
		Aliases:     item.Aliases,
		Leading:     item.Leading,
		Trailing:    item.Trailing,
		Disabled:    item.Disabled,
	}
}

func commandPaletteSelectFilter(filter CommandPaletteFilter) FuzzySelectFilter[CommandPaletteItem] {
	if filter == nil {
		return func(query string, items []CommandPaletteItem, item FuzzySelectItemFunc[CommandPaletteItem]) []CommandPaletteItem {
			return DefaultCommandPaletteFilter(query, items)
		}
	}
	return func(query string, items []CommandPaletteItem, item FuzzySelectItemFunc[CommandPaletteItem]) []CommandPaletteItem {
		return filter(query, items)
	}
}

// DefaultCommandPaletteFilter filters command items with title-weighted fuzzy matching.
func DefaultCommandPaletteFilter(query string, items []CommandPaletteItem) []CommandPaletteItem {
	return DefaultFuzzySelectFilter(query, items, commandPaletteSelectItem)
}

func fuzzySelectFilteredItems[T any](w FuzzySelect[T], query string) []T {
	filter := w.Filter
	if filter == nil {
		filter = DefaultFuzzySelectFilter[T]
	}
	return filter(query, w.Items, w.Item)
}

func fuzzySelectItem[T any](w FuzzySelect[T], item T) FuzzySelectItem {
	return fuzzySelectItemForFunc(item, w.Item)
}

func fuzzySelectItemForFunc[T any](item T, fn FuzzySelectItemFunc[T]) FuzzySelectItem {
	if fn == nil {
		return FuzzySelectItem{Title: strings.TrimSpace(fmt.Sprint(item))}
	}
	return fn(item)
}

// DefaultFuzzySelectFilter filters items with title-weighted fuzzy matching.
func DefaultFuzzySelectFilter[T any](query string, items []T, item FuzzySelectItemFunc[T]) []T {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return items
	}
	type rankedItem struct {
		item  T
		score int
		index int
	}
	ranked := []rankedItem{}
	for i, value := range items {
		if score, ok := fuzzySelectItemMatchScore(fuzzySelectItemForFunc(value, item), query); ok {
			ranked = append(ranked, rankedItem{item: value, score: score, index: i})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].index < ranked[j].index
	})
	out := make([]T, 0, len(ranked))
	for _, match := range ranked {
		out = append(out, match.item)
	}
	return out
}

func fuzzySelectItemMatchScore(item FuzzySelectItem, query string) (int, bool) {
	candidates := append([]string{item.Title, item.Description}, item.Aliases...)
	best := 0
	for i, candidate := range candidates {
		if score, ok := fuzzySelectStringMatchScore(strings.ToLower(candidate), query); ok {
			if i == 0 {
				score += 40
			} else if i > 1 {
				score += 15
			}
			best = max(best, score)
		}
	}
	return best, best > 0
}

func fuzzySelectStringMatchScore(candidate, query string) (int, bool) {
	if candidate == query {
		return 1000, true
	}
	if strings.HasPrefix(candidate, query) {
		return 900 + len(query)*4, true
	}
	if index := strings.Index(candidate, query); index >= 0 {
		score := 800 + len(query)*4 - index
		if fuzzySelectWordBoundary(candidate, index) {
			score += 50
		}
		return score, true
	}
	return fuzzySelectFuzzyMatchScore(candidate, query)
}

func fuzzySelectFuzzyMatchScore(candidate, query string) (int, bool) {
	if query == "" {
		return 0, true
	}
	queryIndex := 0
	start := -1
	last := -1
	consecutive := 0
	wordBoundaryMatches := 0
	for i, ch := range candidate {
		if ch == rune(query[queryIndex]) {
			if start == -1 {
				start = i
			}
			if last == i-1 {
				consecutive++
			}
			if fuzzySelectWordBoundary(candidate, i) {
				wordBoundaryMatches++
			}
			last = i
			queryIndex++
			if queryIndex == len(query) {
				span := last - start + 1
				return 500 + len(query)*8 + consecutive*12 + wordBoundaryMatches*10 - start*2 - span, true
			}
		}
	}
	return 0, false
}

func fuzzySelectWordBoundary(s string, index int) bool {
	if index <= 0 || index >= len(s) {
		return index == 0
	}
	previous := s[index-1]
	current := s[index]
	return previous == ' ' || previous == '-' || previous == '_' || previous == ':' || previous == '/' || previous == '.' || previous == '+' || previous == '#' || previous == '@' || previous == '\t' || ('a' <= previous && previous <= 'z' && 'A' <= current && current <= 'Z')
}

func fuzzySelectPrimaryTextStyle(theme Theme, selected bool) Style {
	style := Style{Foreground: theme.Foreground}
	if selected {
		style.Attribute = AttrBold
	}
	return style
}

func fuzzySelectSecondaryTextStyle(theme Theme, selected bool) Style {
	if selected {
		return Style{Foreground: fuzzySelectMutedPrimaryText(theme)}
	}
	return Style{Foreground: theme.MutedForeground}
}

func fuzzySelectMutedPrimaryText(theme Theme) Color {
	if c, ok := blendColor(theme.Primary, theme.Foreground, 75); ok {
		return c
	}
	if theme.Foreground != 0 {
		return theme.Foreground
	}
	return theme.PrimaryText
}

func fuzzySelectWidth(width int) int {
	if width > 0 {
		return width
	}
	return defaultFuzzySelectWidth
}

func fuzzySelectMaxVisibleRows(rows int) int {
	if rows > 0 {
		return rows
	}
	return defaultFuzzySelectMaxVisibleRows
}

func fuzzySelectRowHeight(style FuzzySelectRowStyle) int {
	if style == FuzzySelectOneLine {
		return 1
	}
	return 2
}

func fuzzySelectList(controller *ScrollPaneController, width, height int, children []Widget) Widget {
	return SizedBox{Width: width, Height: height, Child: Scrollbar{Child: ScrollPane{
		Controller: controller,
		Child:      SizedBox{Width: max(1, width-1), Child: Flex{Axis: Vertical, CrossAxisAlignment: CrossAxisStretch, Children: children}},
	}}}
}

type fuzzySelectPositioner struct{ Child Widget }

func (w fuzzySelectPositioner) WidgetChild() Widget { return w.Child }

func (w fuzzySelectPositioner) CreateRenderObject(BuildContext) RenderObject {
	return &renderFuzzySelectPositioner{}
}

func (w fuzzySelectPositioner) UpdateRenderObject(BuildContext, RenderObject) {}

type renderFuzzySelectPositioner struct {
	SingleChildRenderObject
	offset Offset
}

func (r *renderFuzzySelectPositioner) Layout(ctx LayoutContext, c Constraints) {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	size = c.Constrain(size)
	if child := r.Child(); child != nil {
		child.Layout(ctx, Loose(size))
		childSize := child.Base().Size()
		r.offset = Offset{X: max(0, (size.Width-childSize.Width)/2), Y: fuzzySelectTopOffset(size.Height, childSize.Height)}
	}
	r.SetSize(size)
}

func (r *renderFuzzySelectPositioner) DryLayout(ctx LayoutContext, c Constraints) Size {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func fuzzySelectTopOffset(height, childHeight int) int {
	return min(max(0, height/fuzzySelectTopDivisor), max(0, height-childHeight))
}

func (r *renderFuzzySelectPositioner) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *renderFuzzySelectPositioner) ChildOffset(RenderObject) Offset { return r.offset }

func (r *renderFuzzySelectPositioner) HitTest(*HitTestResult, Point) bool { return false }
