package ui

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const profileWindow = 1000

type profileMetric int

const (
	profileKey profileMetric = iota
	profileMouse
	profileBuild
	profileLayout
	profilePaint
	profileRender
	profileFrame
	profileMetricCount
)

// DebugProfileSample contains timing stats for one profiled event or phase.
type DebugProfileSample struct {
	Count  int     `json:"count"`
	LastMS float64 `json:"last_ms"`
	P95MS  float64 `json:"p95_ms"`
	P99MS  float64 `json:"p99_ms"`
}

// DebugProfileSnapshot contains timing stats for recent UI events and frames.
type DebugProfileSnapshot struct {
	Window int                `json:"window"`
	Key    DebugProfileSample `json:"key"`
	Mouse  DebugProfileSample `json:"mouse"`
	Build  DebugProfileSample `json:"build"`
	Layout DebugProfileSample `json:"layout"`
	Paint  DebugProfileSample `json:"paint"`
	Render DebugProfileSample `json:"render"`
	Frame  DebugProfileSample `json:"frame"`
}

type profileStore struct {
	mu     sync.Mutex
	series [profileMetricCount]profileSeries
}

type profileSeries struct {
	samples [profileWindow]time.Duration
	next    int
	count   int
	last    time.Duration
}

type frameProfileTimings struct {
	build  time.Duration
	layout time.Duration
	paint  time.Duration
	render time.Duration
	frame  time.Duration
}

func (p *profileStore) record(metric profileMetric, d time.Duration) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.series[metric].record(d)
	p.mu.Unlock()
}

func (p *profileStore) recordFrame(t frameProfileTimings) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.series[profileBuild].record(t.build)
	p.series[profileLayout].record(t.layout)
	p.series[profilePaint].record(t.paint)
	p.series[profileRender].record(t.render)
	p.series[profileFrame].record(t.frame)
	p.mu.Unlock()
}

func (p *profileStore) snapshot() DebugProfileSnapshot {
	if p == nil {
		return DebugProfileSnapshot{Window: profileWindow}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return DebugProfileSnapshot{
		Window: profileWindow,
		Key:    p.series[profileKey].snapshot(),
		Mouse:  p.series[profileMouse].snapshot(),
		Build:  p.series[profileBuild].snapshot(),
		Layout: p.series[profileLayout].snapshot(),
		Paint:  p.series[profilePaint].snapshot(),
		Render: p.series[profileRender].snapshot(),
		Frame:  p.series[profileFrame].snapshot(),
	}
}

func (s *profileSeries) record(d time.Duration) {
	s.samples[s.next] = d
	s.next = (s.next + 1) % len(s.samples)
	if s.count < len(s.samples) {
		s.count++
	}
	s.last = d
}

func (s *profileSeries) snapshot() DebugProfileSample {
	if s.count == 0 {
		return DebugProfileSample{}
	}
	samples := make([]time.Duration, s.count)
	copy(samples, s.samples[:s.count])
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return DebugProfileSample{
		Count:  s.count,
		LastMS: durationMS(s.last),
		P95MS:  durationMS(percentileDuration(samples, 95)),
		P99MS:  durationMS(percentileDuration(samples, 99)),
	}
}

func percentileDuration(sorted []time.Duration, percentile int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (percentile*len(sorted) + 99) / 100
	if idx < 1 {
		idx = 1
	}
	if idx > len(sorted) {
		idx = len(sorted)
	}
	return sorted[idx-1]
}

func durationMS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func drawProfileOverlay(p *Painter, snapshot DebugProfileSnapshot) {
	size := p.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}
	lines := profileOverlayLines(snapshot)
	contentWidth := 0
	for _, line := range lines {
		contentWidth = max(contentWidth, len(line))
	}
	if contentWidth == 0 {
		return
	}
	width := contentWidth + 4
	height := len(lines) + 2
	if width > size.Width {
		width = size.Width
	}
	if height > size.Height {
		height = size.Height
	}
	x := max(0, size.Width-width)
	style := Style{Foreground: RGB(235, 235, 235), Background: RGB(28, 32, 36)}
	headerStyle := style
	headerStyle.Attribute = AttrBold
	p.Fill(Rect{X: x, Y: 0, Width: width, Height: height}, Cell{Character: Character{Grapheme: " ", Width: 1}, Style: style})
	drawProfileOverlayBorder(p, x, 0, width, height, style)
	for y, line := range lines {
		row := y + 1
		if row >= height-1 {
			break
		}
		maxTextWidth := max(0, width-4)
		if len(line) > maxTextWidth {
			line = line[:maxTextWidth]
		}
		lineStyle := style
		if y == 0 || y == 1 {
			lineStyle = headerStyle
		}
		p.DrawText(Offset{X: x + 2, Y: row}, line, lineStyle)
	}
}

func drawProfileOverlayBorder(p *Painter, x, y, width, height int, style Style) {
	if width < 2 || height < 2 {
		return
	}
	horizontal := Cell{Character: Character{Grapheme: "─", Width: 1}, Style: style}
	vertical := Cell{Character: Character{Grapheme: "│", Width: 1}, Style: style}
	for col := x + 1; col < x+width-1; col++ {
		p.DrawCell(Point{X: col, Y: y}, horizontal)
		p.DrawCell(Point{X: col, Y: y + height - 1}, horizontal)
	}
	for row := y + 1; row < y+height-1; row++ {
		p.DrawCell(Point{X: x, Y: row}, vertical)
		p.DrawCell(Point{X: x + width - 1, Y: row}, vertical)
	}
	p.DrawCell(Point{X: x, Y: y}, Cell{Character: Character{Grapheme: "╭", Width: 1}, Style: style})
	p.DrawCell(Point{X: x + width - 1, Y: y}, Cell{Character: Character{Grapheme: "╮", Width: 1}, Style: style})
	p.DrawCell(Point{X: x, Y: y + height - 1}, Cell{Character: Character{Grapheme: "╰", Width: 1}, Style: style})
	p.DrawCell(Point{X: x + width - 1, Y: y + height - 1}, Cell{Character: Character{Grapheme: "╯", Width: 1}, Style: style})
}

func profileOverlayLines(snapshot DebugProfileSnapshot) []string {
	rows := []struct {
		name   string
		sample DebugProfileSample
	}{
		{"key", snapshot.Key},
		{"mouse", snapshot.Mouse},
		{"build", snapshot.Build},
		{"layout", snapshot.Layout},
		{"paint", snapshot.Paint},
		{"render", snapshot.Render},
		{"frame", snapshot.Frame},
	}
	lines := []string{
		"metric   last    p95    p99",
		"------ ------ ------ ------",
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%-6s %6.2f %6.2f %6.2f", row.name, row.sample.LastMS, row.sample.P95MS, row.sample.P99MS))
	}
	return lines
}
