package ui

import (
	"fmt"
	"strconv"
	"strings"

	"go.rockorager.dev/vaxis"
)

type debugActionRequest struct {
	Action    string   `json:"action"`
	ID        string   `json:"id,omitempty"`
	Label     string   `json:"label,omitempty"`
	Key       string   `json:"key,omitempty"`
	Text      string   `json:"text,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
	EventType string   `json:"eventType,omitempty"`
}

type debugActionsRequest struct {
	Actions []debugActionRequest `json:"actions"`
}

func (a *App) debugPerformAction(request debugActionRequest, submitEvent func(Event)) error {
	switch strings.ToLower(strings.TrimSpace(request.Action)) {
	case "focus":
		target, err := a.debugFocusTarget(request.ID, request.Label)
		if err != nil {
			return err
		}
		a.setFocused(target)
		return nil
	case "activate":
		target, err := a.debugFocusTarget(request.ID, request.Label)
		if err != nil {
			return err
		}
		a.setFocused(target)
		submitEvent(Key{Keycode: vaxis.KeyEnter})
		return nil
	case "click":
		pt, err := a.debugClickPoint(request.ID, request.Label)
		if err != nil {
			return err
		}
		submitEvent(Mouse{Col: pt.X, Row: pt.Y, Button: MouseLeftButton, EventType: EventPress})
		submitEvent(Mouse{Col: pt.X, Row: pt.Y, Button: MouseLeftButton, EventType: EventRelease})
		return nil
	case "key":
		ev, err := (debugEventRequest{
			Type:      "key",
			Key:       request.Key,
			Text:      request.Text,
			Modifiers: request.Modifiers,
			EventType: request.EventType,
		}).keyEvent()
		if err != nil {
			return err
		}
		submitEvent(ev)
		return nil
	default:
		return fmt.Errorf("unknown debug action %q", request.Action)
	}
}

func (a *App) debugFocusTarget(id, label string) (focusTarget, error) {
	var matches []focusTarget
	elementID := debugElementIDFromTargetID(id)
	for _, target := range a.focusables {
		targetElementID := debugElementID(target.element)
		if id != "" && (debugFocusTargetID(targetElementID, target.index) == id || targetElementID == elementID) {
			matches = append(matches, target)
			continue
		}
		if label != "" && a.debugFocusLabel(target.element, target.index) == label {
			matches = append(matches, target)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return focusTarget{}, fmt.Errorf("debug target is ambiguous")
	}
	return focusTarget{}, fmt.Errorf("debug target not found")
}

func (a *App) debugClickPoint(id, label string) (Point, error) {
	var target element
	if id != "" {
		target = a.debugElementByID(debugElementIDFromTargetID(id))
	}
	if target == nil && label != "" {
		focusTarget, err := a.debugFocusTarget("", label)
		if err != nil {
			return Point{}, err
		}
		target = focusTarget.element
	}
	if target == nil {
		return Point{}, fmt.Errorf("debug target not found")
	}
	rect, ok := a.debugElementRect(target)
	if !ok || rect.Width <= 0 || rect.Height <= 0 {
		return Point{}, fmt.Errorf("debug target has no rendered bounds")
	}
	return Point{X: rect.X + rect.Width/2, Y: rect.Y + rect.Height/2}, nil
}

func debugElementIDFromTargetID(id string) string {
	id = strings.TrimSpace(id)
	if hash := strings.IndexByte(id, '#'); hash >= 0 {
		return id[:hash]
	}
	return id
}

func (a *App) debugElementByID(id string) element {
	if a.build == nil || a.build.root == nil || id == "" {
		return nil
	}
	var out element
	var walk func(element, string)
	walk = func(e element, cur string) {
		if out != nil {
			return
		}
		if cur == id {
			out = e
			return
		}
		for i, child := range elementChildren(e) {
			walk(child, cur+"."+strconv.Itoa(i))
		}
	}
	walk(a.build.root, "0")
	return out
}

func (a *App) debugElementRect(target element) (Rect, bool) {
	if a.build == nil || a.build.root == nil || target == nil {
		return Rect{}, false
	}
	var rect Rect
	var ok bool
	var walk func(element, Offset)
	walk = func(e element, off Offset) {
		if ok {
			return
		}
		if e == target {
			rect, ok = debugFirstRenderRect(e, off)
			return
		}
		ro := ownRenderObject(e)
		for _, child := range elementChildren(e) {
			childOff := off
			if ro != nil {
				if op, hasOffset := ro.(ChildOffsetProvider); hasOffset {
					if childRO := findRenderObject(child); childRO != nil {
						delta := op.ChildOffset(childRO)
						childOff.X += delta.X
						childOff.Y += delta.Y
					}
				}
			}
			walk(child, childOff)
		}
	}
	walk(a.build.root, Offset{})
	return rect, ok
}

func debugFirstRenderRect(e element, off Offset) (Rect, bool) {
	if ro := ownRenderObject(e); ro != nil {
		size := ro.Base().Size()
		return Rect{X: off.X, Y: off.Y, Width: size.Width, Height: size.Height}, true
	}
	for _, child := range elementChildren(e) {
		childOff := off
		if ro := ownRenderObject(e); ro != nil {
			if op, ok := ro.(ChildOffsetProvider); ok {
				if childRO := findRenderObject(child); childRO != nil {
					delta := op.ChildOffset(childRO)
					childOff.X += delta.X
					childOff.Y += delta.Y
				}
			}
		}
		if rect, ok := debugFirstRenderRect(child, childOff); ok {
			return rect, true
		}
	}
	return Rect{}, false
}
