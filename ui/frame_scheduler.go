package ui

import "time"

// DefaultFrameInterval is the default 60Hz frame pacing interval.
const DefaultFrameInterval = time.Second / 60

// FrameScheduler coalesces frame requests and applies a minimum frame interval.
type FrameScheduler struct {
	interval  time.Duration
	lastFrame time.Time
	scheduled bool
	due       time.Time
}

// NewFrameScheduler creates a scheduler using interval, or DefaultFrameInterval when interval is non-positive.
func NewFrameScheduler(interval time.Duration) *FrameScheduler {
	if interval <= 0 {
		interval = DefaultFrameInterval
	}
	return &FrameScheduler{interval: interval}
}

// Request schedules a frame and returns its due time.
func (s *FrameScheduler) Request(now time.Time) time.Time {
	if s.scheduled {
		return s.due
	}
	due := now
	if !s.lastFrame.IsZero() {
		next := s.lastFrame.Add(s.interval)
		if now.Before(next) {
			due = next
		}
	}
	s.scheduled = true
	s.due = due
	return due
}

// Scheduled reports whether a frame is pending.
func (s *FrameScheduler) Scheduled() bool {
	return s.scheduled
}

// Due returns the due time for the pending frame.
func (s *FrameScheduler) Due() time.Time {
	return s.due
}

// DidFrame records that a frame was rendered at now.
func (s *FrameScheduler) DidFrame(now time.Time) {
	s.lastFrame = now
	s.scheduled = false
	s.due = time.Time{}
}
