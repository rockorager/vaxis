package main

import (
	"bytes"
	"fmt"
	"strings"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/log"
)

var app *rtk.App

type model struct {
	val []string
}

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Key:
		out := msg.String()
		switch msg.EventType {
		case rtk.EventPress:
			out += " (press)"
		case rtk.EventRepeat:
			out += " (repeat)"
		case rtk.EventRelease:
			out += " (release)"
		}
		m.val = append(m.val, out)
		switch msg.String() {
		case "<c-c>":
			app.Close()
		}
	}
}

func (m *model) Draw(srf rtk.Surface) {
	_, rows := srf.Size()
	top := len(m.val) - rows
	visible := []string{}
	for i, line := range m.val {
		if i < top {
			continue
		}
		visible = append(visible, line)
	}
	rtk.Clear(srf)
	rtk.Print(srf, strings.Join(visible, "\n"))
}

func main() {
	logBuf := bytes.NewBuffer(nil)
	log.SetLevel(log.LevelTrace)
	log.SetOutput(logBuf)
	defer func() {
		fmt.Print(logBuf.String())
	}()

	var err error
	app, err = rtk.NewApp()
	if err != nil {
		log.Fatal(err)
	}
	m := &model{}
	if err := app.Run(m); err != nil {
		log.Fatal(err)
	}
}
