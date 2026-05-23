package ui

import (
	"fmt"
	"strings"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
)

func TestDefaultCommandPaletteFilterRanksMatches(t *testing.T) {
	items := []CommandPaletteItem{
		{Title: "Switch account", Description: "Work", Aliases: []string{"accounts"}},
		{Title: "Next message", Description: "Move down", Aliases: []string{"j"}},
		{Title: "Search mail", Description: "Find messages"},
	}
	matches := DefaultCommandPaletteFilter("sw", items)
	if len(matches) == 0 || matches[0].Title != "Switch account" {
		t.Fatalf("top match for sw = %v, want Switch account", commandPaletteTestTitles(matches))
	}
	matches = DefaultCommandPaletteFilter("nme", items)
	if len(matches) == 0 || matches[0].Title != "Next message" {
		t.Fatalf("top fuzzy match for nme = %v, want Next message", commandPaletteTestTitles(matches))
	}
}

func TestCommandPaletteUsesCustomFilter(t *testing.T) {
	app := NewApp(CommandPalette{
		Items: []CommandPaletteItem{{Title: "Hidden"}, {Title: "Shown"}},
		Filter: func(query string, items []CommandPaletteItem) []CommandPaletteItem {
			return []CommandPaletteItem{items[1]}
		},
	})
	app.Pump(Size{Width: 80, Height: 24})
	p := NewPainter(Size{Width: 80, Height: 24})
	app.Paint(p)
	if _, _, ok := commandPaletteFindText(p, "Shown"); !ok {
		t.Fatal("custom-filtered command was not rendered")
	}
	if _, _, ok := commandPaletteFindText(p, "Hidden"); ok {
		t.Fatal("custom filter rendered hidden command")
	}
}

func TestFuzzySelectFiltersAndActivatesGenericItems(t *testing.T) {
	type file struct {
		Name string
		Path string
	}
	var selected file
	app := NewApp(FuzzySelect[file]{
		Items: []file{
			{Name: "main.go", Path: "cmd/app/main.go"},
			{Name: "theme.go", Path: "ui/theme.go"},
		},
		Item: func(f file) FuzzySelectItem {
			return FuzzySelectItem{Title: f.Name, Description: f.Path, Aliases: []string{f.Path}}
		},
		OnSelected: func(_ EventContext, f file) { selected = f },
	})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Text: "t", Keycode: 't'})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Text: "h", Keycode: 'h'})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Keycode: vaxis.KeyEnter})

	if selected.Name != "theme.go" {
		t.Fatalf("selected = %#v, want theme.go", selected)
	}
}

func TestFuzzySelectOneLineRows(t *testing.T) {
	app := NewApp(FuzzySelect[string]{
		Items:          []string{"alpha", "beta", "gamma"},
		RowStyle:       FuzzySelectOneLine,
		MaxVisibleRows: 2,
	})
	app.Pump(Size{Width: 80, Height: 24})
	p := NewPainter(Size{Width: 80, Height: 24})
	app.Paint(p)
	_, alphaY, ok := commandPaletteFindText(p, "alpha")
	if !ok {
		t.Fatalf("one-line fuzzy select did not render first row")
	}
	_, betaY, ok := commandPaletteFindText(p, "beta")
	if !ok {
		t.Fatalf("one-line fuzzy select did not render second row")
	}
	if betaY-alphaY != 1 {
		t.Fatalf("one-line row spacing = %d, want 1", betaY-alphaY)
	}
	if _, _, ok := commandPaletteFindText(p, "gamma"); ok {
		t.Fatalf("third row rendered despite MaxVisibleRows 2")
	}
}

func TestCommandPalettePaintsPanelBackground(t *testing.T) {
	theme := DefaultTheme()
	theme.Background = RGB(1, 2, 3)
	theme.SurfaceRaised = RGB(30, 40, 50)
	theme.SurfaceHovered = RGB(40, 50, 60)
	theme.Primary = RGB(70, 80, 90)
	theme.MutedForeground = RGB(100, 110, 120)
	root := DecoratedBox(
		Decoration{Style: Style{Background: theme.Background}},
		CommandPalette{Items: []CommandPaletteItem{
			{Title: "Next mailbox", Description: "Move to the mailbox on the right"},
			{Title: "Previous mailbox", Description: "Move to the mailbox on the left"},
		}},
	)
	app := NewApp(root, WithTheme(theme))
	app.Pump(Size{Width: 80, Height: 30})
	p := NewPainter(Size{Width: 80, Height: 30})
	app.Paint(p)
	x, y, ok := commandPaletteFindText(p, "Search commands")
	if !ok {
		t.Fatalf("rendered command palette search placeholder not found")
	}
	for _, pt := range []Point{{X: x - 3, Y: y - 1}, {X: x - 3, Y: y}} {
		if got := p.Cell(pt.X, pt.Y).Background; got != theme.SurfaceRaised {
			t.Fatalf("panel background at %#v = %#v, want %#v", pt, got, theme.SurfaceRaised)
		}
	}
}

