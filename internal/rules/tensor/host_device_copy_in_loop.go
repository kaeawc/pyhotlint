package tensor

import (
	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// hostDeviceCopyMethods is the set of attribute leaf names that move a
// tensor between host and device. `.to(device)` is included because it
// is the most common modern pattern; we require it to take at least
// one argument so we don't false-positive on, say, `.to_dict()`.
var hostDeviceCopyMethods = map[string]bool{
	"cpu":  true,
	"cuda": true,
	"to":   true,
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "host-device-copy-in-loop",
		Category:    "tensor",
		Severity:    v2.SeverityWarning,
		Description: "A host/device copy (.cpu(), .cuda(), .to(device)) inside a for/while loop blocks on each iteration; pre-copy or batch.",
		NodeTypes:   []string{"call"},
		Confidence:  0.7,
		Check:       checkHostDeviceCopyInLoop,
	})
}

func checkHostDeviceCopyInLoop(ctx *v2.Context, call *sitter.Node) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return
	}
	leaf := fnExpr.ChildByFieldName("attribute")
	if leaf == nil {
		return
	}
	leafName := string(ctx.Source[leaf.StartByte():leaf.EndByte()])
	if !hostDeviceCopyMethods[leafName] {
		return
	}
	// `.to(...)` requires at least one argument; `.cpu()` / `.cuda()`
	// are zero-arg.
	args := call.ChildByFieldName("arguments")
	if leafName == "to" {
		if args == nil || args.NamedChildCount() == 0 {
			return
		}
	}
	if !insideLoop(call) {
		return
	}
	ctx.Emit(call, "host/device copy ."+leafName+"(...) inside a loop; copy outside the loop or move the loop to the device")
}

// insideLoop reports whether node has a for_statement or
// while_statement ancestor before any function/class/lambda boundary.
// We stop at scope boundaries so a `.cpu()` in a sync helper declared
// inside a loop is not (incorrectly) blamed on the outer loop.
func insideLoop(node *sitter.Node) bool {
	cur := node.Parent()
	for cur != nil {
		switch cur.Type() {
		case "for_statement", "while_statement":
			return true
		case "function_definition", "class_definition", "lambda":
			return false
		}
		cur = cur.Parent()
	}
	return false
}
