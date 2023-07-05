package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
)

var log *slog.Logger

type model struct {
	slides   []vaxis.Model
	current  int
	keys     string
	keyClear *time.Timer
}

func (m *model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.InitMsg:
		m.slides = []vaxis.Model{
			&input{},
			newSimpleWidgets(),
			newTerm(),
			newTextInput(),
		}
		img := newImage()
		if img != nil {
			m.slides = append(m.slides, img)
		}
	case vaxis.Key:
		if msg.EventType == vaxis.EventRelease {
			if m.current >= 1 {
				if slide, ok := m.slides[m.current-1].(*input); ok {
					slide.Update(msg)
				}
			}
			break
		}

		m.keys += msg.String()
		if len(msg.String()) > 1 {
			m.keys += "+"
		}
		m.keyClear.Stop()
		m.keyClear = time.AfterFunc(1*time.Second, func() {
			vaxis.PostMsg(vaxis.FuncMsg{
				Func: func() {
					m.keys = ""
				},
			})
		})
		switch msg.String() {
		case "Ctrl+c":
			vaxis.Quit()
		case "Ctrl+l":
			vaxis.Refresh()
		case "Right":
			if m.current >= len(m.slides) {
				break
			}
			if m.current > 0 {
				m.slides[m.current-1].Update(vaxis.Visible(false))
			}
			m.current += 1
			if m.current > 0 {
				m.slides[m.current-1].Update(vaxis.Visible(true))
			}
		case "Left":
			if m.current <= 0 {
				break
			}
			if m.current > 0 {
				m.slides[m.current-1].Update(vaxis.Visible(false))
			}
			m.current -= 1
			if m.current > 0 {
				m.slides[m.current-1].Update(vaxis.Visible(true))
			}
		default:
			if m.current > 0 {
				m.slides[m.current-1].Update(msg)
			}
		}
	case vaxis.PasteMsg:
		if m.current > 0 {
			m.slides[m.current-1].Update(msg)
		}
	}
}

func (m *model) Draw(win vaxis.Window) {
	vaxis.Clear(win)
	vaxis.HideCursor()
	_, rows := win.Size()
	mid := fmt.Sprintf("%d of %d", m.current+1, 1+len(m.slides))
	w := uniseg.StringWidth(mid)
	vaxis.Print(align.BottomRight(win, w, 1), mid)
	w = uniseg.StringWidth(m.keys)
	vaxis.Print(align.BottomMiddle(win, w, 1), m.keys)
	switch m.current {
	case 0:
		blocks := []vaxis.Segment{
			{
				Text: "vaxis\n\n",
			},
			{
				Text:       "Ctrl+C to quit\n",
				Attributes: vaxis.AttrDim,
			},
			{
				Text:       "<Right> next slide\n",
				Attributes: vaxis.AttrDim,
			},
			{
				Text:       "<Left> previous slide\n",
				Attributes: vaxis.AttrDim,
			},
		}
		vaxis.PrintSegments(align.Center(win, 36, len(blocks)+1), blocks...)
	default:
		m.slides[m.current-1].Draw(vaxis.NewWindow(&win, 0, 0, -1, rows-1))
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
	vaxis.Init(vaxis.Options{
		Logger: log,
	})
	m := &model{
		keyClear: time.NewTimer(0),
	}
	if err := vaxis.Run(m); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
