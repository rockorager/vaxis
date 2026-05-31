package ui

import (
	"context"
	"io"

	"go.rockorager.dev/vaxis"
)

// Backend is the runtime boundary between ui and a terminal implementation.
type Backend interface {
	Events() <-chan Event
	Size() Size
	Render(*Painter) error
	Dispatch(func())
	SetMouseShape(MouseShape)
	Close() error
}

// PrimaryScreenAppender is implemented by backends that support appending
// terminal output before a primary-screen live region.
type PrimaryScreenAppender interface {
	Append([]byte)
	AppendString(string)
	AppendWriter() io.Writer
}

// PrimaryScreenRegionSizer is implemented by backends that can resize a
// primary-screen live region independently from the terminal size.
type PrimaryScreenRegionSizer interface {
	SetPrimaryScreenRegionHeight(int)
}

type terminalSizer interface {
	TerminalSize() Resize
}

type vaxisBackend struct{ vx *vaxis.Vaxis }

func (b vaxisBackend) Events() <-chan Event {
	return b.vx.Events()
}

func (b vaxisBackend) Size() Size {
	win := b.vx.Window()
	return Size{Width: win.Width, Height: win.Height}
}

func (b vaxisBackend) TerminalSize() Resize {
	return b.vx.Size()
}

func (b vaxisBackend) Resize(size Resize) {
	b.vx.Resize(size)
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
	if cursor, ok := p.Cursor(); ok {
		b.vx.ShowCursor(cursor.Col, cursor.Row, cursor.Shape)
	} else {
		b.vx.HideCursor()
	}
	b.vx.Render()
	return nil
}

func (b vaxisBackend) Dispatch(fn func()) {
	b.vx.PostEvent(SyncFunc(fn))
}

func (b vaxisBackend) SetMouseShape(shape MouseShape) {
	b.vx.SetMouseShape(shape)
}

func (b vaxisBackend) Append(p []byte) {
	b.vx.Append(p)
}

func (b vaxisBackend) AppendString(s string) {
	b.vx.AppendString(s)
}

func (b vaxisBackend) AppendWriter() io.Writer {
	return b.vx.AppendWriter()
}

func (b vaxisBackend) SetPrimaryScreenRegionHeight(height int) {
	b.vx.SetPrimaryScreenRegionHeight(height)
}

func (b vaxisBackend) SetTitle(title string) {
	b.vx.SetTitle(title)
}

func (b vaxisBackend) CopyToClipboard(text string) {
	b.vx.ClipboardPush(text)
}

func (b vaxisBackend) Notify(title, body string) {
	b.vx.Notify(title, body)
}

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
