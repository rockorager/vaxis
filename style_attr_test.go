package vaxis

import "testing"

func TestParseStyledStringOverline(t *testing.T) {
	cells := ParseStyledString("\x1b[53mx\x1b[55my")
	if len(cells) != 2 {
		t.Fatalf("cells len = %d, want 2", len(cells))
	}
	if cells[0].Attribute&AttrOverline == 0 {
		t.Fatal("overline attribute was not set")
	}
	if cells[1].Attribute&AttrOverline != 0 {
		t.Fatal("overline attribute was not reset")
	}
}

func TestNewStyledStringOverline(t *testing.T) {
	ss := (&Vaxis{}).NewStyledString("\x1b[53mx\x1b[55my", Style{})
	if len(ss.Cells) != 2 {
		t.Fatalf("cells len = %d, want 2", len(ss.Cells))
	}
	if ss.Cells[0].Attribute&AttrOverline == 0 {
		t.Fatal("overline attribute was not set")
	}
	if ss.Cells[1].Attribute&AttrOverline != 0 {
		t.Fatal("overline attribute was not reset")
	}
}

func TestParseStyledStringGhosttySGRDetails(t *testing.T) {
	cells := ParseStyledString("\x1b[21;6;58:5:4mx\x1b[59;4:9my")
	if len(cells) != 2 {
		t.Fatalf("cells len = %d, want 2", len(cells))
	}
	if got, want := cells[0].UnderlineStyle, UnderlineDouble; got != want {
		t.Fatalf("first underline style = %d, want %d", got, want)
	}
	if cells[0].Attribute&AttrBlink == 0 {
		t.Fatal("rapid blink attribute was not set")
	}
	if got, want := cells[0].UnderlineColor, IndexColor(4); got != want {
		t.Fatalf("first underline color = %v, want %v", got, want)
	}
	if got, want := cells[1].UnderlineStyle, UnderlineSingle; got != want {
		t.Fatalf("second underline style = %d, want %d", got, want)
	}
	if cells[1].UnderlineColor != 0 {
		t.Fatalf("second underline color = %v, want default", cells[1].UnderlineColor)
	}
}

func TestParseStyledStringMalformedSGRContinues(t *testing.T) {
	cells := ParseStyledString("\x1b[38;5mx\x1b[0:4;3my")
	if len(cells) != 2 {
		t.Fatalf("cells len = %d, want 2", len(cells))
	}
	if cells[0].Foreground != 0 {
		t.Fatalf("malformed color set foreground to %v", cells[0].Foreground)
	}
	if cells[0].Attribute&AttrBlink == 0 {
		t.Fatal("later malformed-color parameter was not applied as blink")
	}
	if cells[1].Attribute&AttrBlink == 0 {
		t.Fatal("unsupported colon group reset existing blink attribute")
	}
	if cells[1].Attribute&AttrItalic == 0 {
		t.Fatal("later SGR parameter after unsupported colon group was not applied")
	}
}

func TestNewStyledStringGhosttySGRDetails(t *testing.T) {
	ss := (&Vaxis{}).NewStyledString("\x1b[21;6;58:5:4mx\x1b[59;4:9my", Style{})
	if len(ss.Cells) != 2 {
		t.Fatalf("cells len = %d, want 2", len(ss.Cells))
	}
	if got, want := ss.Cells[0].UnderlineStyle, UnderlineDouble; got != want {
		t.Fatalf("first underline style = %d, want %d", got, want)
	}
	if ss.Cells[0].Attribute&AttrBlink == 0 {
		t.Fatal("rapid blink attribute was not set")
	}
	if got, want := ss.Cells[0].UnderlineColor, IndexColor(4); got != want {
		t.Fatalf("first underline color = %v, want %v", got, want)
	}
	if got, want := ss.Cells[1].UnderlineStyle, UnderlineSingle; got != want {
		t.Fatalf("second underline style = %d, want %d", got, want)
	}
	if ss.Cells[1].UnderlineColor != 0 {
		t.Fatalf("second underline color = %v, want default", ss.Cells[1].UnderlineColor)
	}
}

func TestEncodeCellsOverline(t *testing.T) {
	got := EncodeCells([]Cell{
		{
			Character: Character{Grapheme: "x", Width: 1},
			Style:     Style{Attribute: AttrOverline},
		},
		{
			Character: Character{Grapheme: "y", Width: 1},
		},
	})
	want := "\x1b[53mx\x1b[55my"
	if got != want {
		t.Fatalf("encoded cells = %q, want %q", got, want)
	}
}

func TestStyledStringEncodeOverline(t *testing.T) {
	ss := StyledString{Cells: []Cell{
		{
			Character: Character{Grapheme: "x", Width: 1},
			Style:     Style{Attribute: AttrOverline},
		},
		{
			Character: Character{Grapheme: "y", Width: 1},
		},
	}}
	want := "\x1b[53mx\x1b[55my"
	if got := ss.Encode(); got != want {
		t.Fatalf("encoded styled string = %q, want %q", got, want)
	}
}
