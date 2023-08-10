package textinput

import (
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
)

type Model struct {
	Prompt  vaxis.Text
	Content vaxis.Text
	// TODO handle the cursor better with the new wrapping
	cursor    int // the x position of the cursor
	cursorCol int
	cursorRow int
}

func (m *Model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		if m.cursor > len(m.Content.Content) {
			m.cursor = len(m.Content.Content)
		}
		switch msg.String() {
		case "Left":
			if m.cursor == 0 {
				return
			}
			// TODO
		case "Right":
			if m.cursor >= len(m.Content.Content) {
				return
			}
			// TODO
		case "BackSpace":
			switch {
			case m.cursor == 0:
				return
			case m.cursor == len(m.Content.Content):
				m.Content.Content = m.Content.Content[:m.cursor-1]
			default:
				m.Content.Content = m.Content.Content[:m.cursor-1] + m.Content.Content[m.cursor:]
			}
			m.cursor -= 1
		default:
			if msg.EventType == vaxis.EventRelease {
				return
			}
			if msg.Modifiers&vaxis.ModCtrl != 0 {
				return
			}
			if msg.Modifiers&vaxis.ModAlt != 0 {
				return
			}
			if msg.Modifiers&vaxis.ModSuper != 0 {
				return
			}
			if unicode.IsGraphic(msg.Codepoint) {
				switch {
				case m.cursor == len(m.Content.Content):
					m.Content.Content = m.Content.Content + string(msg.Codepoint)
				default:
					m.Content.Content = m.Content.Content[:m.cursor] +
						string(msg.Codepoint) +
						m.Content.Content[m.cursor:]
				}
				m.cursor += 1
			}
		}
	}
}

func (m *Model) Draw(win vaxis.Window) {
	var col int
	var row int
	switch m.Prompt.Content {
	case "":
		col, row = vaxis.Print(win, m.Content)
	default:
		col, row = vaxis.Print(win, m.Prompt, m.Content)
	}
	win.ShowCursor(col, row, 0)
	m.cursorCol = col
	m.cursorRow = row
}