func TestCommandPaletteTopEdgeIsOneQuarterDown(t *testing.T) {
	theme := DefaultTheme()
	theme.Background = RGB(1, 2, 3)
	theme.SurfaceRaised = RGB(30, 40, 50)
	theme.SurfaceHovered = RGB(40, 50, 60)
	root := DecoratedBox(
		Decoration{Style: Style{Background: theme.Background}},
		CommandPalette{Items: commandPaletteTestItems(2)},
	)
	app := NewApp(root, WithTheme(theme))
	size := Size{Width: 80, Height: 30}
	app.Pump(size)
	p := NewPainter(size)
	app.Paint(p)
	x, y, ok := commandPaletteFindText(p, "Search commands")
	if !ok {
		t.Fatalf("rendered command palette search placeholder not found")
	}
	topEdge := y - 1
	want := size.Height / fuzzySelectTopDivisor
	if topEdge != want {
		t.Fatalf("command palette top edge = %d, want %d", topEdge, want)
	}
	if got := p.Cell(x-3, topEdge).Background; got != theme.SurfaceRaised {
		t.Fatalf("command palette top edge background = %#v, want %#v", got, theme.SurfaceRaised)
	}
}

func TestCommandPaletteScrollsOverflowingCommandsToSelection(t *testing.T) {
	app := NewApp(CommandPalette{Items: commandPaletteTestItems(8)})
	app.Pump(Size{Width: 80, Height: 30})
	for i := 0; i < 5; i++ {
		app.Send(Key{Keycode: KeyDown})
		app.Pump(Size{Width: 80, Height: 30})
	}
	p := NewPainter(Size{Width: 80, Height: 30})
	app.Paint(p)
	if _, _, ok := commandPaletteFindText(p, "Command 06"); !ok {
		t.Fatal("selected command was not scrolled into view")
	}
	if _, _, ok := commandPaletteFindText(p, "Command 01"); ok {
		t.Fatal("first command remained visible after scrolling to the sixth command")
	}
}

func TestCommandPaletteCtrlNAndCtrlPMoveSelection(t *testing.T) {
	selected := ""
	app := NewApp(CommandPalette{
		Items: []CommandPaletteItem{
			{Title: "First", OnSelected: func(EventContext) { selected = "First" }},
			{Title: "Second", OnSelected: func(EventContext) { selected = "Second" }},
			{Title: "Third", OnSelected: func(EventContext) { selected = "Third" }},
		},
	})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Text: "n", Keycode: 'n', Modifiers: vaxis.ModCtrl})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Text: "n", Keycode: 'n', Modifiers: vaxis.ModCtrl})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Text: "p", Keycode: 'p', Modifiers: vaxis.ModCtrl})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Keycode: vaxis.KeyEnter})

	if selected != "Second" {
		t.Fatalf("selected = %q, want Second", selected)
	}
}

func TestCommandPaletteActivatesSelectedItem(t *testing.T) {
	selected := ""
	dismissed := false
	app := NewApp(CommandPalette{
		Items: []CommandPaletteItem{
			{Title: "First"},
			{Title: "Second", OnSelected: func(EventContext) { selected = "item" }},
		},
		OnDismiss: func(EventContext) { dismissed = true },
		OnSelected: func(_ EventContext, item CommandPaletteItem) {
			selected += ":" + item.Title
		},
	})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Keycode: KeyDown})
	app.Pump(Size{Width: 80, Height: 24})
	app.Send(Key{Keycode: vaxis.KeyEnter})
	app.Pump(Size{Width: 80, Height: 24})
	if !dismissed {
		t.Fatal("command palette did not dismiss before activation")
	}
	if selected != "item:Second" {
		t.Fatalf("selected = %q, want item:Second", selected)
	}
}

func TestCommandPaletteSelectedTextStyles(t *testing.T) {
	theme := DefaultTheme()
	if got := fuzzySelectPrimaryTextStyle(theme, false).Attribute; got&AttrBold != 0 {
		t.Fatalf("unselected primary text was bold: %#v", got)
	}
	if got := fuzzySelectPrimaryTextStyle(theme, true).Attribute; got&AttrBold == 0 {
		t.Fatalf("selected primary text was not bold: %#v", got)
	}
	if got := fuzzySelectSecondaryTextStyle(theme, false).Foreground; got != theme.MutedForeground {
		t.Fatalf("unselected secondary foreground = %#v, want muted foreground %#v", got, theme.MutedForeground)
	}
	want, ok := blendColor(theme.Primary, theme.Foreground, 75)
	if !ok {
		t.Fatalf("default theme primary/foreground colors were not blendable")
	}
	if got := fuzzySelectSecondaryTextStyle(theme, true).Foreground; got != want {
		t.Fatalf("selected secondary foreground = %#v, want primary/foreground blend %#v", got, want)
	}
}

func commandPaletteTestItems(n int) []CommandPaletteItem {
	items := make([]CommandPaletteItem, n)
	for i := range items {
		items[i] = CommandPaletteItem{Title: fmt.Sprintf("Command %02d", i+1), Description: "test command"}
	}
	return items
}

func commandPaletteTestTitles(items []CommandPaletteItem) []string {
	titles := make([]string, len(items))
	for i, item := range items {
		titles[i] = item.Title
	}
	return titles
}

func commandPaletteFindText(p *Painter, text string) (int, int, bool) {
	for y := 0; y < p.Size().Height; y++ {
		var line strings.Builder
		for x := 0; x < p.Size().Width; x++ {
			line.WriteString(p.Cell(x, y).Grapheme)
		}
		if x := strings.Index(line.String(), text); x >= 0 {
			return x, y, true
		}
	}
	return 0, 0, false
}
