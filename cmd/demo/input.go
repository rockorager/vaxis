package main

import (
	"bytes"
	"fmt"

	"git.sr.ht/~rockorager/rtk"
	"git.sr.ht/~rockorager/rtk/widgets/align"
)

type input struct {
	events []string
}

func (m *input) Update(msg rtk.Msg) {
	switch msg := msg.(type) {
	case rtk.Key:
		prefix := "[key]"
		switch msg.EventType {
		case rtk.EventPress:
			prefix = "[key::press]"
		case rtk.EventRelease:
			prefix = "[key::release]"
		case rtk.EventRepeat:
			prefix = "[key::repeat]"
		}
		val := fmt.Sprintf("%-16s%s", prefix, msg)
		m.events = append(m.events, val)
		if len(m.events) > 200 {
			m.events = m.events[100:]
		}
	}
}

func (m *input) Draw(srf rtk.Surface) {
	_, rows := srf.Size()
	srf = align.TopMiddle(srf, 36, rows-5)
	_, rows = srf.Size()
	top := len(m.events) - rows
	if top < 0 {
		top = 0
	}
	out := bytes.NewBuffer(nil)
	for i := top; i < len(m.events); i += 1 {
		out.WriteString(m.events[i] + "\n")
	}
	rtk.Print(srf, out.String())
}
