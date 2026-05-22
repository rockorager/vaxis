package ui

import (
	"sort"
	"strings"
)

const (
	defaultCommandPaletteWidth          = 54
	defaultCommandPaletteMaxVisibleRows = 5
	commandPaletteRowHeight             = 2
	commandPaletteTopDivisor            = 4

	commandPaletteSelectionIntentType IntentType = "ui.command-palette.selection"
)

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

func (w CommandPalette) CreateState() State {
	return &commandPaletteState{}
}

type commandPaletteState struct {
	StateBase
	query          string
	selected       int
	listController ScrollPaneController
}

func (s *commandPaletteState) Build(ctx BuildContext) Widget {
	w := s.Widget().(CommandPalette)
	items := commandPaletteFilteredItems(w, s.query)
	selected := s.selected
	if len(items) > 0 {
		selected = clampInt(selected, 0, len(items)-1)
	}
	return s.view(ctx, w, items, selected)
}

func (s *commandPaletteState) view(ctx BuildContext, w CommandPalette, items []CommandPaletteItem, selected int) Widget {
	theme := MustDepend[Theme](ctx)
	width := commandPaletteWidth(w)
	placeholder := w.Placeholder
	if placeholder == "" {
		placeholder = "Search commands…"
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
			empty = "No matching commands"
		}
		children = append(children, commandPaletteList(&s.listController, width, 1, []Widget{
			Text{Value: empty, Style: Style{Foreground: theme.MutedForeground}},
		}))
	} else {
		rows := make([]Widget, 0, len(items))
		for i, item := range items {
			index := i
			item := item
			isSelected := index == selected
			rows = append(rows, ListTile{
				Leading:  item.Leading,
				Trailing: item.Trailing,
				Title:    Text{Value: item.Title, Style: commandPalettePrimaryTextStyle(theme, isSelected), MaxLines: 1, Overflow: TextOverflowEllipsis},
				Subtitle: Text{Value: item.Description, Style: commandPaletteSecondaryTextStyle(theme, isSelected), MaxLines: 1, Overflow: TextOverflowEllipsis},
				Selected: isSelected,
				Disabled: item.Disabled,
				OnPressed: func(ctx EventContext) {
					s.run(ctx, []CommandPaletteItem{item}, 0)
				},
			})
		}
		listHeight := min(len(items)*commandPaletteRowHeight, commandPaletteMaxVisibleRows(w)*commandPaletteRowHeight)
		children = append(children, commandPaletteList(&s.listController, width, listHeight, rows))
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
			commandPaletteSelectionIntentType: func(ctx EventContext, intent Intent) EventResult {
				s.moveSelection(intent.(commandPaletteSelectionIntent).Delta)
				return EventHandled
			},
		},
		Child: Shortcuts{
			Bindings: ShortcutMap{
				"Down":   commandPaletteSelectionIntent{Delta: 1},
				"Ctrl+n": commandPaletteSelectionIntent{Delta: 1},
				"Up":     commandPaletteSelectionIntent{Delta: -1},
				"Ctrl+p": commandPaletteSelectionIntent{Delta: -1},
			},
			Child: commandPalettePositioner{Child: FocusScope{Trap: true, AutoFocus: true, Child: DecoratedBox(
				Decoration{Style: panelStyle},
				Padding(Symmetric(2, 1), ConstrainedBox{
					Constraints: Constraints{MinWidth: width, MaxWidth: width},
					Child:       Flex{Axis: Vertical, MainAxisSize: MainAxisSizeMin, CrossAxisAlignment: CrossAxisStretch, Children: children},
				}),
			)}},
		},
	}
}

func (s *commandPaletteState) setQuery(_ EventContext, query string) {
	s.SetState(func() {
		s.query = query
		s.selected = 0
	})
	s.listController.ScrollToStart()
}

func (s *commandPaletteState) moveSelection(delta int) {
	items := commandPaletteFilteredItems(s.Widget().(CommandPalette), s.query)
	if len(items) == 0 {
		return
	}
	next := clampInt(s.selected+delta, 0, len(items)-1)
	s.SetState(func() { s.selected = next })
	s.revealSelection(next)
}

func (s *commandPaletteState) revealSelection(index int) {
	metrics := s.listController.Metrics(ScrollVertical)
	if metrics.ViewportHeight == 0 {
		return
	}
	top := index * commandPaletteRowHeight
	bottom := top + commandPaletteRowHeight
	visibleTop := metrics.ScrollOffset
	visibleBottom := metrics.ScrollOffset + metrics.ViewportHeight
	switch {
	case top < visibleTop:
		s.listController.ScrollTo(0, top)
	case bottom > visibleBottom:
		s.listController.ScrollTo(0, bottom-metrics.ViewportHeight)
	}
}

func (s *commandPaletteState) run(ctx EventContext, items []CommandPaletteItem, index int) {
	if len(items) == 0 {
		return
	}
	w := s.Widget().(CommandPalette)
	item := items[clampInt(index, 0, len(items)-1)]
	if item.Disabled {
		return
	}
	if w.OnDismiss != nil {
		w.OnDismiss(ctx)
	}
	if item.OnSelected != nil {
		item.OnSelected(ctx)
	}
	if w.OnSelected != nil {
		w.OnSelected(ctx, item)
	}
}

type commandPaletteSelectionIntent struct{ Delta int }

func (i commandPaletteSelectionIntent) IntentType() IntentType {
	return commandPaletteSelectionIntentType
}

