package rtk

import "golang.org/x/exp/slog"

// Options provide setup options to the RTK instance
type Options struct {
	// Should the application use the full screen?
	Fullscreen bool

	// A slog.Handler to receive logs. RTK logs using the stdlib levels.
	LogHandler slog.Handler
}
