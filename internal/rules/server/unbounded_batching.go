package server

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

func init() {
	v2.Register(&v2.Rule{
		ID:          "unbounded-batching",
		Category:    "server",
		Severity:    v2.SeverityWarning,
		Description: "Async function appends to a `self.<name>` accumulator but never flushes, clears, or measures it; the buffer grows without bound.",
		NodeTypes:   []string{"function_definition"},
		Confidence:  0.5,
		Check:       checkUnboundedBatching,
	})
}

// checkUnboundedBatching flags `self.<name>.append(...)` (or .extend)
// calls inside an async-def body when the same function never bounds
// the accumulator. Bounding signals: a `clear()`, `flush()`, `pop(...)`
// call on the same receiver; a `len(self.<name>)` reference; or a
// reassignment `self.<name> = ...`. Local-list accumulators are out of
// scope — they are scope-bounded and freed when the function returns.
//
// Confidence is 0.5: detection is purely structural and there are
// legitimate accumulator patterns (e.g. background drain task) that
// will need `# pyhotlint: ignore[unbounded-batching]`.
func checkUnboundedBatching(ctx *v2.Context, fn *sitter.Node) {
	if !isAsyncFn(fn) {
		return
	}
	body := fn.ChildByFieldName("body")
	if body == nil {
		return
	}
	bodyText := string(ctx.Source[body.StartByte():body.EndByte()])
	walkInScope(body, func(n *sitter.Node) {
		if n.Type() != "call" {
			return
		}
		recv, leaf, ok := selfAttrCallReceiver(n, ctx.Source)
		if !ok || (leaf != "append" && leaf != "extend") {
			return
		}
		if accumulatorIsBounded(bodyText, recv) {
			return
		}
		ctx.Emit(n, "accumulator "+recv+" appended without a corresponding clear/flush/pop or len() check; potential unbounded growth")
	})
}

// isAsyncFn mirrors the async helper but lives in the server package
// to avoid cross-package coupling between rule taxonomies.
func isAsyncFn(fn *sitter.Node) bool {
	for i := 0; i < int(fn.ChildCount()); i++ {
		c := fn.Child(i)
		if c == nil {
			continue
		}
		if c.Type() == "async" {
			return true
		}
		if c.Type() == "def" {
			return false
		}
	}
	return false
}

// walkInScope visits descendants without crossing nested
// function/class/lambda boundaries.
func walkInScope(root *sitter.Node, visit func(*sitter.Node)) {
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

// selfAttrCallReceiver decomposes a call like `self.NAME.append(arg)`
// into ("self.NAME", "append", true). Returns false if the call is not
// of that shape.
func selfAttrCallReceiver(call *sitter.Node, src []byte) (recv, leaf string, ok bool) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return "", "", false
	}
	leafNode := fnExpr.ChildByFieldName("attribute")
	objNode := fnExpr.ChildByFieldName("object")
	if leafNode == nil || objNode == nil {
		return "", "", false
	}
	if objNode.Type() != "attribute" {
		return "", "", false
	}
	innerObj := objNode.ChildByFieldName("object")
	if innerObj == nil || innerObj.Type() != "identifier" {
		return "", "", false
	}
	if string(src[innerObj.StartByte():innerObj.EndByte()]) != "self" {
		return "", "", false
	}
	return strings.TrimSpace(string(src[objNode.StartByte():objNode.EndByte()])),
		string(src[leafNode.StartByte():leafNode.EndByte()]),
		true
}

// accumulatorIsBounded reports whether bodyText contains any of the
// patterns that bound `recv` (e.g. "self.batch.clear", "self.batch =",
// "len(self.batch)", "self.batch.pop").
func accumulatorIsBounded(bodyText, recv string) bool {
	patterns := []string{
		recv + ".clear",
		recv + ".flush",
		recv + ".pop",
		recv + " = ",
		recv + " =\n",
		"len(" + recv + ")",
		recv + "[:",
		recv + "[0",
	}
	for _, p := range patterns {
		if strings.Contains(bodyText, p) {
			return true
		}
	}
	return false
}
