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

var (
	app *rtk.App
	log *slog.Logger
)

type model struct {
	slides  []rtk.Model
	current int
}

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Init:
		m.slides = []rtk.Model{
			&input{},
		}
	case rtk.Key:
		if msg.EventType == rtk.EventRelease {
			break
		}
		switch msg.String() {
		case "c-c":
			app.Close()
		case "right":
			if m.current >= len(m.slides) {
				break
			}
			m.current += 1
		case "left":
			if m.current <= 0 {
				break
			}
			m.current -= 1
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
		blocks := []rtk.Block{
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
				Text:       "    <Left> previous slide",
				Attributes: rtk.AttrDim,
			},
		}
		rtk.PrintBlocks(align.Center(srf, 36, 5), blocks...)
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
	opts := &rtk.Options{
		LogHandler: handler,
	}
	log = slog.New(handler)
	app, err = rtk.NewApp(opts)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	m := &model{}
	if err := app.Run(m); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
