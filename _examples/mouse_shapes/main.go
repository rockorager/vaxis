package main

import (
	"git.sr.ht/~rockorager/vaxis"
)

type model struct {
	vertical         int
	horizontal       int
	resizeVertical   bool
	resizeHorizontal bool
}

func (m *model) Update(msg vaxis.Msg) {
	switch msg := msg.(type) {
	case vaxis.Key:
		switch msg.String() {
		case "Ctrl+c", "q":
			vaxis.Close()
		}
	case vaxis.Mouse:
		vaxis.SetMouseShape(vaxis.MouseShapeDefault)
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
			vaxis.SetMouseShape(vaxis.MouseShapeResizeVertical)
			if msg.EventType == vaxis.EventPress {
				m.resizeVertical = true
			}
		}
		if msg.Col == m.vertical {
			vaxis.SetMouseShape(vaxis.MouseShapeResizeHorizontal)
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
	nw := vaxis.NewWindow(&win, 0, 0, m.vertical, m.horizontal)
	ne := vaxis.NewWindow(&win, m.vertical, 0, -1, m.horizontal)
	se := vaxis.NewWindow(&win, m.vertical, m.horizontal, -1, -1)
	sw := vaxis.NewWindow(&win, 0, m.horizontal, m.vertical, -1)
	vaxis.Fill(nw, vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(1),
	})
	vaxis.Fill(ne, vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(2),
	})
	vaxis.Fill(se, vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(3),
	})
	vaxis.Fill(sw, vaxis.Text{
		Content:    " ",
		Background: vaxis.IndexColor(4),
	})
}

func main() {
	err := vaxis.Init(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	model := &model{}
	vaxis.Run(model)
}
