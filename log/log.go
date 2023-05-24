// Package log is a zero-dependency colorful, levelled logger.
package log

import (
	"fmt"
	"io"
	"log"
	"os"
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
	fatal = "FATAL"

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
	l:     log.New(os.Stderr, "", 0),
	color: true,
	fmt:   "15:04:05.000",
}

func init() {
	SetOutput(os.Stderr)
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

func Tracef(format string, args ...any) {
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

func Debugf(format string, args ...any) {
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

func Infof(format string, args ...any) {
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

func Warnf(format string, args ...any) {
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

func Errorf(format string, args ...any) {
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

func Error(e error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.level < LevelError {
		return
	}
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 1, err, now())
	default:
		prefix = fmt.Sprintf(basic, err, now())
	}
	l.l.Print(prefix + e.Error())
}

func Fatal(e error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 1, fatal, now())
	default:
		prefix = fmt.Sprintf(basic, fatal, now())
	}
	l.l.Print(prefix + e.Error())
}

// Fatalf logs the error and exits with code 1
func Fatalf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var prefix string
	switch l.color {
	case true:
		prefix = fmt.Sprintf(color, 1, fatal, now())
	default:
		prefix = fmt.Sprintf(basic, fatal, now())
	}
	l.l.Fatalf(prefix+format, args...)
}
