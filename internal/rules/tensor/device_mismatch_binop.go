package tensor

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/typeinfer"
)

func init() {
	v2.Register(&v2.Rule{
		ID:          "device-mismatch-binop",
		Category:    "tensor",
		Severity:    v2.SeverityError,
		Description: "Binary operator combines tensors on different devices (e.g. cpu and cuda); will raise RuntimeError at runtime.",
		NodeTypes:   []string{"function_definition", "module"},
		Confidence:  0.85,
		Check:       checkDeviceMismatchBinop,
	})
}

// checkDeviceMismatchBinop fires once per scope (function or module).
// For each scope it builds a DeviceTracker from typeinfer, then walks
// `binary_operator` nodes inside the scope (stopping at nested
// function/class/lambda boundaries so a tracker built for one scope
// does not analyze another). If both operands resolve to known
// devices that differ, the binop is flagged.
//
// Listening on both `function_definition` and `module` is intentional:
// the dispatcher fires Check once per node-type match, and each
// invocation only walks its own scope thanks to the walkScope guard,
// so a binop at module scope is processed exactly once and a binop
// inside a function is processed exactly once (by the function's run).
func checkDeviceMismatchBinop(ctx *v2.Context, scope *sitter.Node) {
	body := scopeBody(scope)
	if body == nil {
		return
	}
	tracker := typeinfer.NewDeviceTracker(body, ctx.Source)
	walkScope(body, func(n *sitter.Node) {
		if n.Type() != "binary_operator" {
			return
		}
		left := n.ChildByFieldName("left")
		right := n.ChildByFieldName("right")
		if left == nil || right == nil {
			return
		}
		ld := tracker.DeviceOf(left)
		rd := tracker.DeviceOf(right)
		if ld == typeinfer.DeviceUnknown || rd == typeinfer.DeviceUnknown || ld == rd {
			return
		}
		ctx.Emit(n, fmt.Sprintf(
			"binop combines tensor on %s with tensor on %s; runtime will raise RuntimeError",
			ld, rd))
	})
}

// scopeBody returns the body to analyze for a Check invocation. For
// function_definition we use the `body` field; for module we use the
// node itself.
func scopeBody(scope *sitter.Node) *sitter.Node {
	switch scope.Type() {
	case "function_definition":
		return scope.ChildByFieldName("body")
	case "module":
		return scope
	}
	return nil
}

// walkScope visits descendants of root in preorder, stopping at
// nested function/class/lambda boundaries. Mirrors the same-scope
// walkers in the async and server rule packages — duplicated rather
// than centralized because the rule taxonomies should not have to
// import each other.
func walkScope(root *sitter.Node, visit func(*sitter.Node)) {
	var rec func(n *sitter.Node)
	rec = func(n *sitter.Node) {
		t := n.Type()
		if t == "function_definition" || t == "class_definition" || t == "lambda" {
			return
		}
		visit(n)
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			rec(c)
		}
	}
	for i := 0; i < int(root.ChildCount()); i++ {
		c := root.Child(i)
		if c == nil {
			continue
		}
		rec(c)
	}
}
