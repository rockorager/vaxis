package main

import (
	"bytes"
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/align"
)

type input struct {
	events []string
}

func (m *input) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		prefix := "[key]"
		switch msg.EventType {
		case vaxis.EventPress:
			prefix = "[key::press]"
		case vaxis.EventRelease:
			prefix = "[key::release]"
		case vaxis.EventRepeat:
			prefix = "[key::repeat]"
		}
		val := fmt.Sprintf("%-16s%s", prefix, msg)
		m.events = append(m.events, val)
		if len(m.events) > 200 {
			m.events = m.events[100:]
		}
	case vaxis.Mouse:
		prefix := "[mouse]"
		switch msg.EventType {
		case vaxis.EventPress:
			prefix = "[mouse::press]"
		case vaxis.EventRelease:
			prefix = "[mouse::release]"
		case vaxis.EventMotion:
			prefix = "[mouse::motion]"
		}
		button := ""
		switch msg.Modifiers {
		case vaxis.ModShift:
			button += "s-"
		case vaxis.ModAlt:
			button += "a-"
		case vaxis.ModCtrl:
			button += "c-"
		}
		switch msg.Button {
		case vaxis.MouseLeftButton:
			button += "left"
		case vaxis.MouseMiddleButton:
			button += "middle"
		case vaxis.MouseRightButton:
			button += "right"
		case vaxis.MouseNoButton:
			button += "none"
		case vaxis.MouseWheelUp:
			button += "wheel-up"
		case vaxis.MouseWheelDown:
			button += "wheel-down"
		case vaxis.MouseButton8:
			button += "button 8"
		case vaxis.MouseButton9:
			button += "button 9"
		case vaxis.MouseButton10:
			button += "button 10"
		case vaxis.MouseButton11:
			button += "button 11"
		}
		val := fmt.Sprintf("%-16s %s row=%d col=%d", prefix, button, msg.Row, msg.Col)
		m.events = append(m.events, val)
		if len(m.events) > 200 {
			m.events = m.events[100:]
		}
	}
}

func (m *input) Draw(win vaxis.Window) {
	_, rows := win.Size()
	win = align.TopMiddle(win, 50, rows-5)
	_, rows = win.Size()
	top := len(m.events) - rows
	if top < 0 {
		top = 0
	}
	out := bytes.NewBuffer(nil)
	for i := top; i < len(m.events); i += 1 {
		out.WriteString(m.events[i] + "\n")
	}
	vaxis.Print(win, vaxis.Text{Content: out.String()})
}
