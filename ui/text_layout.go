package ui

type textLayoutOptions struct {
	SoftWrap bool
	Overflow TextOverflow
	MaxLines int
	Align    TextAlign
}

type laidOutText struct {
	Lines []laidOutLine
	Size  Size
}

type laidOutLine struct {
	Runs  []TextSpan
	Width int
}

type textAtom struct {
	char  Character
	style Style
}

func layoutText(spans []TextSpan, c Constraints, opts textLayoutOptions) laidOutText {
	maxWidth := c.MaxWidth
	if !opts.SoftWrap || maxWidth == Unbounded {
		maxWidth = Unbounded
	}
	var lines []laidOutLine
	line := laidOutLine{}
	word := []textAtom{}
	wordWidth := 0
	flushLine := func() {
		appendWord(&line, word)
		word = nil
		wordWidth = 0
		lines = append(lines, line)
		line = laidOutLine{}
	}
	commitLine := func() {
		lines = append(lines, line)
		line = laidOutLine{}
	}
	flushWord := func() {
		appendWord(&line, word)
		word = nil
		wordWidth = 0
	}
	for _, span := range spans {
		for _, ch := range vaxisCharacters(span.Text) {
			if ch.Grapheme == "\n" {
				flushLine()
				continue
			}
			atom := textAtom{char: ch, style: span.Style}
			if !opts.SoftWrap || maxWidth == Unbounded {
				appendAtom(&line, atom)
				continue
			}
			if isSpace(ch) {
				flushWord()
				if line.Width+ch.Width > maxWidth && line.Width > 0 {
					flushLine()
				} else if line.Width+ch.Width <= maxWidth {
					appendAtom(&line, atom)
				}
				continue
			}
			if wordWidth+ch.Width > maxWidth && wordWidth > 0 {
				if line.Width > 0 {
					flushLine()
				}
				appendWord(&line, word)
				word = nil
				wordWidth = 0
				flushLine()
			}
			word = append(word, atom)
			wordWidth += ch.Width
			if line.Width+wordWidth > maxWidth && line.Width > 0 {
				commitLine()
			}
		}
	}
	appendWord(&line, word)
	if len(lines) == 0 || line.Width > 0 || len(line.Runs) > 0 {
		lines = append(lines, line)
	}
	if opts.MaxLines > 0 && len(lines) > opts.MaxLines {
		lines = lines[:opts.MaxLines]
		if opts.Overflow == TextOverflowEllipsis && len(lines) > 0 {
			applyEllipsis(&lines[len(lines)-1], c.MaxWidth)
		}
	} else if opts.Overflow == TextOverflowEllipsis && opts.MaxLines == 1 && c.HasBoundedWidth() && len(lines) == 1 && lines[0].Width > c.MaxWidth {
		applyEllipsis(&lines[0], c.MaxWidth)
	}
	width := 0
	for _, line := range lines {
		width = max(width, line.Width)
	}
	size := c.Constrain(Size{Width: width, Height: len(lines)})
	return laidOutText{Lines: lines, Size: size}
}

func appendWord(line *laidOutLine, word []textAtom) {
	for _, atom := range word {
		appendAtom(line, atom)
	}
}

func appendAtom(line *laidOutLine, atom textAtom) {
	if len(line.Runs) == 0 || line.Runs[len(line.Runs)-1].Style != atom.style {
		line.Runs = append(line.Runs, TextSpan{Style: atom.style})
	}
	line.Runs[len(line.Runs)-1].Text += atom.char.Grapheme
	line.Width += atom.char.Width
}

func isSpace(ch Character) bool {
	return ch.Grapheme == " " || ch.Grapheme == "\t"
}

func applyEllipsis(line *laidOutLine, maxWidth int) {
	if maxWidth == Unbounded || maxWidth <= 0 {
		line.Runs = nil
		line.Width = 0
		return
	}
	atoms := lineAtoms(*line)
	width := 1
	keep := 0
	for keep < len(atoms) && width+atoms[keep].char.Width <= maxWidth {
		width += atoms[keep].char.Width
		keep++
	}
	if keep < len(atoms) {
		atoms = atoms[:keep]
	}
	style := Style{}
	if len(atoms) > 0 {
		style = atoms[len(atoms)-1].style
	} else if len(line.Runs) > 0 {
		style = line.Runs[0].Style
	}
	atoms = append(atoms, textAtom{char: Character{Grapheme: "…", Width: 1}, style: style})
	line.Runs = nil
	line.Width = 0
	appendWord(line, atoms)
}

func lineAtoms(line laidOutLine) []textAtom {
	var atoms []textAtom
	for _, run := range line.Runs {
		for _, ch := range vaxisCharacters(run.Text) {
			atoms = append(atoms, textAtom{char: ch, style: run.Style})
		}
	}
	return atoms
}

func paintLaidOutText(p *Painter, off Offset, layout laidOutText, opts textLayoutOptions) {
	for y, line := range layout.Lines {
		x := off.X + textAlignOffset(layout.Size.Width, line.Width, opts.Align)
		for _, run := range line.Runs {
			p.DrawText(Offset{X: x, Y: off.Y + y}, run.Text, run.Style)
			x += textWidth(run.Text)
		}
	}
}

func paintTextBackground(p *Painter, off Offset, size Size, spans []TextSpan) {
	style, ok := textBackgroundStyle(spans)
	if !ok || size.Width <= 0 || size.Height <= 0 {
		return
	}
	p.Fill(Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, Cell{Character: Character{Grapheme: " ", Width: 1}, Style: style})
}

func textBackgroundStyle(spans []TextSpan) (Style, bool) {
	for _, span := range spans {
		if span.Style.Background != 0 {
			return span.Style, true
		}
	}
	return Style{}, false
}

func textAlignOffset(width, lineWidth int, align TextAlign) int {
	delta := max(0, width-lineWidth)
	switch align {
	case TextAlignEnd, TextAlignRight:
		return delta
	case TextAlignCenter:
		return delta / 2
	default:
		return 0
	}
}
