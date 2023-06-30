package main

import (
	"context"

	"git.sr.ht/~rockorager/vaxis"
)

type model struct{}

func (m *model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "C-c":
			vaxis.Quit()
		}
	}
}

func (m *model) Draw(win vaxis.Window) {
	vaxis.Print(win, "Hello, World!")
}

func main() {
	err := vaxis.Init(context.Background(), vaxis.Options{})
	if err != nil {
		panic(err)
	}
	m := &model{}
	if err := vaxis.Run(m); err != nil {
		panic(err)
	}
}
