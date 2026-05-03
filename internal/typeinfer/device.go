package typeinfer

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Device is the source-resolvable host/accelerator device for a tensor
// expression. DeviceUnknown means the tracker could not derive the
// device structurally; rules treat that the same as "no information".
type Device int

const (
	DeviceUnknown Device = iota
	DeviceCPU
	DeviceCUDA
	DeviceMPS
)

// String renders d for finding messages.
func (d Device) String() string {
	switch d {
	case DeviceCPU:
		return "cpu"
	case DeviceCUDA:
		return "cuda"
	case DeviceMPS:
		return "mps"
	default:
		return "unknown"
	}
}

// DeviceTracker resolves the device of expressions inside a single
// scope (typically a function body or a module). It records local
// assignments where the right-hand side resolves to a known device:
//
//	a = x.cpu()         -> a: cpu
//	b = x.cuda()        -> b: cuda
//	c = a.to("cuda:0")  -> c: cuda
//
// The walk stops at nested function/class/lambda boundaries — bindings
// in inner scopes do not leak. Last-write wins; control-flow merging
// (if/else, try) is intentionally not modeled at MVP. Rules that need
// runtime-resolved devices use the PyOracle instead.
type DeviceTracker struct {
	bindings map[string]Device
	src      []byte
}

// NewDeviceTracker builds and populates a tracker over scope. scope is
// typically a `block` (function body) or `module` node. src is the
// full source slice for the file.
func NewDeviceTracker(scope *sitter.Node, src []byte) *DeviceTracker {
	t := &DeviceTracker{
		bindings: map[string]Device{},
		src:      src,
	}
	if scope != nil {
		t.populate(scope)
	}
	return t
}

// DeviceOf resolves the device of a single expression. Recursive: it
// handles identifiers, host/device-copy calls, .to(<literal>), and
// binary operators where both operands agree on a device.
func (t *DeviceTracker) DeviceOf(expr *sitter.Node) Device {
	if expr == nil {
		return DeviceUnknown
	}
	switch expr.Type() {
	case "identifier":
		return t.bindings[string(t.src[expr.StartByte():expr.EndByte()])]
	case "call":
		return t.deviceOfCall(expr)
	case "binary_operator":
		left := expr.ChildByFieldName("left")
		right := expr.ChildByFieldName("right")
		ld := t.DeviceOf(left)
		rd := t.DeviceOf(right)
		if ld != DeviceUnknown && ld == rd {
			return ld
		}
		return DeviceUnknown
	case "parenthesized_expression":
		// One named child: the inner expression.
		if inner := expr.NamedChild(0); inner != nil {
			return t.DeviceOf(inner)
		}
	}
	return DeviceUnknown
}

func (t *DeviceTracker) populate(scope *sitter.Node) {
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "function_definition", "class_definition", "lambda":
			return
		case "assignment":
			t.recordAssignment(n)
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			walk(c)
		}
	}
	for i := 0; i < int(scope.ChildCount()); i++ {
		c := scope.Child(i)
		if c == nil {
			continue
		}
		walk(c)
	}
}

func (t *DeviceTracker) recordAssignment(n *sitter.Node) {
	left := n.ChildByFieldName("left")
	right := n.ChildByFieldName("right")
	if left == nil || right == nil || left.Type() != "identifier" {
		return
	}
	dev := t.DeviceOf(right)
	if dev == DeviceUnknown {
		return
	}
	t.bindings[string(t.src[left.StartByte():left.EndByte()])] = dev
}

func (t *DeviceTracker) deviceOfCall(call *sitter.Node) Device {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return DeviceUnknown
	}
	leaf := fnExpr.ChildByFieldName("attribute")
	if leaf == nil {
		return DeviceUnknown
	}
	switch string(t.src[leaf.StartByte():leaf.EndByte()]) {
	case "cpu":
		return DeviceCPU
	case "cuda":
		return DeviceCUDA
	case "to":
		return t.deviceFromToArg(call)
	}
	return DeviceUnknown
}

func (t *DeviceTracker) deviceFromToArg(call *sitter.Node) Device {
	args := call.ChildByFieldName("arguments")
	if args == nil {
		return DeviceUnknown
	}
	first := args.NamedChild(0)
	if first == nil || first.Type() != "string" {
		return DeviceUnknown
	}
	raw := string(t.src[first.StartByte():first.EndByte()])
	text := strings.Trim(raw, `"'`)
	if idx := strings.Index(text, ":"); idx >= 0 {
		text = text[:idx]
	}
	switch text {
	case "cpu":
		return DeviceCPU
	case "cuda":
		return DeviceCUDA
	case "mps":
		return DeviceMPS
	}
	return DeviceUnknown
}
