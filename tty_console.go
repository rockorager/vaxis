package vaxis

type consoleTTY struct {
	Console
}

func (t consoleTTY) Size() (Resize, error) {
	cols, rows, xPixels, yPixels, err := t.Console.Size()
	if err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols:   cols,
		Rows:   rows,
		XPixel: xPixels,
		YPixel: yPixels,
	}, nil
}

func (t consoleTTY) StartInput(*Vaxis) error {
	return nil
}

func (t consoleTTY) StopInput() error {
	return nil
}
