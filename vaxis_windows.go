//go:build windows

package vaxis

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/vaxis/log"
)

func (vx *Vaxis) setupSignals() {
	signal.Notify(
		vx.chSigKill,
		// kill signals
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGTERM,
	)
}

func (vx *Vaxis) reportWinsize() (Resize, error) {
	vx.mu.Lock()
	inBandResize := vx.caps.inBandResize
	reportSizeChars := vx.caps.reportSizeChars
	reportSizePixels := vx.caps.reportSizePixels
	vx.mu.Unlock()
	if inBandResize {
		select {
		case report := <-vx.chSizeReport:
			if report.chars && report.pixels {
				return report.size, nil
			}
		default:
		}
	}
	if reportSizeChars && reportSizePixels {
		log.Trace("requesting screen size from terminal")
		vx.drainSizeReports()
		vx.writeControlString(textAreaSize)
		deadline := time.NewTimer(100 * time.Millisecond)
		defer deadline.Stop()
		size := Resize{}
		chars := false
		pixels := false
		for !chars || !pixels {
			select {
			case <-deadline.C:
				return Resize{}, fmt.Errorf("screen size request deadline exceeded")
			case report := <-vx.chSizeReport:
				if report.chars {
					size.Cols = report.size.Cols
					size.Rows = report.size.Rows
					chars = true
				}
				if report.pixels {
					size.XPixel = report.size.XPixel
					size.YPixel = report.size.YPixel
					pixels = true
				}
			}
		}
		return size, nil
	}
	log.Trace("requesting screen size from console")
	return vx.tty.Size()
}
