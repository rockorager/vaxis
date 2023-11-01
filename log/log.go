package log

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

const (
	LevelError = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace

	trace = "TRACE"
	debug = "DEBUG"
	info  = "INFO"
	warn  = "WARN"
	err   = "ERROR"

	// formats
	// prefix + time
	basic = "%-5s %s "
	color = "\x1b[3%dm%-5s\x1b[m \x1b[2m%s\x1b[m "
)

type logger struct {
	l  *log.Logger
	mu sync.Mutex

	level int
	fmt   string
	color bool
}

var l = &logger{
	l:     log.New(io.Discard, "", 0),
	color: true,
	fmt:   "15:04:05.000",
}

// LevelError = 0
// LevelWarn = 1
// LevelInfo  = 2
// LevelDebug  = 3
// LevelTrace = 4
func SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.SetOutput(w)
}

func SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fmt = format
}

func DisableColor() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color = false
}

// Default
func EnableColor() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.color = true
}

func now() string {
	return time.Now().Format(l.fmt)
}

func Trace(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelTrace {
		return
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 7, trace, now())
	default:
		prefix = fmt.Sprintf(basic, trace, now())
	}
	l.l.Printf(prefix+format, args...)
}

func Debug(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelDebug {
		return
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 5, debug, now())
	default:
		prefix = fmt.Sprintf(basic, debug, now())
	}
	l.l.Printf(prefix+format, args...)
}

func Info(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelInfo {
		return
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 4, info, now())
	default:
		prefix = fmt.Sprintf(basic, info, now())
	}
	l.l.Printf(prefix+format, args...)
}

func Warn(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelWarn {
		return
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 3, warn, now())
	default:
		prefix = fmt.Sprintf(basic, warn, now())
	}
	l.l.Printf(prefix+format, args...)
}

func Error(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelError {
		return
	}
	if l.color {
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 1, err, now())
	default:
		prefix = fmt.Sprintf(basic, err, now())
	}
	l.l.Printf(prefix+format, args...)
}
