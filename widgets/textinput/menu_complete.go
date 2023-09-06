package textinput

import (
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
)

type MenuComplete struct {
	input       *Model
	complete    func(string) []string
	original    string
	completions []string
	option      int
}

func NewMenuComplete(complete func(string) []string) *MenuComplete {
	m := &MenuComplete{
		input:    New(),
		complete: complete,
	}
	return m
}

func (m *MenuComplete) Update(msg vaxis.Event) {
	switch msg := msg.(type) {
	case vaxis.Key:
		if msg.EventType == vaxis.EventRelease {
			return
		}
		switch msg.String() {
		case "Tab":
			// Trigger completion
			switch len(m.completions) {
			case 0:
				m.option = 0
				m.original = m.input.String()
				m.completions = m.complete(m.input.String())
				m.completions = append(m.completions, m.original)
			default:
				m.option += 1
				if m.option >= len(m.completions) {
					m.option = 0
				}
			}

			if len(m.completions) > 0 {
				m.input.SetContent(m.completions[m.option])
			}
			return
		case "Shift+Tab":
			if len(m.completions) > 0 {
				m.option -= 1
				if m.option < 0 {
					m.option = len(m.completions) - 1
				}
				m.input.SetContent(m.completions[m.option])
				return
			}
		case "Enter":
			m.reset()
		case "Escape":
			m.input.SetContent(m.original)
			m.reset()
		case "Backspace":
			m.reset()
		default:
			if len(m.completions) > 0 && unicode.IsGraphic(msg.Codepoint) {
				m.reset()
			}
		}
	}
	m.input.Update(msg)
}

func (m *MenuComplete) reset() {
	m.option = 0
	m.completions = []string{}
	m.original = ""
}

func (m *MenuComplete) Draw(win vaxis.Window) {
	m.input.Draw(win)
}
