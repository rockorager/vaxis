package rtk

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type terminfo struct {
	Names    []string
	Bools    map[string]bool
	Numerics map[string]int
	Strings  map[string]string
}

func infocmp(name string) (*terminfo, error) {
	cmd := exec.Command("infocmp", "-1", "-x", name)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(r)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	defer cmd.Wait()
	ti := &terminfo{
		Bools:    make(map[string]bool),
		Numerics: make(map[string]int),
		Strings:  make(map[string]string),
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), ",")
		switch {
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "\t"):
			line = strings.TrimSpace(line)
			if key, val, found := strings.Cut(line, "#"); found {
				// int
				i, err := strconv.ParseUint(val, 0, 0)
				if err != nil {
					return nil, err
				}
				ti.Numerics[key] = int(i)
				continue
			}
			if key, val, found := strings.Cut(line, "="); found {
				// string
				ti.Strings[key] = unescape(val)
				continue
			}
			ti.Bools[line] = true
		default:
			ti.Names = strings.Split(line, "|")
		}
	}
	return ti, nil
}

// Everything below is copied directly from github.com/gdamore/tcell/v2/terminfo

func unescape(s string) string {
	const (
		NONE = iota
		CTRL
		ESC_KEY
	)
	// Various escapes are in \x format.  Control codes are
	// encoded as ^M (carat followed by ASCII equivalent).
	// Escapes are: \e, \E - escape
	//  \0 NULL, \n \l \r \t \b \f \s for equivalent C escape.
	buf := &bytes.Buffer{}
	esc := NONE

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch esc {
		case NONE:
			switch c {
			case '\\':
				esc = ESC_KEY
			case '^':
				esc = CTRL
			default:
				buf.WriteByte(c)
			}
		case CTRL:
			buf.WriteByte(c ^ 1<<6)
			esc = NONE
		case ESC_KEY:
			switch c {
			case 'E', 'e':
				buf.WriteByte(0x1b)
			case '0', '1', '2', '3', '4', '5', '6', '7':
				if i+2 < len(s) && s[i+1] >= '0' && s[i+1] <= '7' && s[i+2] >= '0' && s[i+2] <= '7' {
					buf.WriteByte(((c - '0') * 64) + ((s[i+1] - '0') * 8) + (s[i+2] - '0'))
					i = i + 2
				} else if c == '0' {
					buf.WriteByte(0)
				}
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case 's':
				buf.WriteByte(' ')
			case 'l':
				panic("WTF: weird format: " + s)
			default:
				buf.WriteByte(c)
			}
			esc = NONE
		}
	}
	return (buf.String())
}
