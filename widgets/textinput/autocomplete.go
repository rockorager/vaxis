package textinput

import (
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
)

type AutoComplete struct {
	vx          *vaxis.Vaxis
	input       *Model
	complete    func(string) []string
	original    string
	completions []string
	option      int
}

func NewAutoComplete(vx *vaxis.Vaxis, complete func(string) []string) *AutoComplete {
	m := &AutoComplete{
		vx:       vx,
		input:    New(),
		complete: complete,
	}
	return m
}

func (a *AutoComplete) Update(msg vaxis.Event) {
	switch msg := msg.(type) {
	case vaxis.Key:
		if msg.EventType == vaxis.EventRelease {
			return
		}
		switch msg.String() {
		case "Enter":
			a.reset()
		case "Escape":
			a.input.SetContent(a.original)
			a.reset()
		case "Backspace":
			a.reset()
		default:
			if len(a.completions) > 0 && unicode.IsGraphic(msg.Keycode) {
				a.reset()
			}
		}
	}
	a.input.Update(msg)
}

func (a *AutoComplete) reset() {
	a.option = 0
	a.completions = []string{}
	a.original = ""
}

func (a *AutoComplete) Draw(win vaxis.Window) {
	// col, row := win.Origin()
	a.input.Draw(win)
}
