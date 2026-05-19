package ui

import "strings"

// DebugRenderedSnapshot describes the last painted terminal frame.
type DebugRenderedSnapshot struct {
	Size   DebugSize           `json:"size"`
	Cursor *DebugCursor        `json:"cursor,omitempty"`
	Cells  []DebugRenderedCell `json:"cells"`
}

// DebugCursor describes the cursor from a rendered debug snapshot.
type DebugCursor struct {
	Col   int         `json:"col"`
	Row   int         `json:"row"`
	Shape CursorStyle `json:"shape"`
}

// DebugRenderedCell describes one painted terminal cell.
type DebugRenderedCell struct {
	Col             int            `json:"col"`
	Row             int            `json:"row"`
	Grapheme        string         `json:"grapheme,omitempty"`
	Width           int            `json:"width,omitempty"`
	Foreground      Color          `json:"foreground,omitempty"`
	Background      Color          `json:"background,omitempty"`
	UnderlineColor  Color          `json:"underlineColor,omitempty"`
	UnderlineStyle  UnderlineStyle `json:"underlineStyle,omitempty"`
	Attribute       AttributeMask  `json:"attribute,omitempty"`
	Hyperlink       string         `json:"hyperlink,omitempty"`
	HyperlinkParams string         `json:"hyperlinkParams,omitempty"`
}

func debugRenderedSnapshot(p *Painter) DebugRenderedSnapshot {
	size := p.Size()
	snapshot := DebugRenderedSnapshot{
		Size:  debugSize(size),
		Cells: make([]DebugRenderedCell, 0, len(p.Cells())),
	}
	if cursor, ok := p.Cursor(); ok {
		snapshot.Cursor = &DebugCursor{Col: cursor.Col, Row: cursor.Row, Shape: cursor.Shape}
	}
	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			cell := p.Cell(x, y)
			snapshot.Cells = append(snapshot.Cells, DebugRenderedCell{
				Col:             x,
				Row:             y,
				Grapheme:        cell.Grapheme,
				Width:           cell.Width,
				Foreground:      cell.Foreground,
				Background:      cell.Background,
				UnderlineColor:  cell.UnderlineColor,
				UnderlineStyle:  cell.UnderlineStyle,
				Attribute:       cell.Attribute,
				Hyperlink:       cell.Hyperlink,
				HyperlinkParams: cell.HyperlinkParams,
			})
		}
	}
	return snapshot
}

func debugRenderedText(p *Painter) string {
	if p == nil {
		return ""
	}
	size := p.Size()
	var b strings.Builder
	for y := 0; y < size.Height; y++ {
		line := make([]string, size.Width)
		for x := 0; x < size.Width; x++ {
			cell := p.Cell(x, y)
			switch {
			case cell.Width == 0 && cell.Grapheme != "":
				// Continuation cell for a wide grapheme.
			case cell.Grapheme != "":
				line[x] = cell.Grapheme
			default:
				line[x] = " "
			}
		}
		end := len(line)
		for end > 0 && line[end-1] == " " {
			end--
		}
		for x := 0; x < end; x++ {
			b.WriteString(line[x])
		}
		if y < size.Height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
