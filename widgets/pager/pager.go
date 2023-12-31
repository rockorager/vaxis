package pager

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
)

const (
	WrapFast = iota
)

var defaultFill = vaxis.Character{Grapheme: " "}

type Model struct {
	Segments []vaxis.Segment
	lines    []*line
	Fill     vaxis.Cell
	Offset   int
	WrapMode int
	width    int
}

type line struct {
	characters []vaxis.Cell
}

func (l *line) append(t vaxis.Cell) {
	l.characters = append(l.characters, t)
}

func (m *Model) Draw(win vaxis.Window) {
	w, h := win.Size()
	if w != m.width {
		m.width = w
		m.Layout()
	}
	if len(m.lines)-m.Offset < h {
		m.Offset = len(m.lines) - h
	}
	if m.Offset < 0 {
		m.Offset = 0
	}
	if m.Fill.Grapheme == "" {
		m.Fill.Character = defaultFill
	}
	win.Fill(m.Fill)
	for row, l := range m.lines {
		if row < m.Offset {
			continue
		}
		if (row - m.Offset) >= h {
			return
		}
		col := 0
		for _, cell := range l.characters {
			win.SetCell(col, row-m.Offset, cell)
			col += cell.Width
		}
	}
}

func (m *Model) Layout() {
	m.lines = []*line{}
	l := &line{}
	col := 0
	for _, seg := range m.Segments {
		for _, char := range vaxis.Characters(seg.Text) {
			if strings.ContainsRune(char.Grapheme, '\n') {
				m.lines = append(m.lines, l)
				l = &line{}
				col = 0
				continue
			}
			cell := vaxis.Cell{
				Character: char,
				Style:     seg.Style,
			}
			l.append(cell)
			col += char.Width
			if col >= m.width {
				m.lines = append(m.lines, l)
				l = &line{}
				col = 0
			}
		}
	}
}

// Scrolls the pager down n lines, if it can
func (m *Model) ScrollDown() {
	m.Offset += 1
}

func (m *Model) ScrollUp() {
	m.Offset -= 1
}
