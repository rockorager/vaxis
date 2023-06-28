package textinput

import (
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
)

type Model struct {
	Prompt  vaxis.Segment
	Content vaxis.Segment
	cursor  int // the x position of the cursor
}

func (m *Model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "Left":
			if m.cursor == 0 {
				return
			}
			m.cursor -= 1
		case "Right":
			if m.cursor >= len(m.Content.Text) {
				return
			}
			m.cursor += 1
		case "BackSpace":
			switch {
			case m.cursor == 0:
				return
			case m.cursor == len(m.Content.Text):
				m.Content.Text = m.Content.Text[:m.cursor-1]
			default:
				m.Content.Text = m.Content.Text[:m.cursor-1] + m.Content.Text[m.cursor:]
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
				case m.cursor == len(m.Content.Text):
					m.Content.Text = m.Content.Text + string(msg.Codepoint)
				default:
					m.Content.Text = m.Content.Text[:m.cursor] +
						string(msg.Codepoint) +
						m.Content.Text[m.cursor:]
				}
				m.cursor += 1
			}
		}
	}
}

func (m *Model) Draw(win vaxis.Window) {
	switch m.Prompt.Text {
	case "":
		vaxis.PrintSegments(win, m.Content)
	default:
		vaxis.PrintSegments(win, m.Prompt, m.Content)
	}
	win.ShowCursor(m.cursor, 0, 0)
}
