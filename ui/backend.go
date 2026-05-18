package ui

import (
	"context"

	"git.sr.ht/~rockorager/vaxis"
)

type Backend interface {
	Events() <-chan Event
	Size() Size
	Render(*Painter) error
	SetMouseShape(MouseShape)
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

func (b vaxisBackend) SetMouseShape(shape MouseShape) { b.vx.SetMouseShape(shape) }

func (b vaxisBackend) Close() error {
	b.vx.Close()
	return nil
}

func (b vaxisBackend) QueryForeground(ctx context.Context) Color {
	return b.vx.QueryForegroundContext(ctx)
}

func (b vaxisBackend) QueryBackground(ctx context.Context) Color {
	return b.vx.QueryBackgroundContext(ctx)
}

func (b vaxisBackend) QueryColor(ctx context.Context, index uint8) Color {
	return b.vx.QueryColorContext(ctx, vaxis.IndexColor(index))
}

type backendColorQuerier struct{ backend Backend }

func (q backendColorQuerier) QueryForeground(ctx context.Context) Color {
	b, ok := q.backend.(interface{ QueryForeground(context.Context) Color })
	if !ok {
		return 0
	}
	return b.QueryForeground(ctx)
}

func (q backendColorQuerier) QueryBackground(ctx context.Context) Color {
	b, ok := q.backend.(interface{ QueryBackground(context.Context) Color })
	if !ok {
		return 0
	}
	return b.QueryBackground(ctx)
}

func (q backendColorQuerier) QueryColor(ctx context.Context, index uint8) Color {
	b, ok := q.backend.(interface {
		QueryColor(context.Context, uint8) Color
	})
	if !ok {
		return 0
	}
	return b.QueryColor(ctx, index)
}