func commandPaletteFilteredItems(w CommandPalette, query string) []CommandPaletteItem {
	filter := w.Filter
	if filter == nil {
		filter = DefaultCommandPaletteFilter
	}
	return filter(query, w.Items)
}

// DefaultCommandPaletteFilter filters command items with title-weighted fuzzy matching.
func DefaultCommandPaletteFilter(query string, items []CommandPaletteItem) []CommandPaletteItem {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return items
	}
	type rankedItem struct {
		item  CommandPaletteItem
		score int
		index int
	}
	ranked := []rankedItem{}
	for i, item := range items {
		if score, ok := commandPaletteItemMatchScore(item, query); ok {
			ranked = append(ranked, rankedItem{item: item, score: score, index: i})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].index < ranked[j].index
	})
	out := make([]CommandPaletteItem, 0, len(ranked))
	for _, match := range ranked {
		out = append(out, match.item)
	}
	return out
}

func commandPaletteItemMatchScore(item CommandPaletteItem, query string) (int, bool) {
	candidates := append([]string{item.Title, item.Description}, item.Aliases...)
	best := 0
	for i, candidate := range candidates {
		if score, ok := commandPaletteStringMatchScore(strings.ToLower(candidate), query); ok {
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

func commandPaletteStringMatchScore(candidate, query string) (int, bool) {
	if candidate == query {
		return 1000, true
	}
	if strings.HasPrefix(candidate, query) {
		return 900 + len(query)*4, true
	}
	if index := strings.Index(candidate, query); index >= 0 {
		score := 800 + len(query)*4 - index
		if commandPaletteWordBoundary(candidate, index) {
			score += 50
		}
		return score, true
	}
	return commandPaletteFuzzyMatchScore(candidate, query)
}

func commandPaletteFuzzyMatchScore(candidate, query string) (int, bool) {
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
			if commandPaletteWordBoundary(candidate, i) {
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

func commandPaletteWordBoundary(s string, index int) bool {
	if index <= 0 || index >= len(s) {
		return index == 0
	}
	previous := s[index-1]
	current := s[index]
	return previous == ' ' || previous == '-' || previous == '_' || previous == ':' || previous == '/' || previous == '.' || previous == '+' || previous == '#' || previous == '@' || previous == '\t' || ('a' <= previous && previous <= 'z' && 'A' <= current && current <= 'Z')
}

func commandPalettePrimaryTextStyle(theme Theme, selected bool) Style {
	style := Style{Foreground: theme.Foreground}
	if selected {
		style.Attribute = AttrBold
	}
	return style
}

func commandPaletteSecondaryTextStyle(theme Theme, selected bool) Style {
	if selected {
		return Style{Foreground: commandPaletteMutedPrimaryText(theme)}
	}
	return Style{Foreground: theme.MutedForeground}
}

func commandPaletteMutedPrimaryText(theme Theme) Color {
	if c, ok := blendColor(theme.Primary, theme.Foreground, 75); ok {
		return c
	}
	if theme.Foreground != 0 {
		return theme.Foreground
	}
	return theme.PrimaryText
}

func commandPaletteWidth(w CommandPalette) int {
	if w.Width > 0 {
		return w.Width
	}
	return defaultCommandPaletteWidth
}

func commandPaletteMaxVisibleRows(w CommandPalette) int {
	if w.MaxVisibleRows > 0 {
		return w.MaxVisibleRows
	}
	return defaultCommandPaletteMaxVisibleRows
}

func commandPaletteList(controller *ScrollPaneController, width, height int, children []Widget) Widget {
	return SizedBox{Width: width, Height: height, Child: Scrollbar{Child: ScrollPane{
		Controller: controller,
		Child:      SizedBox{Width: max(1, width-1), Child: Flex{Axis: Vertical, CrossAxisAlignment: CrossAxisStretch, Children: children}},
	}}}
}

type commandPalettePositioner struct{ Child Widget }

func (w commandPalettePositioner) WidgetChild() Widget { return w.Child }

func (w commandPalettePositioner) CreateRenderObject(BuildContext) RenderObject {
	return &renderCommandPalettePositioner{}
}

func (w commandPalettePositioner) UpdateRenderObject(BuildContext, RenderObject) {}

type renderCommandPalettePositioner struct {
	SingleChildRenderObject
	offset Offset
}

func (r *renderCommandPalettePositioner) Layout(ctx LayoutContext, c Constraints) {
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
		r.offset = Offset{X: max(0, (size.Width-childSize.Width)/2), Y: commandPaletteTopOffset(size.Height, childSize.Height)}
	}
	r.SetSize(size)
}

func (r *renderCommandPalettePositioner) DryLayout(ctx LayoutContext, c Constraints) Size {
	size := Size{}
	if c.HasBoundedWidth() {
		size.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		size.Height = c.MaxHeight
	}
	return c.Constrain(size)
}

func commandPaletteTopOffset(height, childHeight int) int {
	return min(max(0, height/commandPaletteTopDivisor), max(0, height-childHeight))
}

func (r *renderCommandPalettePositioner) Paint(p *Painter, off Offset) {
	if child := r.Child(); child != nil {
		child.Paint(p, off.Add(r.offset))
	}
}

func (r *renderCommandPalettePositioner) ChildOffset(RenderObject) Offset { return r.offset }

func (r *renderCommandPalettePositioner) HitTest(*HitTestResult, Point) bool { return false }
