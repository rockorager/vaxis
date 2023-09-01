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
	vx       *vaxis.Vaxis
}

// New creates a new spinner
func New(vx *vaxis.Vaxis, dur time.Duration) *Model {
	return &Model{
		Frames:   []rune{'-', '\\', '|', '/'},
		Duration: dur,
		vx:       vx,
	}
}

func (m *Model) Draw(w vaxis.Window) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spinning {
		w.SetCell(0, 0, vaxis.Text{
			Content:    string(m.Frames[m.frame]),
			Foreground: m.Foreground,
			Background: m.Background,
			Attribute:  m.Attribute,
		})
	}
}

// Start the spinner. Start is thread safe and non-blocking
func (m *Model) Start() {
	m.vx.SyncFunc(func() {
		m.start()
	})
}

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
				m.vx.PostEvent(vaxis.Redraw{})
				m.mu.Unlock()
			}
		}
	}()
}

// Stop the spinner. Stop is thread safe and non-blocking
func (m *Model) Stop() {
	m.vx.SyncFunc(func() {
		m.stop()
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
	m.vx.SyncFunc(func() {
		m.toggle()
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
