package main

import (
	"os"
	"os/exec"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

type vt struct {
	term *term.Model
}

func newTerm() *vt {
	vt := &vt{
		term: term.New(),
	}
	vt.term.Logger = log
	vt.term.Start(exec.Command(os.Getenv("SHELL")))
	return vt
}

func (vt *vt) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case visible:
		vt.term.SetVisible(bool(msg))
	default:
		vt.term.Update(msg)
	}
}

func (vt *vt) Draw(win vaxis.Window) {
	vt.term.Draw(win)
}
