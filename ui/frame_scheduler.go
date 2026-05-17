package ui

import "time"

const DefaultFrameInterval = time.Second / 60

type FrameScheduler struct {
	interval  time.Duration
	lastFrame time.Time
	scheduled bool
	due       time.Time
}

func NewFrameScheduler(interval time.Duration) *FrameScheduler {
	if interval <= 0 {
		interval = DefaultFrameInterval
	}
	return &FrameScheduler{interval: interval}
}

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

func (s *FrameScheduler) Scheduled() bool { return s.scheduled }
func (s *FrameScheduler) Due() time.Time  { return s.due }

func (s *FrameScheduler) DidFrame(now time.Time) {
	s.lastFrame = now
	s.scheduled = false
	s.due = time.Time{}
}
