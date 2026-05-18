//go:build windows

package vaxis

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procCreateEvent                   = kernel32.NewProc("CreateEventW")
	procGetConsoleMode                = kernel32.NewProc("GetConsoleMode")
	procGetConsoleScreenBufferInfo    = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetNumberOfConsoleInputEvents = kernel32.NewProc("GetNumberOfConsoleInputEvents")
	procReadConsoleInput              = kernel32.NewProc("ReadConsoleInputW")
	procSetConsoleMode                = kernel32.NewProc("SetConsoleMode")
	procSetEvent                      = kernel32.NewProc("SetEvent")
	procWaitForMultipleObjects        = kernel32.NewProc("WaitForMultipleObjects")
)

const (
	waitObject0 = 0
	waitFailed  = 0xFFFFFFFF
	infinite    = 0xFFFFFFFF

	keyEvent    uint16 = 0x0001
	resizeEvent uint16 = 0x0004

	enableEchoInput              = 0x0004
	enableLineInput              = 0x0002
	enableProcessedInput         = 0x0001
	enableMouseInput             = 0x0010
	enableWindowInput            = 0x0008
	enableExtendedFlags          = 0x0080
	enableVirtualTerminalInput   = 0x0200
	enableVirtualTerminalOutput  = 0x0004
	disableNewlineAutoReturn     = 0x0008
	enableLVBGridWorldwideOutput = 0x0010
)

type inputRecord struct {
	typ  uint16
	_    uint16
	data [16]byte
}

type coord struct {
	x int16
	y int16
}

type smallRect struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type consoleScreenBufferInfo struct {
	size       coord
	cursor     coord
	attrs      uint16
	window     smallRect
	maxWinSize coord
}

type windowsTTY struct {
	in        syscall.Handle
	out       syscall.Handle
	inFile    *os.File
	outFile   *os.File
	inMode    uint32
	outMode   uint32
	buf       chan byte
	done      chan struct{}
	stopOnce  sync.Once
	cancel    syscall.Handle
	wg        sync.WaitGroup
	mu        sync.Mutex
	closed    bool
	surrogate rune
}

func openTTY(string) (tty, error) {
	in, err := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	out, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil {
		_ = syscall.CloseHandle(in)
		return nil, err
	}
	t := &windowsTTY{
		in:      in,
		out:     out,
		inFile:  os.NewFile(uintptr(in), "CONIN$"),
		outFile: os.NewFile(uintptr(out), "CONOUT$"),
		buf:     make(chan byte, 1024),
		done:    make(chan struct{}),
	}
	if err := getConsoleMode(t.in, &t.inMode); err != nil {
		_ = t.Close()
		return nil, err
	}
	if err := getConsoleMode(t.out, &t.outMode); err != nil {
		_ = t.Close()
		return nil, err
	}
	return t, nil
}

func (t *windowsTTY) Read(p []byte) (int, error) {
	select {
	case b, ok := <-t.buf:
		if !ok {
			return 0, io.EOF
		}
		p[0] = b
	case <-t.done:
		return 0, io.EOF
	}
	n := 1
	for n < len(p) {
		select {
		case b, ok := <-t.buf:
			if !ok {
				return n, nil
			}
			p[n] = b
			n += 1
		default:
			return n, nil
		}
	}
	return n, nil
}

func (t *windowsTTY) Write(p []byte) (int, error) {
	return t.outFile.Write(p)
}

func (t *windowsTTY) Fd() uintptr {
	return uintptr(t.out)
}

func (t *windowsTTY) SetRaw() error {
	inMode := t.inMode
	inMode &^= enableEchoInput
	inMode &^= enableLineInput
	inMode &^= enableProcessedInput
	inMode &^= enableMouseInput
	inMode |= enableExtendedFlags
	inMode |= enableWindowInput
	inMode |= enableVirtualTerminalInput
	if err := setConsoleMode(t.in, inMode); err != nil {
		return err
	}
	outMode := t.outMode
	outMode |= enableVirtualTerminalOutput
	outMode |= disableNewlineAutoReturn
	outMode |= enableLVBGridWorldwideOutput
	return setConsoleMode(t.out, outMode)
}

func (t *windowsTTY) Reset() error {
	if err := setConsoleMode(t.in, t.inMode); err != nil {
		return err
	}
	return setConsoleMode(t.out, t.outMode)
}

