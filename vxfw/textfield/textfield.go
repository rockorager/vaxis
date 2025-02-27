package textfield

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/vxfw"
	"github.com/rivo/uniseg"
)

const scrolloff = 4

var truncator = vaxis.Character{
	Grapheme: "…",
	Width:    1,
}

type TextField struct {
	Value    string
	Style    vaxis.Style
	OnSubmit func(line string) (vxfw.Command, error)
	OnChange func(line string) (vxfw.Command, error)

	cursor uint
	n      uint
}

func New() *TextField {
	return &TextField{}
}

func (tf *TextField) HandleEvent(ev vaxis.Event, ph vxfw.EventPhase) (vxfw.Command, error) {
	switch ev := ev.(type) {
	case vaxis.Key:
		if ev.EventType == vaxis.EventRelease {
			return nil, nil
		}
		if len(ev.Text) > 0 {
			pre := tf.Value
			cmd := tf.InsertStringAtCursor(ev.Text)
			return tf.checkChanged(cmd, pre)
		}

		// Cursor to Beginning of line
		if ev.Matches('a', vaxis.ModCtrl) || ev.Matches(vaxis.KeyHome) {
			return tf.CursorTo(0), nil
		}
		// Cursor to end of line
		if ev.Matches('e', vaxis.ModCtrl) || ev.Matches(vaxis.KeyEnd) {
			return tf.CursorTo(tf.n), nil
		}
		// Cursor forward one character
		if ev.Matches('f', vaxis.ModCtrl) || ev.Matches(vaxis.KeyRight) {
			return tf.CursorTo(tf.cursor + 1), nil
		}
		// Cursor backward one character
		if ev.Matches('b', vaxis.ModCtrl) || ev.Matches(vaxis.KeyLeft) {
			if tf.cursor == 0 {
				return nil, nil
			}
			return tf.CursorTo(tf.cursor - 1), nil
		}
		// Delete character right of cursor
		if ev.Matches('d', vaxis.ModCtrl) || ev.Matches(vaxis.KeyDelete) {
			pre := tf.Value
			cmd := tf.DeleteCharRightOfCursor()
			return tf.checkChanged(cmd, pre)
		}
		// Delete character left of cursor
		if ev.Matches('h', vaxis.ModCtrl) || ev.Matches(vaxis.KeyBackspace) {
			pre := tf.Value
			cmd := tf.DeleteCharLeftOfCursor()
			return tf.checkChanged(cmd, pre)
		}
		// Delete to end of line
		if ev.Matches('k', vaxis.ModCtrl) {
			pre := tf.Value
			cmd := tf.DeleteCursorToEndOfLine()
			return tf.checkChanged(cmd, pre)
		}
		// Submit
		if ev.Matches(vaxis.KeyEnter) {
			defer tf.Reset()
			if tf.OnSubmit != nil {
				return tf.OnSubmit(tf.Value)
			}
			return vxfw.ConsumeAndRedraw(), nil
		}
	}
	return nil, nil
}

func (tf *TextField) checkChanged(cmd vxfw.Command, pre string) (vxfw.Command, error) {
	// If the value is the same, we return the cmd we were passed
	if tf.Value == pre {
		return cmd, nil
	}

	// Value is different. If we have an OnChange handler, we call it
	if tf.OnChange != nil {
		cmd2, err := tf.OnChange(tf.Value)
		if err != nil {
			return nil, err
		}
		return []vxfw.Command{cmd, cmd2}, nil
	}

	// Otherwise, return what we had
	return cmd, nil
}

func (tf *TextField) Reset() {
	tf.n = 0
	tf.Value = ""
	tf.cursor = 0
}

func (tf *TextField) InsertStringAtCursor(s string) vxfw.Command {
	tf.insertStringAtCursor(s)
	tf.n = graphemeCountInString(tf.Value)
	return vxfw.ConsumeAndRedraw()
}

func (tf *TextField) CursorTo(i uint) vxfw.Command {
	i = min(i, tf.n)

	// Nothing to do if state is the same
	if i == tf.cursor {
		return nil
	}

	tf.cursor = i
	return vxfw.ConsumeAndRedraw()
}

func (tf *TextField) DeleteCharRightOfCursor() vxfw.Command {
	// Nothing to do if at end of line
	if tf.n == tf.cursor {
		return nil
	}

	var (
		cluster      = ""
		rest         = tf.Value
		state        = -1
		i       uint = 0
		next         = strings.Builder{}
	)

	for len(rest) > 0 {
		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if i == tf.cursor {
			i += 1
			continue
		}
		i += 1
		next.WriteString(cluster)
	}
	tf.Value = next.String()
	return vxfw.ConsumeAndRedraw()
}

func (tf *TextField) DeleteCharLeftOfCursor() vxfw.Command {
	// Nothing to do if at beginning of line
	if tf.cursor == 0 {
		return nil
	}

	var (
		cluster      = ""
		rest         = tf.Value
		state        = -1
		i       uint = 0
		next         = strings.Builder{}
	)

	for len(rest) > 0 {
		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		i += 1
		if i == tf.cursor {
			continue
		}
		// insert the string
		next.WriteString(cluster)
	}
	tf.Value = next.String()
	tf.cursor -= 1
	return vxfw.ConsumeAndRedraw()
}

func (tf *TextField) DeleteCursorToEndOfLine() vxfw.Command {
	if tf.cursor == tf.n {
		return nil
	}

	var (
		cluster      = ""
		rest         = tf.Value
		state        = -1
		i       uint = 0
		next         = strings.Builder{}
	)

	for len(rest) > 0 {
		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if i == tf.cursor {
			break
		}
		i += 1
		next.WriteString(cluster)
	}
	tf.Value = next.String()
	return vxfw.ConsumeAndRedraw()
}

func (tf *TextField) Draw(ctx vxfw.DrawContext) (vxfw.Surface, error) {
	if ctx.Max.Width == 0 || ctx.Max.Height == 0 {
		return vxfw.Surface{}, nil
	}

	s := vxfw.NewSurface(ctx.Max.Width, 1, tf)
	s.Cursor = &vxfw.CursorState{
		Row:   0,
		Col:   0,
		Shape: vaxis.CursorBlock,
	}
	chars := ctx.Characters(tf.Value)
	var (
		i   uint
		col uint16
	)
	for _, char := range chars {
		cell := vaxis.Cell{
			Character: char,
			Style:     tf.Style,
		}
		s.WriteCell(col, 0, cell)

		col += uint16(char.Width)
		i += 1
		if i == tf.cursor {
			s.Cursor.Col = col
		}
	}
	if i < tf.cursor {
		s.Cursor.Col = col
	}

	return s, nil
}

func (tf *TextField) insertStringAtCursor(s string) {
	// Find the cursor position
	var (
		cluster      = ""
		rest         = tf.Value
		state        = -1
		i       uint = 0
		next         = strings.Builder{}
	)

	count := graphemeCountInString(s)

	for {
		if len(rest) > 0 && i < tf.cursor {
			cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
			next.WriteString(cluster)
			i += 1
			continue
		}
		// insert the string
		next.WriteString(s)
		// advance the cursor
		tf.cursor = max(tf.cursor, tf.cursor+count)
		next.WriteString(rest)
		break
	}

	tf.Value = next.String()
}

func graphemeCountInString(s string) uint {
	var (
		rest       = s
		state      = -1
		count uint = 0
	)

	for len(rest) > 0 {
		_, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		count += 1
	}
	return count
}
