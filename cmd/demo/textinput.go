package main

import (
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

type textInput struct {
	input *textinput.Model
}

func newTextInput() *textInput {
	input := &textInput{
		input: &textinput.Model{},
	}
	return input
}

func (m *textInput) Update(msg vaxis.Msg) {
	m.input.Update(msg)
}

func (m *textInput) Draw(win vaxis.Window) {
	m.input.Draw(win)
}
