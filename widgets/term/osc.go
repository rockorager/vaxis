package term

import (
	"encoding/base64"
	"fmt"
	"git.sr.ht/~rockorager/vaxis/log"
	"strings"
)

func (vt *Model) osc(data string) {
	selector, val, found := cutString(data, ";")
	if !found {
		return
	}
	switch selector {
	case "0", "2":
		vt.postEvent(EventTitle(val))
	case "8":
		if vt.OSC8 {
			params, url, found := cutString(val, ";")
			if !found {
				return
			}
			vt.cursor.Hyperlink = url
			vt.cursor.HyperlinkParams = params
		}
	case "9":
		vt.postEvent(EventNotify{Body: val})
	case "11":
		if val == "?" {
			if vt.vx == nil {
				return
			}
			rgb := vt.vx.QueryBackground().Params()
			if len(rgb) == 0 {
				return
			}
			resp := fmt.Sprintf("\x1b]11;rgb:%02x/%02x/%02x\x07", rgb[0], rgb[1], rgb[2])
			vt.pty.WriteString(resp)
		}
	case "52":
		_, val, _ := cutString(val, ";")
		decodedBytes, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			log.Error("[term] error decoding Base64")
			return
		}
		vt.vx.ClipboardPush(string(decodedBytes))
	case "777":
		selector, val, found := cutString(val, ";")
		if !found {
			return
		}
		switch selector {
		case "notify":
			title, body, found := cutString(val, ";")
			if !found {
				return
			}
			vt.postEvent(EventNotify{
				Title: title,
				Body:  body,
			})
		}
	}
}

// parses an osc8 payload into the URL and optional ID
func osc8(val string) (string, string) {
	// OSC 8 ; params ; url ST
	// params: key1=value1:key2=value2
	var id string
	params, url, found := cutString(val, ";")
	if !found {
		return "", ""
	}
	for _, param := range strings.Split(params, ":") {
		key, val, found := cutString(param, "=")
		if !found {
			continue
		}
		switch key {
		case "id":
			id = val
		}
	}
	return url, id
}

// Copied from stdlib to here for go 1.16 compat
func cutString(s string, sep string) (before string, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
