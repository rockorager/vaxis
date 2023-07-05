package spinner

import (
	"context"
	"sync"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

// Model is a spinner. It has a duration and a set of frames. It will request
// partial-draws using the last provided Surface at the duration specified
type Model struct {
	Duration   time.Duration
	Foreground vaxis.Color
	Background vaxis.Color
	Attribute  vaxis.AttributeMask
	Frames     []rune

	frame    int
	mu       sync.Mutex
	spinning bool
	cancel   context.CancelFunc
	win      vaxis.Window
}

// New creates a new spinner
func New(dur time.Duration) *Model {
	return &Model{
		Frames:   []rune{'-', '\\', '|', '/'},
		Duration: dur,
	}
}

func (m *Model) Update(msg vaxis.Msg) {
	switch msg.(type) {
	case startMsg:
		m.start()
	case toggleMsg:
		m.toggle()
	case stopMsg:
		m.stop()
	}
}

func (m *Model) Draw(w vaxis.Window) {
	m.win = w
	if m.spinning {
		m.mu.Lock()
		w.SetCell(0, 0, vaxis.Cell{
			Character:  string(m.Frames[m.frame]),
			Foreground: m.Foreground,
			Background: m.Background,
			Attribute:  m.Attribute,
		})
		m.mu.Unlock()
	}
}

// Start the spinner. Start is thread safe and non-blocking
func (m *Model) Start() {
	vaxis.PostMsg(vaxis.SendMsg{
		Msg:   startMsg{},
		Model: m,
	})
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
				m.mu.Lock()
				m.frame = (m.frame + 1) % len(m.Frames)
				m.mu.Unlock()
				vaxis.PostMsg(vaxis.DrawModelMsg{
					Model:  m,
					Window: m.win,
				})
			}
		}
	}()
}

// Stop the spinner. Stop is thread safe and non-blocking
func (m *Model) Stop() {
	vaxis.PostMsg(vaxis.SendMsg{
		Msg:   stopMsg{},
		Model: m,
	})
}

func (m *Model) stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.spinning = false
}

// Toggle the spinner. Stop is thread safe and non-blocking
func (m *Model) Toggle() {
	vaxis.PostMsg(vaxis.SendMsg{
		Msg:   toggleMsg{},
		Model: m,
	})
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
