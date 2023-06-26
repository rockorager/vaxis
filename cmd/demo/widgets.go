package main

import (
	"context"
	"time"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/widgets/progress"
	"git.sr.ht/~rockorager/rtk/widgets/spinner"
)

type simpleWidgets struct {
	ctx      context.Context
	cancel   context.CancelFunc
	spinner1 *spinner.Model
	spinner2 *spinner.Model
	spinner3 *spinner.Model

	progress1 *progress.Model
}

func (s *simpleWidgets) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case visible:
		switch msg {
		case true:
			s.spinner1.Start()
			s.spinner2.Start()
			s.spinner3.Start()
			s.ctx, s.cancel = context.WithCancel(context.Background())
			ticker := time.NewTicker(1 * time.Millisecond)
			go func() {
				p := 0
				total := 100
				for {
					select {
					case <-s.ctx.Done():
						ticker.Stop()
						return
					case <-ticker.C:
						p += 1
						rtk.SendMsg(progress.DataMsg{
							Progress: float64(p),
							Total:    float64(total),
						}, s.progress1)

					}
				}
			}()
		case false:
			s.cancel()
			s.spinner1.Stop()
			s.spinner2.Stop()
			s.spinner3.Stop()
		}
	}
}

func (s *simpleWidgets) Draw(win rtk.Window) {
	s.spinner1.Draw(rtk.NewWindow(&win, 0, 0, 1, 1))
	s.spinner2.Draw(rtk.NewWindow(&win, 1, 0, 1, 1))
	s.spinner3.Draw(rtk.NewWindow(&win, 2, 0, 1, 1))
	s.progress1.Draw(rtk.NewWindow(&win, 0, 3, -1, 1))
}

func newSimpleWidgets() *simpleWidgets {
	s := &simpleWidgets{
		spinner1:  spinner.New(100 * time.Millisecond),
		spinner2:  spinner.New(10 * time.Millisecond),
		spinner3:  spinner.New(500 * time.Millisecond),
		progress1: progress.New(),
	}
	return s
}
