package spinner

import (
	"context"
	"time"

	"git.sr.ht/~rockorager/rtk"
)

// Model is a spinner. It has a duration and a set of frames. It will request
// partial-draws using the last provided Surface at the duration specified
type Model struct {
	Duration   time.Duration
	Foreground rtk.Color
	Background rtk.Color
	Attribute  rtk.AttributeMask
	Frames     []rune

	frame    int
	spinning bool
	cancel   context.CancelFunc
	srf      rtk.Surface
}

// New creates a new spinner
func New(dur time.Duration) *Model {
	return &Model{
		Frames:   []rune{'-', '\\', '|', '/'},
		Duration: dur,
	}
}

func (m *Model) Update(msg rtk.Msg) {
	switch msg.(type) {
	case startMsg:
		m.start()
	case toggleMsg:
		m.toggle()
	case stopMsg:
		m.stop()
	}
}

func (m *Model) Draw(srf rtk.Surface) {
	m.srf = srf
	if m.spinning {
		srf.SetCell(0, 0, rtk.Cell{
			EGC:        string(m.Frames[m.frame]),
			Foreground: m.Foreground,
			Background: m.Background,
			Attribute:  m.Attribute,
		})
	}
}

// Start the spinner. Start is thread safe and non-blocking
func (m *Model) Start() {
	rtk.SendMsg(startMsg{}, m)
}

// Start should only be called from the Update loop
func (m *Model) start() {
	if m.spinning {
		return
	}
	if len(m.Frames) == 0 {
		m.Frames = []rune{'-', '\\', '|', '/'}
	}
	var ctx context.Context

	ctx, m.cancel = context.WithCancel(context.Background())
	m.spinning = true
	ticker := time.NewTicker(m.Duration)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				m.frame = (m.frame + 1) % len(m.Frames)
				rtk.PartialDraw(m, m.srf)
			}
		}
	}()
}

// Stop the spinner. Stop is thread safe and non-blocking
func (m *Model) Stop() {
	rtk.SendMsg(stopMsg{}, m)
}

func (m *Model) stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.spinning = false
}

// Toggle the spinner. Stop is thread safe and non-blocking
func (m *Model) Toggle() {
	rtk.SendMsg(toggleMsg{}, m)
}

func (m *Model) toggle() {
	on := m.spinning
	if on {
		m.stop()
		return
	}
	m.start()
}

type startMsg struct{}

type stopMsg struct{}

type toggleMsg struct{}
