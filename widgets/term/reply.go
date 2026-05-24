package term

import (
	"context"
	"os"
	"time"

	"go.rockorager.dev/vaxis/log"
)

// termReply is a terminal-generated response to write back to the child PTY.
// The reply worker evaluates these in FIFO order so slow replies, such as OSC
// color queries to the host terminal, cannot be overtaken by later immediate
// replies.
type termReply func(context.Context) (string, bool)

var termReplyTimeout = 500 * time.Millisecond

func (vt *Model) startReplyWorker() {
	vt.stopReplyWorker()
	ctx, cancel := context.WithCancel(context.Background())
	vt.replyCancel = cancel
	vt.replyQueue = make(chan termReply, 1024)
	pty := vt.pty
	go vt.runReplyWorker(ctx, pty)
}

func (vt *Model) stopReplyWorker() {
	if vt.replyCancel == nil {
		return
	}
	vt.replyCancel()
	vt.replyCancel = nil
}

func (vt *Model) runReplyWorker(ctx context.Context, pty *os.File) {
	for {
		select {
		case <-ctx.Done():
			return
		case reply := <-vt.replyQueue:
			if ctx.Err() != nil {
				return
			}
			replyCtx, cancel := context.WithTimeout(ctx, termReplyTimeout)
			resp, ok := reply(replyCtx)
			cancel()
			if !ok || resp == "" {
				continue
			}
			if _, err := pty.WriteString(resp); err != nil {
				log.Error("[term] failed to write terminal reply: %v", err)
			}
		}
	}
}

// enqueueReply must not block the PTY parser. If a child floods us with
// terminal queries faster than we can answer them, drop replies rather than
// wedging the parser.
func (vt *Model) enqueueReply(reply termReply) {
	if reply == nil || vt.replyQueue == nil {
		return
	}
	select {
	case vt.replyQueue <- reply:
	default:
		log.Warn("[term] terminal reply queue full; dropping reply")
	}
}

func (vt *Model) enqueueReplyString(resp string) {
	if resp == "" {
		return
	}
	vt.enqueueReply(func(context.Context) (string, bool) {
		return resp, true
	})
}
