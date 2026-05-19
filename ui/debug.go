package ui

import (
	"reflect"
	"strconv"
	"strings"
)

// DebugSnapshot describes the current UI tree for development tooling.
type DebugSnapshot struct {
	Size       DebugSize          `json:"size"`
	Window     DebugWindow        `json:"window"`
	MouseShape MouseShape         `json:"mouseShape"`
	Focused    string             `json:"focused,omitempty"`
	Focusables []DebugFocusTarget `json:"focusables,omitempty"`
	Tree       *DebugNode         `json:"tree,omitempty"`
}

// DebugNode describes one mounted element in the UI tree.
type DebugNode struct {
	ID           string             `json:"id"`
	Widget       string             `json:"widget"`
	Element      string             `json:"element"`
	State        string             `json:"state,omitempty"`
	Render       string             `json:"render,omitempty"`
	Size         *DebugSize         `json:"size,omitempty"`
	Offset       *DebugOffset       `json:"offset,omitempty"`
	Dirty        bool               `json:"dirty,omitempty"`
	NeedsLayout  bool               `json:"needsLayout,omitempty"`
	NeedsPaint   bool               `json:"needsPaint,omitempty"`
	ParentData   string             `json:"parentData,omitempty"`
	FocusTargets []DebugFocusTarget `json:"focusTargets,omitempty"`
	Children     []DebugNode        `json:"children,omitempty"`
}

// DebugFocusTarget describes one keyboard focus stop.
type DebugFocusTarget struct {
	ID      string `json:"id"`
	Index   int    `json:"index"`
	Label   string `json:"label,omitempty"`
	Focused bool   `json:"focused,omitempty"`
}

// DebugSize is a JSON-friendly terminal cell size.
type DebugSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DebugOffset is a JSON-friendly terminal cell offset.
type DebugOffset struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// DebugWindow is a JSON-friendly terminal window size.
type DebugWindow struct {
	Cols   int `json:"cols"`
	Rows   int `json:"rows"`
	XPixel int `json:"xPixel,omitempty"`
	YPixel int `json:"yPixel,omitempty"`
}

type debugFocusTargetProvider interface {
	DebugFocusTargets() []DebugFocusTarget
}

// DebugSnapshot returns a development snapshot of the mounted widget/render tree.
func (a *App) DebugSnapshot() DebugSnapshot {
	snapshot := DebugSnapshot{
		Size:       debugSize(a.size),
		Window:     debugWindow(a.window),
		MouseShape: a.MouseShape(),
	}
	if a.focused.element != nil {
		snapshot.Focused = debugFocusTargetID(debugElementID(a.focused.element), a.focused.index)
	}
	if a.build == nil || a.build.Root() == nil {
		return snapshot
	}
	focusables := make([]DebugFocusTarget, 0, len(a.focusables))
	tree := a.debugNode(a.build.Root(), "0", Offset{}, &focusables)
	snapshot.Tree = &tree
	snapshot.Focusables = focusables
	return snapshot
}

func (a *App) debugNode(e element, id string, offset Offset, focusables *[]DebugFocusTarget) DebugNode {
	node := DebugNode{
		ID:      id,
		Widget:  typeName(e.Base().widget),
		Element: typeName(e),
		Dirty:   e.Base().dirty,
	}
	if stateful, ok := e.(*statefulElement); ok && stateful.state != nil {
		node.State = typeName(stateful.state)
	}
	ro := ownRenderObject(e)
	if ro != nil {
		size := debugSize(ro.Base().Size())
		node.Render = typeName(ro)
		node.Size = &size
		debugOffset := DebugOffset(offset)
		node.Offset = &debugOffset
		node.NeedsLayout = ro.Base().NeedsLayout()
		node.NeedsPaint = ro.Base().NeedsPaint()
		if pd := ro.Base().ParentData(); pd != nil {
			node.ParentData = typeName(pd)
		}
	}
	node.FocusTargets = a.debugFocusTargets(e, id)
	*focusables = append(*focusables, node.FocusTargets...)
	children := elementChildren(e)
	node.Children = make([]DebugNode, 0, len(children))
	for i, child := range children {
		childOffset := offset
		if ro != nil {
			if op, ok := ro.(ChildOffsetProvider); ok {
				if childRO := findRenderObject(child); childRO != nil {
					off := op.ChildOffset(childRO)
					childOffset.X += off.X
					childOffset.Y += off.Y
				}
			}
		}
		node.Children = append(node.Children, a.debugNode(child, id+"."+strconv.Itoa(i), childOffset, focusables))
	}
	return node
}

func (a *App) debugFocusTargets(e element, id string) []DebugFocusTarget {
	var out []DebugFocusTarget
	for _, target := range a.focusables {
		if target.element != e {
			continue
		}
		out = append(out, DebugFocusTarget{
			ID:      debugFocusTargetID(id, target.index),
			Index:   target.index,
			Label:   a.debugFocusLabel(e, target.index),
			Focused: target == a.focused,
		})
	}
	return out
}

func (a *App) debugFocusLabel(e element, index int) string {
	if index >= 0 {
		if provider, ok := ownRenderObject(e).(debugFocusTargetProvider); ok {
			for _, target := range provider.DebugFocusTargets() {
				if target.Index == index {
					return target.Label
				}
			}
		}
		return ""
	}
	return debugElementLabel(e)
}

func debugFocusTargetID(elementID string, index int) string {
	if index == elementFocusIndex {
		return elementID + "#focus"
	}
	return elementID + "#focus:" + strconv.Itoa(index)
}

func debugElementID(e element) string {
	if e == nil || e.Base().owner == nil || e.Base().owner.root == nil {
		return ""
	}
	var out string
	var walk func(element, string) bool
	walk = func(cur element, id string) bool {
		if cur == e {
			out = id
			return true
		}
		children := elementChildren(cur)
		for i, child := range children {
			if walk(child, id+"."+strconv.Itoa(i)) {
				return true
			}
		}
		return false
	}
	walk(e.Base().owner.root, "0")
	return out
}

func debugElementLabel(e element) string {
	if label, ok := debugWidgetLabel(e.Base().widget); ok {
		return label
	}
	var out string
	e.VisitChildren(func(child element) {
		if out == "" {
			out = debugElementLabel(child)
		}
	})
	return out
}

func debugWidgetLabel(widget any) (string, bool) {
	switch w := widget.(type) {
	case Text:
		return w.Value, w.Value != ""
	case RichText:
		var b strings.Builder
		for _, span := range w.Spans {
			b.WriteString(span.Text)
		}
		label := strings.TrimSpace(b.String())
		return label, label != ""
	}
	v := reflect.ValueOf(widget)
	if !v.IsValid() {
		return "", false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	for _, name := range []string{"Label", "Value"} {
		field := v.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.String && field.String() != "" {
			return field.String(), true
		}
	}
	return "", false
}

func debugSize(size Size) DebugSize {
	return DebugSize(size)
}

func debugWindow(size Resize) DebugWindow {
	return DebugWindow{Cols: size.Cols, Rows: size.Rows, XPixel: size.XPixel, YPixel: size.YPixel}
}

func typeName(v any) string {
	if v == nil {
		return ""
	}
	return reflect.TypeOf(v).String()
}
