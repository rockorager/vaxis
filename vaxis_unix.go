//go:build darwin || freebsd || linux || netbsd || openbsd || zos

package vaxis

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/vaxis/log"
	"golang.org/x/sys/unix"
)

func (vx *Vaxis) setupSignals() {
	vx.mu.Lock()
	inBandResize := vx.caps.inBandResize
	vx.mu.Unlock()
	if !inBandResize {
		signal.Notify(
			vx.chSigWinSz,
			syscall.SIGWINCH,
		)
	}
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

// reportWinsize
func (vx *Vaxis) reportWinsize() (Resize, error) {
	vx.mu.Lock()
	inBandResize := vx.caps.inBandResize
	xtwinops := vx.xtwinops
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
	if xtwinops && reportSizeChars && reportSizePixels {
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
	log.Trace("requesting screen size from ioctl")
	ws, err := unix.IoctlGetWinsize(int(vx.tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		cws, err := vx.tty.Size()
		if err == nil {
			return cws, nil
		}

		return Resize{}, err
	}
	return Resize{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		XPixel: int(ws.Xpixel),
		YPixel: int(ws.Ypixel),
	}, nil
}
