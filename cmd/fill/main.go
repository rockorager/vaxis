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
	rowOff int
	colOff int

	colDir int
	rowDir int

	cols int
	rows int

	color    int
	colorDir int
	fill     rtk.Cell

	box   rtk.Surface
	clear bool
}

func (m *model) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Key:
		switch msg.String() {
		case "<c-c>":
			app.Close()
		}
	case increment:
		if m.color == 255 {
			m.colorDir = -1
		}
		if m.color == 0 {
			m.colorDir = 1
		}
		m.color += m.colorDir
		m.fill.Background = rtk.IndexColor(uint8(m.color))
		m.colOff += m.colDir
		m.rowOff += m.rowDir
	}
}

func (m *model) View(srf rtk.Surface) {
	cols, rows := srf.Size()
	if m.colOff+m.cols >= cols {
		m.colDir = -1
	}
	if m.colOff <= 0 {
		m.colDir = 1
	}
	if m.rowOff+m.rows >= rows {
		m.rowDir = -1
	}
	if m.rowOff <= 0 {
		m.rowDir = 1
	}
	if m.box == nil {
		m.box = rtk.NewSubSurface(srf, m.colOff, m.rowOff, m.cols, m.rows)
	}
	rtk.Clear(srf)
	m.box.Move(m.colOff, m.rowOff)
	rtk.Fill(m.box, m.fill)
}

type increment struct{}

func main() {
	logBuf := bytes.NewBuffer(nil)
	log.SetLevel(log.LevelTrace)
	log.SetOutput(logBuf)
	defer func() {
		fmt.Print(logBuf.String())
	}()

	model := &model{
		fill: rtk.Cell{
			EGC: " ",
		},
		cols: 16,
		rows: 8,
	}

	var err error
	app, err = rtk.NewApp()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			app.PostMsg(increment{})
		}
	}()
	if err := app.Run(model); err != nil {
		log.Fatal(err)
	}
}
