package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/widgets/align"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
)

var log *slog.Logger

type model struct {
	slides   []rtk.Model
	current  int
	keys     string
	keyClear *time.Timer
}

type visible bool

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Init:
		m.slides = []rtk.Model{
			&input{},
			newSimpleWidgets(),
			newTerm(),
			newTextInput(),
		}
	case rtk.Key:
		if msg.EventType == rtk.EventRelease {
			break
		}

		m.keys += msg.String()
		if len(msg.String()) > 1 {
			m.keys += "+"
		}
		m.keyClear.Stop()
		m.keyClear = time.AfterFunc(1*time.Second, func() {
			rtk.PostMsg(rtk.FuncMsg{
				Func: func() {
					m.keys = ""
				},
			})
		})
		switch msg.String() {
		case "Ctrl+c":
			rtk.Quit()
		case "Ctrl+l":
			rtk.Refresh()
		case "Right":
			if m.current >= len(m.slides) {
				break
			}
			if m.current > 0 {
				m.slides[m.current-1].Update(visible(false))
			}
			m.current += 1
			if m.current > 0 {
				m.slides[m.current-1].Update(visible(true))
			}
		case "Left":
			if m.current <= 0 {
				break
			}
			if m.current > 0 {
				m.slides[m.current-1].Update(visible(false))
			}
			m.current -= 1
			if m.current > 0 {
				m.slides[m.current-1].Update(visible(true))
			}
		default:
			if m.current > 0 {
				m.slides[m.current-1].Update(msg)
			}
		}
	case rtk.Paste:
	}
}

func (m *model) Draw(win rtk.Window) {
	rtk.Clear(win)
	rtk.HideCursor()
	_, rows := win.Size()
	mid := fmt.Sprintf("%d of %d", m.current+1, 1+len(m.slides))
	w := uniseg.StringWidth(mid)
	rtk.Print(align.BottomRight(win, w, 1), mid)
	w = uniseg.StringWidth(m.keys)
	rtk.Print(align.BottomMiddle(win, w, 1), m.keys)
	switch m.current {
	case 0:
		blocks := []rtk.Segment{
			{
				Text: "rockorager's modern terminal toolkit\n\n",
			},
			{
				Text:       "    Ctrl+C to quit\n",
				Attributes: rtk.AttrDim,
			},
			{
				Text:       "    <Right> next slide\n",
				Attributes: rtk.AttrDim,
			},
			{
				Text:       "    <Left> previous slide\n",
				Attributes: rtk.AttrDim,
			},
		}
		rtk.PrintSegments(align.Center(win, 36, len(blocks)+1), blocks...)
	default:
		m.slides[m.current-1].Draw(rtk.NewWindow(&win, 0, 0, -1, rows-1))
	}
}

func main() {
	logBuf := bytes.NewBuffer(nil)
	defer func() {
		fmt.Print(logBuf.String())
	}()

	var err error
	handler := slog.NewTextHandler(logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	log = slog.New(handler)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	rtk.Logger = log
	m := &model{
		keyClear: time.NewTimer(0),
	}
	if err := rtk.Run(m); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
