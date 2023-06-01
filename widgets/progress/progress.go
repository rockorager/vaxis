package progress

import (
	"io"
	"math"

	"git.sr.ht/~rockorager/rtk"
)

// Model represents a progress bar. A progress bar is also an io.Reader and an
// io.Writer. If you pass it a DataMsg with a Total before calling Read or
// Write, it will pass through the R/W and display the progress
type Model struct {
	Foreground rtk.Color
	Background rtk.Color
	Reader     io.Reader
	Writer     io.Writer

	progress float64
	total    float64
}

func New() *Model {
	return &Model{}
}

func (m *Model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case DataMsg:
		m.total = msg.Total
		m.progress = msg.Progress
	}
}

func (m *Model) Draw(srf rtk.Surface) {
	if m.total == 0 {
		return
	}
	_, w := srf.Size()
	fracBlocks := (m.progress / m.total) * float64(w)
	fullBlocks := math.Floor(fracBlocks)
	remainder := fracBlocks - fullBlocks

	for i := 0; i <= int(fullBlocks); i += 1 {
		srf.SetCell(i, 0, rtk.Cell{
			EGC:        "█",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	}
	switch {
	case remainder >= 0.875:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▉",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.75:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▊",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.625:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▋",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.5:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▌",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.375:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▍",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.25:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▎",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	case remainder >= 0.125:
		srf.SetCell(int(fullBlocks)+1, 0, rtk.Cell{
			EGC:        "▏",
			Foreground: m.Foreground,
			Background: m.Background,
		})
	}
}

// Read counts the bytes read from Reader and sends the Model an updated
// progress message
func (m *Model) Read(p []byte) (int, error) {
	n, err := m.Reader.Read(p)
	rtk.SendMsg(DataMsg{
		Progress: m.progress + float64(n),
		Total:    m.total,
	}, m)
	return n, err
}

// Write counts the bytes written to Writer and sends the Model an updated
// progress message
func (m *Model) Write(p []byte) (int, error) {
	n, err := m.Writer.Write(p)
	rtk.SendMsg(DataMsg{
		Progress: m.progress + float64(n),
		Total:    m.total,
	}, m)
	return n, err
}

type DataMsg struct {
	Progress float64
	Total    float64
}
