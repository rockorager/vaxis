package main

import (
	"bytes"
	"fmt"
	"time"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/log"
)

var app *rtk.App

type model struct {
	val int
}

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Init:
		log.Debugf("Initializing")
	case rtk.Key:
		switch msg.String() {
		case "<c-c>":
			app.Close()
		}
	case increment:
		m.val += 1
	}
}

func (m *model) View(srf rtk.Surface) {
	str := fmt.Sprintf("%d", m.val)
	for i, egc := range rtk.EGCs(str) {
		srf.SetCell(i, 0, rtk.Cell{
			EGC:        egc,
			Foreground: rtk.IndexColor(uint8(m.val)),
		})
	}
}

type increment struct{}

func main() {
	logBuf := bytes.NewBuffer(nil)
	log.SetLevel(log.LevelTrace)
	log.SetOutput(logBuf)
	defer func() {
		fmt.Print(logBuf.String())
	}()

	log.Infof("Demo starting")

	var err error
	app, err = rtk.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(60 * time.Millisecond)
			app.PostMsg(increment{})
		}
	}()

	m := &model{
		val: 0,
	}
	if err := app.Run(m); err != nil {
		log.Fatal(err)
	}
}
