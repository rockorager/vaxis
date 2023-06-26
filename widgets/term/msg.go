package term

import "git.sr.ht/~rockorager/rtk"

type Redraw struct {
	Term   *Model
	Window rtk.Window
}
