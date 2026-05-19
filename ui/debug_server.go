package ui

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

func startDebugServer(app *App, dispatch func(func()), submitEvent func(Event), rendered func() (DebugRenderedSnapshot, bool), renderedText func() (string, bool)) (func(), error) {
	token, ok := debugServerToken()
	if !ok {
		return func() {}, nil
	}
	addr := os.Getenv("VAXIS_UI_DEBUG_ADDR")
	if addr == "" {
		addr = "127.0.0.1:2113"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: newDebugServerHandler(app, token, dispatch, submitEvent, rendered, renderedText), ReadHeaderTimeout: 2 * time.Second}
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "vaxis ui debug server error: %v\n", err)
		}
	}()
	fmt.Fprintf(os.Stderr, "vaxis ui debug listening on http://%s/debug/ui\n", ln.Addr())
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}, nil
}

func newDebugServerHandler(app *App, token string, dispatch func(func()), submitEvent func(Event), rendered func() (DebugRenderedSnapshot, bool), renderedText func() (string, bool)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/ui", func(w http.ResponseWriter, r *http.Request) {
		if !debugAuthenticated(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snapshot, ok := debugSnapshotViaDispatch(r.Context(), app, dispatch)
		if !ok {
			http.Error(w, "timed out waiting for UI debug snapshot", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(snapshot)
	})
	mux.HandleFunc("/debug/ui/rendered", func(w http.ResponseWriter, r *http.Request) {
		if !debugAuthenticated(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snapshot, ok := debugRenderedViaDispatch(r.Context(), dispatch, rendered)
		if !ok {
			http.Error(w, "rendered frame is unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(snapshot)
	})
	mux.HandleFunc("/debug/ui/rendered.txt", func(w http.ResponseWriter, r *http.Request) {
		if !debugAuthenticated(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		text, ok := debugRenderedTextViaDispatch(r.Context(), dispatch, renderedText)
		if !ok {
			http.Error(w, "rendered frame is unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(text))
	})
	mux.HandleFunc("/debug/ui/events", func(w http.ResponseWriter, r *http.Request) {
		if !debugAuthenticated(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		events, err := parseDebugEvents(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if submitEvent == nil {
			http.Error(w, "debug event submission is unavailable", http.StatusServiceUnavailable)
			return
		}
		for _, ev := range events {
			if !debugSubmitEventViaDispatch(r.Context(), dispatch, submitEvent, ev) {
				http.Error(w, "timed out waiting for UI debug event", http.StatusServiceUnavailable)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"submitted": len(events)})
	})
	return mux
}

func debugServerToken() (string, bool) {
	token := os.Getenv("VAXIS_UI_DEBUG")
	switch token {
	case "", "0", "false", "FALSE":
		return "", false
	default:
		return token, true
	}
}

func debugAuthenticated(r *http.Request, token string) bool {
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(got), "bearer ") {
		got = strings.TrimSpace(got[len("bearer "):])
	} else {
		got = strings.TrimSpace(r.Header.Get("X-Vaxis-UI-Debug"))
	}
	return token != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

func debugSnapshotViaDispatch(ctx context.Context, app *App, dispatch func(func())) (DebugSnapshot, bool) {
	done := make(chan DebugSnapshot, 1)
	dispatch(func() {
		done <- app.DebugSnapshot()
	})
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case snapshot := <-done:
		return snapshot, true
	case <-ctx.Done():
		return DebugSnapshot{}, false
	case <-timer.C:
		return DebugSnapshot{}, false
	}
}

func debugSubmitEventViaDispatch(ctx context.Context, dispatch func(func()), submitEvent func(Event), ev Event) bool {
	done := make(chan struct{}, 1)
	dispatch(func() {
		submitEvent(ev)
		done <- struct{}{}
	})
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	case <-timer.C:
		return false
	}
}

func debugRenderedViaDispatch(ctx context.Context, dispatch func(func()), rendered func() (DebugRenderedSnapshot, bool)) (DebugRenderedSnapshot, bool) {
	if rendered == nil {
		return DebugRenderedSnapshot{}, false
	}
	done := make(chan DebugRenderedSnapshot, 1)
	var ok bool
	dispatch(func() {
		var snapshot DebugRenderedSnapshot
		snapshot, ok = rendered()
		done <- snapshot
	})
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case snapshot := <-done:
		return snapshot, ok
	case <-ctx.Done():
		return DebugRenderedSnapshot{}, false
	case <-timer.C:
		return DebugRenderedSnapshot{}, false
	}
}

func debugRenderedTextViaDispatch(ctx context.Context, dispatch func(func()), renderedText func() (string, bool)) (string, bool) {
	if renderedText == nil {
		return "", false
	}
	done := make(chan string, 1)
	var ok bool
	dispatch(func() {
		var text string
		text, ok = renderedText()
		done <- text
	})
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case text := <-done:
		return text, ok
	case <-ctx.Done():
		return "", false
	case <-timer.C:
		return "", false
	}
}

type debugEventsRequest struct {
	Events []debugEventRequest `json:"events"`
}

type debugEventRequest struct {
	Type         string   `json:"type"`
	Key          string   `json:"key,omitempty"`
	Text         string   `json:"text,omitempty"`
	Modifiers    []string `json:"modifiers,omitempty"`
	EventType    string   `json:"eventType,omitempty"`
	Col          int      `json:"col,omitempty"`
	Row          int      `json:"row,omitempty"`
	XPixel       int      `json:"xPixel,omitempty"`
	YPixel       int      `json:"yPixel,omitempty"`
	Button       string   `json:"button,omitempty"`
	Cols         int      `json:"cols,omitempty"`
	Rows         int      `json:"rows,omitempty"`
	XPixelWindow int      `json:"xPixelWindow,omitempty"`
	YPixelWindow int      `json:"yPixelWindow,omitempty"`
	Width        int      `json:"width,omitempty"`
	Height       int      `json:"height,omitempty"`
	WidthPixels  int      `json:"widthPixels,omitempty"`
	HeightPixels int      `json:"heightPixels,omitempty"`
}

func parseDebugEvents(r *http.Request) ([]Event, error) {
	defer func() { _ = r.Body.Close() }()
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var requests []debugEventRequest
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal(raw, &requests); err != nil {
			return nil, err
		}
	} else {
		var wrapper debugEventsRequest
		if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Events != nil {
			requests = wrapper.Events
		} else {
			var request debugEventRequest
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			requests = []debugEventRequest{request}
		}
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("missing events")
	}
	events := make([]Event, 0, len(requests))
	for _, request := range requests {
		ev, err := request.event()
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r debugEventRequest) event() (Event, error) {
	switch strings.ToLower(r.Type) {
	case "key":
		return r.keyEvent()
	case "mouse":
		return r.mouseEvent()
	case "resize":
		cols, rows := r.Cols, r.Rows
		if cols == 0 {
			cols = r.Width
		}
		if rows == 0 {
			rows = r.Height
		}
		xPixel, yPixel := r.XPixelWindow, r.YPixelWindow
		if xPixel == 0 {
			xPixel = r.WidthPixels
		}
		if yPixel == 0 {
			yPixel = r.HeightPixels
		}
		return Resize{Cols: cols, Rows: rows, XPixel: xPixel, YPixel: yPixel}, nil
	case "redraw":
		return Redraw{}, nil
	case "focusin":
		return FocusIn{}, nil
	case "focusout":
		return FocusOut{}, nil
	default:
		return nil, fmt.Errorf("unknown debug event type %q", r.Type)
	}
}

func (r debugEventRequest) keyEvent() (Event, error) {
	keySpec := r.Key
	if keySpec == "" && len([]rune(r.Text)) == 1 {
		keySpec = r.Text
	}
	keycode, modifiers, err := parseDebugKey(keySpec)
	if err != nil {
		return nil, err
	}
	extraModifiers, err := parseDebugModifiers(r.Modifiers)
	if err != nil {
		return nil, err
	}
	modifiers |= extraModifiers
	eventType, err := parseDebugEventType(r.EventType, vaxis.EventPress)
	if err != nil {
		return nil, err
	}
	text := r.Text
	if text == "" && keycode != 0 && modifiers&(vaxis.ModCtrl|vaxis.ModAlt|vaxis.ModSuper|vaxis.ModMeta|vaxis.ModHyper) == 0 && eventType == vaxis.EventPress {
		if len(r.Key) == 1 {
			text = r.Key
		}
	}
	return Key{Text: text, Keycode: keycode, Modifiers: modifiers, EventType: eventType}, nil
}

func (r debugEventRequest) mouseEvent() (Event, error) {
	eventType, err := parseDebugEventType(r.EventType, vaxis.EventPress)
	if err != nil {
		return nil, err
	}
	button, err := parseDebugMouseButton(r.Button)
	if err != nil {
		return nil, err
	}
	modifiers, err := parseDebugModifiers(r.Modifiers)
	if err != nil {
		return nil, err
	}
	return Mouse{
		Button:    button,
		Col:       r.Col,
		Row:       r.Row,
		XPixel:    r.XPixel,
		YPixel:    r.YPixel,
		EventType: eventType,
		Modifiers: modifiers,
	}, nil
}

func parseDebugKey(spec string) (rune, vaxis.ModifierMask, error) {
	if spec == "" {
		return 0, 0, fmt.Errorf("key event requires key")
	}
	parts := strings.Split(spec, "+")
	modifiers, err := parseDebugModifiers(parts[:len(parts)-1])
	if err != nil {
		return 0, 0, err
	}
	key := parts[len(parts)-1]
	if len([]rune(key)) == 1 {
		r := []rune(key)[0]
		if modifiers&vaxis.ModShift != 0 && 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r, modifiers, nil
	}
	if code, err := strconv.Atoi(key); err == nil {
		return rune(code), modifiers, nil
	}
	if len(key) > 1 && (key[0] == 'f' || key[0] == 'F') {
		if n, err := strconv.Atoi(key[1:]); err == nil && n >= 0 && n <= 63 {
			return vaxis.KeyF00 + rune(n), modifiers, nil
		}
	}
	if code, ok := debugKeyNames[strings.ToLower(strings.ReplaceAll(key, "_", ""))]; ok {
		return code, modifiers, nil
	}
	return 0, 0, fmt.Errorf("unknown debug key %q", key)
}

func parseDebugModifiers(names []string) (vaxis.ModifierMask, error) {
	var out vaxis.ModifierMask
	for _, name := range names {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "":
		case "shift":
			out |= vaxis.ModShift
		case "alt", "option":
			out |= vaxis.ModAlt
		case "ctrl", "control":
			out |= vaxis.ModCtrl
		case "super", "cmd", "command":
			out |= vaxis.ModSuper
		case "hyper":
			out |= vaxis.ModHyper
		case "meta":
			out |= vaxis.ModMeta
		case "caps", "capslock":
			out |= vaxis.ModCapsLock
		case "num", "numlock":
			out |= vaxis.ModNumLock
		default:
			return 0, fmt.Errorf("unknown debug modifier %q", name)
		}
	}
	return out, nil
}

func parseDebugEventType(name string, fallback vaxis.EventType) (vaxis.EventType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "":
		return fallback, nil
	case "press", "down":
		return vaxis.EventPress, nil
	case "repeat":
		return vaxis.EventRepeat, nil
	case "release", "up":
		return vaxis.EventRelease, nil
	case "motion", "move", "drag":
		return vaxis.EventMotion, nil
	case "paste":
		return vaxis.EventPaste, nil
	default:
		return 0, fmt.Errorf("unknown debug event type %q", name)
	}
}

func parseDebugMouseButton(name string) (MouseButton, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "left":
		return MouseLeftButton, nil
	case "middle":
		return MouseMiddleButton, nil
	case "right":
		return MouseRightButton, nil
	case "none":
		return MouseNoButton, nil
	case "wheelup":
		return MouseWheelUp, nil
	case "wheeldown":
		return MouseWheelDown, nil
	case "wheelleft":
		return vaxis.MouseWheelLeft, nil
	case "wheelright":
		return vaxis.MouseWheelRight, nil
	default:
		if code, err := strconv.Atoi(name); err == nil {
			return MouseButton(code), nil
		}
		return MouseNoButton, fmt.Errorf("unknown debug mouse button %q", name)
	}
}

var debugKeyNames = map[string]rune{
	"up":        vaxis.KeyUp,
	"right":     vaxis.KeyRight,
	"down":      vaxis.KeyDown,
	"left":      vaxis.KeyLeft,
	"insert":    vaxis.KeyInsert,
	"delete":    vaxis.KeyDelete,
	"backspace": vaxis.KeyBackspace,
	"pagedown":  vaxis.KeyPgDown,
	"pageup":    vaxis.KeyPgUp,
	"pgdown":    vaxis.KeyPgDown,
	"pgup":      vaxis.KeyPgUp,
	"home":      vaxis.KeyHome,
	"end":       vaxis.KeyEnd,
	"enter":     vaxis.KeyEnter,
	"return":    vaxis.KeyEnter,
	"tab":       vaxis.KeyTab,
	"esc":       vaxis.KeyEsc,
	"escape":    vaxis.KeyEsc,
	"space":     vaxis.KeySpace,
}
