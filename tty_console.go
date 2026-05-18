package vaxis

import "github.com/containerd/console"

type consoleTTY struct {
	console.Console
}

func (t consoleTTY) Size() (Resize, error) {
	ws, err := t.Console.Size()
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols: int(ws.Width),
		Rows: int(ws.Height),
	}, nil
}

func (t consoleTTY) StartInput(*Vaxis) error {
	return nil
}

func (t consoleTTY) StopInput() error {
	return nil
}
