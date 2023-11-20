package textinput

import (
	"strings"
	"unicode"

	"git.sr.ht/~rockorager/vaxis"
	"golang.org/x/exp/slices"
)

const scrolloff = 4

var truncator = vaxis.Character{
	Grapheme: "…",
	Width:    1,
}

type Model struct {
	content []vaxis.Character
	prompt  []vaxis.Character

	Content vaxis.Style
	Prompt  vaxis.Style
	// HideCursor tells the textinput not to draw the cursor
	HideCursor bool

	cursor int // the x position of the cursor, relative to the start of Content
	offset int

	paste []rune
}

func New() *Model {
	return &Model{}
}

func (m *Model) SetPrompt(s string) *Model {
	m.prompt = vaxis.Characters(s)
	return m
}

func (m *Model) SetContent(s string) *Model {
	m.content = vaxis.Characters(s)
	m.cursor = len(m.content)
	return m
}

func (m *Model) String() string {
	buf := strings.Builder{}
	for _, ch := range m.content {
		buf.WriteString(ch.Grapheme)
	}
	return buf.String()
}

func (m *Model) Update(msg vaxis.Event) {
	switch msg := msg.(type) {
	case vaxis.PasteEndEvent:
		chars := vaxis.Characters(string(m.paste))
		m.content = slices.Insert(m.content, m.cursor, chars...)
		m.cursor += len(chars)
		m.paste = []rune{}
	case vaxis.Key:
		if msg.EventType == vaxis.EventRelease {
			return
		}
		if msg.EventType == vaxis.EventPaste {
			m.paste = append(m.paste, []rune(msg.Text)...)
			return
		}
		switch msg.String() {
		case "Ctrl+a", "Home":
			// Beginning of line
			m.cursor = 0
		case "Ctrl+e", "End":
			// End of line
			m.cursor = len(m.content)
		case "Ctrl+f", "Right":
			// forward one character
			m.cursor += 1
		case "Ctrl+b", "Left":
			// backward one character
			m.cursor -= 1
		case "Alt+f", "Ctrl+Right":
			// Forward one word
			// skip non-alphanumerics
			for i := m.cursor; i < len(m.content); i += 1 {
				if !isAlphaNumeric(m.content[i]) {
					m.cursor += 1
					continue
				}
				break
			}
			for i := m.cursor; i < len(m.content); i += 1 {
				if isAlphaNumeric(m.content[i]) {
					m.cursor += 1
					continue
				}
				break
			}
		case "Alt+b", "Ctrl+Left":
			// backward one word
			// skip non-alphanumerics
			m.cursor -= 1
			if m.cursor >= len(m.content) {
				m.cursor = len(m.content) - 1
			}
			for i := m.cursor; i >= 0; i -= 1 {
				if !isAlphaNumeric(m.content[i]) {
					m.cursor -= 1
					continue
				}
				break
			}
			for i := m.cursor; i >= 0; i -= 1 {
				if isAlphaNumeric(m.content[i]) {
					m.cursor -= 1
					continue
				}
				m.cursor += 1
				break
			}
		case "Ctrl+d", "Delete":
			// delete character under cursor
			switch {
			case m.cursor == len(m.content):
				m.content = m.content[:m.cursor]
			default:
				m.content = append(m.content[:m.cursor], m.content[m.cursor+1:]...)
			}
		case "Ctrl+k":
			m.content = m.content[:m.cursor]
		case "Ctrl+u":
			m.content = m.content[m.cursor:]
			m.cursor = 0
		case "Ctrl+h", "BackSpace":
			// delete character behind cursor
			switch {
			case m.cursor == 0:
				return
			case m.cursor == len(m.content):
				m.content = m.content[:m.cursor-1]
			default:
				m.content = append(m.content[:m.cursor-1], m.content[m.cursor:]...)
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
			if msg.Text != "" {
				chars := vaxis.Characters(msg.Text)
				for _, char := range chars {
					m.content = slices.Insert(m.content, m.cursor, char)
					m.cursor += 1
				}
			}
		}
	}
	if m.cursor > len(m.content) {
		m.cursor = len(m.content)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) Draw(win vaxis.Window) {
	winW, _ := win.Size()
	if winW == 0 {
		return
	}
	win.Clear()
	col := 0
	for _, char := range m.prompt {
		cell := vaxis.Cell{
			Character: char,
			Style:     m.Prompt,
		}
		win.SetCell(col, 0, cell)
		col += char.Width
		if col >= winW {
			return
		}
	}

	chars := m.content
	cursor := col
	// Make sure we've scrolled enough to have the cursor in the view
	for widthToCursor(chars, m.cursor, m.offset)+col+scrolloff >= winW {
		m.offset += 1
	}
	// Or we need to scroll toward beginning of line
	if m.cursor-scrolloff-m.offset < 0 {
		m.offset = m.cursor - scrolloff
	}
	if m.offset < 0 {
		m.offset = 0
	}

	for i, char := range m.content {
		if i < m.offset {
			continue
		}
		if i+1 == m.cursor {
			cursor = col + char.Width
		}
		cell := vaxis.Cell{
			Character: char,
			Style:     m.Content,
		}
		if m.offset > 0 && i == m.offset {
			cell.Character = truncator
		}
		if col+char.Width >= winW {
			cell.Character = truncator
		}
		win.SetCell(col, 0, cell)
		col += char.Width
		if col >= winW {
			break
		}
	}
	if !m.HideCursor {
		win.ShowCursor(cursor, 0, vaxis.CursorBlock)
	}
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
