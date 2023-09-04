package textinput

import (
	"strings"
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
	"golang.org/x/exp/slices"
)

const scrolloff = 4

type Model struct {
	Prompt  vaxis.Text
	Content vaxis.Text
	// TODO handle the cursor better with the new wrapping
	cursor int // the x position of the cursor, relative to the start of Content
	offset int
}

func (m *Model) Update(msg vaxis.Event) {
	chars := vaxis.Characters(m.Content.Content)
	switch msg := msg.(type) {
	case vaxis.PasteEvent:
		pChars := vaxis.Characters(string(msg))
		chars = slices.Insert(chars, m.cursor, pChars...)
		m.cursor += len(pChars)
	case vaxis.Key:
		if msg.EventType == vaxis.EventRelease {
			return
		}
		switch msg.String() {
		case "Ctrl+a":
			// Beginning of line
			m.cursor = 0
		case "Ctrl+e":
			// End of line
			m.cursor = len(chars)
		case "Ctrl+f", "Right":
			// forward one character
			m.cursor += 1
		case "Ctrl+b", "Left":
			// backward one character
			m.cursor -= 1
		case "Alt+f", "Ctrl+Right":
			// Forward one word
			// skip non-alphanumerics
			for i := m.cursor; i < len(chars); i += 1 {
				if !isAlphaNumeric(chars[i]) {
					m.cursor += 1
					continue
				}
				break
			}
			for i := m.cursor; i < len(chars); i += 1 {
				if isAlphaNumeric(chars[i]) {
					m.cursor += 1
					continue
				}
				break
			}
		case "Alt+b", "Ctrl+Left":
			// backward one word
			// skip non-alphanumerics
			m.cursor -= 1
			if m.cursor >= len(chars) {
				m.cursor = len(chars) - 1
			}
			for i := m.cursor; i >= 0; i -= 1 {
				if !isAlphaNumeric(chars[i]) {
					m.cursor -= 1
					continue
				}
				break
			}
			for i := m.cursor; i >= 0; i -= 1 {
				if isAlphaNumeric(chars[i]) {
					m.cursor -= 1
					continue
				}
				m.cursor += 1
				break
			}
		case "Ctrl+d", "Delete":
			// delete character under cursor
			switch {
			case m.cursor == len(m.Content.Content):
				chars = chars[:m.cursor]
			default:
				chars = append(chars[:m.cursor], chars[m.cursor+1:]...)
			}
		case "Ctrl+k":
			chars = chars[:m.cursor]
		case "Ctrl+u":
			chars = chars[m.cursor:]
			m.cursor = 0
		case "Ctrl+h", "BackSpace":
			// delete character behind cursor
			switch {
			case m.cursor == 0:
				return
			case m.cursor == len(m.Content.Content):
				chars = chars[:m.cursor-1]
			default:
				chars = append(chars[:m.cursor-1], chars[m.cursor:]...)
			}
			m.cursor -= 1
		default:
			if msg.Modifiers&vaxis.ModCtrl != 0 {
				return
			}
			if msg.Modifiers&vaxis.ModAlt != 0 {
				return
			}
			if msg.Modifiers&vaxis.ModSuper != 0 {
				return
			}
			if unicode.IsGraphic(msg.Codepoint) {
				egc := vaxis.Character{
					Grapheme: string(msg.Codepoint),
				}
				chars = slices.Insert(chars, m.cursor, egc)
				m.cursor += 1
			}
		}
	}
	s := &strings.Builder{}
	for _, char := range chars {
		s.WriteString(char.Grapheme)
	}
	if m.cursor > len(chars) {
		m.cursor = len(chars)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.Content.Content = s.String()
}

func (m *Model) Draw(win vaxis.Window) {
	winW, _ := win.Size()
	col := 0
	for _, char := range vaxis.Characters(m.Prompt.Content) {
		chText := m.Prompt
		chText.Content = char.Grapheme
		chText.WidthHint = char.Width
		win.SetCell(col, 0, chText)
		col += char.Width
		if col >= winW {
			return
		}
	}

	chars := vaxis.Characters(m.Content.Content)
	cursor := col
	for widthToCursor(chars, m.cursor, m.offset)+col+scrolloff >= winW {
		m.offset += 1
	}
	if m.cursor-scrolloff-m.offset < 0 {
		m.offset = m.cursor - scrolloff
	}
	if m.offset < 0 {
		m.offset = 0
	}

	for i, char := range vaxis.Characters(m.Content.Content) {
		if i < m.offset {
			continue
		}
		if i+1 == m.cursor {
			cursor = col + char.Width
		}
		w := char.Width
		chText := m.Content
		chText.Content = char.Grapheme
		chText.WidthHint = w
		if m.offset > 0 && i == m.offset {
			chText.Content = "…"
		}
		if col+w >= winW {
			chText.Content = "…"
		}
		win.SetCell(col, 0, chText)
		col += w
		if col >= winW {
			break
		}
	}
	win.ShowCursor(cursor, 0, vaxis.CursorBlock)
}

// isAlphaNumeric returns true if the character is a letter or a number
func isAlphaNumeric(c vaxis.Character) bool {
	runes := []rune(c.Grapheme)
	if len(runes) > 1 {
		return false
	}
	if unicode.IsLetter(runes[0]) || unicode.IsNumber(runes[0]) {
		return true
	}
	return false
}

func widthToCursor(chars []vaxis.Character, cursor int, offset int) int {
	w := 0
	for i, ch := range chars {
		if i < offset {
			continue
		}
		w += ch.Width
		if i == cursor {
			break
		}
	}
	return w
}
