package log

import (
	"fmt"
	"io"
	"log"
	"time"
)

const (
	LevelError int = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace

	calldepth = 3
	flags     = log.Lshortfile
)

var (
	level       = LevelError
	traceLogger = log.New(io.Discard, "TRACE ", flags)
	debugLogger = log.New(io.Discard, "DEBUG ", flags)
	infoLogger  = log.New(io.Discard, "INFO  ", flags)
	warnLogger  = log.New(io.Discard, "WARN  ", flags)
	errorLogger = log.New(io.Discard, "ERROR ", flags)
)

// LevelError = 0
// LevelWarn = 1
// LevelInfo  = 2
// LevelDebug  = 3
// LevelTrace = 4
func SetLevel(l int) {
	level = l
}

func SetOutput(w io.Writer) {
	traceLogger.SetOutput(w)
	debugLogger.SetOutput(w)
	infoLogger.SetOutput(w)
	warnLogger.SetOutput(w)
	errorLogger.SetOutput(w)
}

func now() string {
	return time.Now().Format("15:04:05.000")
}

func fmtMessage(message string, args ...any) string {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}
	return fmt.Sprintf("%s %s", now(), message)
}

func Trace(format string, args ...any) {
	if level < LevelTrace {
		return
	}
	message := fmtMessage(format, args...)
	traceLogger.Output(calldepth, message)
}

func Debug(format string, args ...any) {
	if level < LevelDebug {
		return
	}
	message := fmtMessage(format, args...)
	debugLogger.Output(calldepth, message)
}

func Info(format string, args ...any) {
	if level < LevelInfo {
		return
	}
	message := fmtMessage(format, args...)
	infoLogger.Output(calldepth, message)
}

func Warn(format string, args ...any) {
	if level < LevelWarn {
		return
	}
	message := fmtMessage(format, args...)
	warnLogger.Output(calldepth, message)
}

func Error(format string, args ...any) {
	if level < LevelError {
		return
	}
	message := fmtMessage(format, args...)
	errorLogger.Output(calldepth, message)
}
