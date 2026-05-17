package ui

import "git.sr.ht/~rockorager/vaxis"

type Backend interface {
	Events() <-chan Event
	Size() Size
	Render(*Painter) error
	Close() error
}

type vaxisBackend struct{ vx *vaxis.Vaxis }

func (b vaxisBackend) Events() <-chan Event { return b.vx.Events() }

func (b vaxisBackend) Size() Size {
	win := b.vx.Window()
	return Size{Width: win.Width, Height: win.Height}
}

func (b vaxisBackend) Render(p *Painter) error {
	win := b.vx.Window()
	win.Clear()
	size := p.Size()
	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			win.SetCell(x, y, p.Cell(x, y))
		}
	}
	b.vx.Render()
	return nil
}

func (b vaxisBackend) Close() error {
	b.vx.Close()
	return nil
}
