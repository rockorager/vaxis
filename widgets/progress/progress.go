package progress

import (
	"io"
	"math"

	"git.sr.ht/~rockorager/vaxis"
)

// Model represents a progress bar. A progress bar is also an io.Reader and an
// io.Writer. If you pass it a DataMsg with a Total before calling Read or
// Write, it will pass through the R/W and display the progress
type Model struct {
	Style  vaxis.Style
	Reader io.Reader
	Writer io.Writer

	Progress float64
	Total    float64
	vx       *vaxis.Vaxis
}

func New(vx *vaxis.Vaxis) *Model {
	return &Model{vx: vx}
}

var (
	full         = vaxis.Character{Grapheme: "█", Width: 1}
	sevenEighths = vaxis.Character{Grapheme: "▉", Width: 1}
	threeFourths = vaxis.Character{Grapheme: "▊", Width: 1}
	fiveEighths  = vaxis.Character{Grapheme: "▋", Width: 1}
	half         = vaxis.Character{Grapheme: "▌", Width: 1}
	threeEighths = vaxis.Character{Grapheme: "▍", Width: 1}
	oneFourth    = vaxis.Character{Grapheme: "▎", Width: 1}
	oneEighth    = vaxis.Character{Grapheme: "▏", Width: 1}
)

func (m *Model) Draw(win vaxis.Window) {
	if m.Total == 0 {
		return
	}
	_, w := win.Size()
	fracBlocks := (m.Progress / m.Total) * float64(w)
	fullBlocks := math.Floor(fracBlocks)
	remainder := fracBlocks - fullBlocks

	for i := 0; i <= int(fullBlocks); i += 1 {
		win.SetCell(i, 0, vaxis.Cell{
			Character: full,
			Style:     m.Style,
		})
	}
	switch {
	case remainder >= 0.875:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: sevenEighths,
			Style:     m.Style,
		})
	case remainder >= 0.75:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: threeFourths,
			Style:     m.Style,
		})
	case remainder >= 0.625:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: fiveEighths,
			Style:     m.Style,
		})
	case remainder >= 0.5:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: half,
			Style:     m.Style,
		})
	case remainder >= 0.375:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: threeEighths,
			Style:     m.Style,
		})
	case remainder >= 0.25:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: oneFourth,
			Style:     m.Style,
		})
	case remainder >= 0.125:
		win.SetCell(int(fullBlocks)+1, 0, vaxis.Cell{
			Character: oneEighth,
			Style:     m.Style,
		})
	}
}

// Read counts the bytes read from Reader and sends the Model an updated
// progress message. The Total field should be set to an expected value for this
// to work properly
func (m *Model) Read(p []byte) (int, error) {
	n, err := m.Reader.Read(p)
	fn := func() {
		m.Progress = m.Progress + float64(n)
	}
	m.vx.PostEvent(fn)
	return n, err
}

// Write counts the bytes written to Writer and sends the Model an updated
// progress message. The Total field should be set to an expected value for this
// to work properly
func (m *Model) Write(p []byte) (int, error) {
	n, err := m.Writer.Write(p)
	fn := func() {
		m.Progress = m.Progress + float64(n)
	}
	m.vx.PostEvent(fn)
	return n, err
}
