package term

import (
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
