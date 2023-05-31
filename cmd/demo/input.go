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
	case rtk.Mouse:
		prefix := "[mouse]"
		switch msg.EventType {
		case rtk.EventPress:
			prefix = "[mouse::press]"
		case rtk.EventRelease:
			prefix = "[mouse::release]"
		case rtk.EventMotion:
			prefix = "[mouse::motion]"
		}
		button := ""
		switch msg.Modifiers {
		case rtk.ModShift:
			button += "s-"
		case rtk.ModAlt:
			button += "a-"
		case rtk.ModCtrl:
			button += "c-"
		}
		switch msg.Button {
		case rtk.MouseLeftButton:
			button += "left"
		case rtk.MouseMiddleButton:
			button += "middle"
		case rtk.MouseRightButton:
			button += "right"
		case rtk.MouseNoButton:
			button += "none"
		case rtk.MouseWheelUp:
			button += "wheel-up"
		case rtk.MouseWheelDown:
			button += "wheel-down"
		case rtk.MouseButton8:
			button += "button 8"
		case rtk.MouseButton9:
			button += "button 9"
		case rtk.MouseButton10:
			button += "button 10"
		case rtk.MouseButton11:
			button += "button 11"
		}
		val := fmt.Sprintf("%-16s %s row=%d col=%d", prefix, button, msg.Row, msg.Col)
		m.events = append(m.events, val)
		if len(m.events) > 200 {
			m.events = m.events[100:]
		}
	}
}

func (m *input) Draw(srf rtk.Surface) {
	_, rows := srf.Size()
	srf = align.TopMiddle(srf, 50, rows-5)
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
