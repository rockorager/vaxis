package main

import (
	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/widgets/textinput"
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

func (m *textInput) Update(msg rtk.Msg) {
	m.input.Update(msg)
}

func (m *textInput) Draw(win rtk.Window) {
	m.input.Draw(win)
}
