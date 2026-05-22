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

func TestCommandPalettePaintsPanelBackground(t *testing.T) {
	theme := DefaultTheme()
	theme.Background = RGB(1, 2, 3)
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
	for _, pt := range []Point{{X: x - 3, Y: y - 1}, {X: x - 3, Y: y}, {X: x + 52, Y: y}} {
		if got := p.Cell(pt.X, pt.Y).Background; got != theme.SurfaceHovered {
			t.Fatalf("panel background at %#v = %#v, want %#v", pt, got, theme.SurfaceHovered)
		}
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
