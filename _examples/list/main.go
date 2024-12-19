package main

import (
	"fmt"
	"os"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/list"
)

func ProduceLines(path string) ([]string, error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("cannot list directory: %v", err)
	}

	messages := make([]string, len(dir))
	for i, entry := range dir {
		messages[i] = entry.Name()
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

func main() {
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(fmt.Errorf("failed to initialise: %v", err))
	}
	defer vx.Close()

	var path string
	if len(os.Args) < 2 {
		path, err = os.Getwd()
		if err != nil {
			panic(fmt.Errorf("failed to determine current working directory"))
		}
	} else {
		path = os.Args[1]
	}

	lines, err := ProduceLines(path)
	if err != nil {
		panic(fmt.Errorf("could not read messages: %v", err))
	}

	list := list.New(lines)

	for ev := range vx.Events() {
		win := vx.Window()
		win.Clear()

		width, height := win.Size()
		listWin := win.New(0, 1, width, height-2)

		switch ev := ev.(type) {
		case vaxis.Key:
			switch ev.String() {
			case "Ctrl+c", "q":
				return
			case "Down", "j":
				list.Down()
			case "Up", "k":
				list.Up()
			case "End":
				list.End()
			case "Home":
				list.Home()
			case "Page_Down":
				list.PageDown(win)
			case "Page_Up":
				list.PageUp(win)
			}
		}
		list.Draw(listWin)
		vx.Render()
	}
}
