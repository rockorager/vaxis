package ui

import "git.sr.ht/~rockorager/vaxis"

func Run(root Widget, opts ...Option) error {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		return err
	}
	defer vx.Close()

	app := NewApp(root, opts...)
	for ev := range vx.Events() {
		app.Send(ev)
		if app.quit {
			return nil
		}
		win := vx.Window()
		size := Size{Width: win.Width, Height: win.Height}
		app.Pump(size)
		painter := NewPainter(size)
		app.Paint(painter)
		win.Clear()
		for y := 0; y < size.Height; y++ {
			for x := 0; x < size.Width; x++ {
				win.SetCell(x, y, painter.Cell(x, y))
			}
		}
		vx.Render()
	}
	return nil
}