func (t *windowsTTY) Size() (Resize, error) {
	var info consoleScreenBufferInfo
	if err := getConsoleScreenBufferInfo(t.out, &info); err != nil {
		return Resize{}, err
	}
	return Resize{
		Cols: int(info.window.right - info.window.left + 1),
		Rows: int(info.window.bottom - info.window.top + 1),
	}, nil
}

func (t *windowsTTY) StartInput(vx *Vaxis) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	cancel, _, err := procCreateEvent.Call(0, 1, 0, 0)
	if cancel == 0 {
		return err
	}
	t.cancel = syscall.Handle(cancel)
	t.wg.Add(1)
	go t.readConsoleInput(vx)
	return nil
}

func (t *windowsTTY) StopInput() error {
	t.stopOnce.Do(func() {
		t.mu.Lock()
		cancel := t.cancel
		t.mu.Unlock()
		close(t.done)
		if cancel != 0 {
			_, _, _ = procSetEvent.Call(uintptr(cancel))
		}
	})
	t.wg.Wait()
	return nil
}

func (t *windowsTTY) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()
	_ = t.StopInput()
	if t.cancel != 0 {
		_ = syscall.CloseHandle(t.cancel)
	}
	_ = t.inFile.Close()
	return t.outFile.Close()
}

func (t *windowsTTY) readConsoleInput(vx *Vaxis) {
	defer t.wg.Done()
	for {
		if err := t.readConsoleInputOnce(vx); err != nil {
			return
		}
	}
}

func (t *windowsTTY) readConsoleInputOnce(vx *Vaxis) error {
	handles := []syscall.Handle{t.cancel, t.in}
	wait, _, waitErr := procWaitForMultipleObjects.Call(
		uintptr(len(handles)),
		uintptr(unsafe.Pointer(&handles[0])),
		0,
		infinite,
	)
	switch wait {
	case waitObject0:
		return errors.New("cancelled")
	case waitObject0 + 1:
	default:
		if wait == waitFailed {
			return waitErr
		}
		return nil
	}
	var nrec int32
	rv, _, err := procGetNumberOfConsoleInputEvents.Call(
		uintptr(t.in),
		uintptr(unsafe.Pointer(&nrec)),
	)
	if rv == 0 {
		return err
	}
	if nrec == 0 {
		return nil
	}
	records := make([]inputRecord, nrec)
	rv, _, err = procReadConsoleInput.Call(
		uintptr(t.in),
		uintptr(unsafe.Pointer(&records[0])),
		uintptr(nrec),
		uintptr(unsafe.Pointer(&nrec)),
	)
	if rv == 0 {
		return err
	}
	for i := 0; i < int(nrec); i += 1 {
		switch records[i].typ {
		case keyEvent:
			t.handleKey(records[i])
		case resizeEvent:
			size, err := t.Size()
			if err == nil {
				vx.PostEventBlocking(size)
			}
		}
	}
	return nil
}

func (t *windowsTTY) handleKey(record inputRecord) {
	keyDown := binary.LittleEndian.Uint32(record.data[0:]) != 0
	if !keyDown {
		return
	}
	repeat := binary.LittleEndian.Uint16(record.data[4:])
	char := rune(binary.LittleEndian.Uint16(record.data[10:]))
	if char == 0 {
		return
	}
	if char >= 0xD800 && char <= 0xDBFF {
		t.surrogate = char
		return
	}
	if char >= 0xDC00 && char <= 0xDFFF {
		char = utf16.DecodeRune(t.surrogate, char)
		t.surrogate = 0
	}
	bytes := []byte(string(char))
	for ; repeat > 0; repeat -= 1 {
		for _, b := range bytes {
			select {
			case t.buf <- b:
			case <-t.done:
				return
			}
		}
	}
}

func getConsoleMode(handle syscall.Handle, mode *uint32) error {
	rv, _, err := procGetConsoleMode.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(mode)),
	)
	if rv == 0 {
		return err
	}
	return nil
}

func setConsoleMode(handle syscall.Handle, mode uint32) error {
	rv, _, err := procSetConsoleMode.Call(
		uintptr(handle),
		uintptr(mode),
	)
	if rv == 0 {
		return err
	}
	return nil
}

func getConsoleScreenBufferInfo(handle syscall.Handle, info *consoleScreenBufferInfo) error {
	rv, _, err := procGetConsoleScreenBufferInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(info)),
	)
	if rv == 0 {
		return err
	}
	return nil
}
