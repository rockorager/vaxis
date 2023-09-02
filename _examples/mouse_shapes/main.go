package main

import (
	"git.sr.ht/~rockorager/vaxis"
)

type model struct {
	vertical         int
	horizontal       int
	resizeVertical   bool
	resizeHorizontal bool
	vx               *vaxis.Vaxis
}

func (m *model) Update(msg vaxis.Event) {
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "Ctrl+c", "q":
			m.vx.Close()
		}
	case vaxis.Mouse:
		m.vx.SetMouseShape(vaxis.MouseShapeDefault)
		if msg.EventType == vaxis.EventRelease {
			m.resizeHorizontal = false
			m.resizeVertical = false
		}
		if m.resizeVertical {
			m.horizontal = msg.Row
		}
		if m.resizeHorizontal {
			m.vertical = msg.Col
		}
		if msg.Row == m.horizontal {
			m.vx.SetMouseShape(vaxis.MouseShapeResizeVertical)
			if msg.EventType == vaxis.EventPress {
				m.resizeVertical = true
			}
		}
		if msg.Col == m.vertical {
			m.vx.SetMouseShape(vaxis.MouseShapeResizeHorizontal)
			if msg.EventType == vaxis.EventPress {
				m.resizeHorizontal = true
			}
		}
	default:
	}
}

func (m *model) Draw(win vaxis.Window) {
	w, h := win.Size()
	if m.vertical == 0 {
		m.vertical = w / 2
	}
	if m.horizontal == 0 {
		m.horizontal = h / 2
	}
	nw := win.New(0, 0, m.vertical, m.horizontal)
	ne := win.New(m.vertical, 0, -1, m.horizontal)
	se := win.New(m.vertical, m.horizontal, -1, -1)
	sw := win.New(0, m.horizontal, m.vertical, -1)
	nw.Fill(vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(1),
	})
	ne.Fill(vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(2),
	})
	se.Fill(vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(3),
	})
	sw.Fill(vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(4),
	})
}

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()
	model := &model{vx: vx}
	for ev := range vx.Events() {
		model.Update(ev)
		model.Draw(vx.Window())
		vx.Render()
	}
}
