package main

import (
	"bytes"
	"fmt"
	"os"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/widgets/align"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
)

var log *slog.Logger

type model struct {
	slides  []rtk.Model
	current int
}

type visible bool

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Init:
		m.slides = []rtk.Model{
			&input{},
			newSimpleWidgets(),
		}
	case rtk.Key:
		if msg.EventType == rtk.EventRelease {
			break
		}
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
		}
	}
	if m.current > 0 {
		m.slides[m.current-1].Update(msg)
	}
}

func (m *model) Draw(srf rtk.Surface) {
	rtk.Clear(srf)
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
		rtk.PrintSegments(align.Center(srf, 36, len(blocks)+1), blocks...)
	default:
		m.slides[m.current-1].Draw(srf)
	}
	mid := fmt.Sprintf("%d of %d", m.current+1, 1+len(m.slides))
	w := uniseg.StringWidth(mid)
	rtk.Print(align.BottomMiddle(srf, w, 1), mid)
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
	m := &model{}
	if err := rtk.Run(m); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
