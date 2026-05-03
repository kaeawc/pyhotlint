// Package tensor holds rules in the "tensor / device hygiene" taxonomy.
package tensor

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// noGradContexts enumerates context-manager call expressions that
// disable gradient tracking.
var noGradContexts = map[string]bool{
	"torch.no_grad":        true,
	"torch.inference_mode": true,
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "untracked-grad-in-eval",
		Category:    "tensor",
		Severity:    v2.SeverityWarning,
		Description: ".eval() called without torch.no_grad() / torch.inference_mode() in the surrounding scope; gradient buffers are still allocated during inference.",
		NodeTypes:   []string{"call"},
		Confidence:  0.6,
		Check:       checkUntrackedGradInEval,
	})
}

func checkUntrackedGradInEval(ctx *v2.Context, call *sitter.Node) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return
	}
	attrLeaf := fnExpr.ChildByFieldName("attribute")
	if attrLeaf == nil {
		return
	}
	if string(ctx.Source[attrLeaf.StartByte():attrLeaf.EndByte()]) != "eval" {
		return
	}
	// Receiver/owner proof: model.eval() takes zero arguments. A call
	// like `engine.eval(expr)` is a different API.
	args := call.ChildByFieldName("arguments")
	if args == nil || args.NamedChildCount() != 0 {
		return
	}
	if hasNoGradAncestor(call, ctx.Source) {
		return
	}
	ctx.Emit(call, ".eval() in scope without torch.no_grad() / torch.inference_mode(); gradients are still tracked during forward passes")
}

// hasNoGradAncestor walks parent nodes looking for a `with torch.no_grad():`
// or `with torch.inference_mode():` enclosing block, or a function
// decorated with @torch.no_grad / @torch.inference_mode.
func hasNoGradAncestor(node *sitter.Node, src []byte) bool {
	cur := node.Parent()
	for cur != nil {
		switch cur.Type() {
		case "with_statement":
			if withItemMatchesContext(cur, src, noGradContexts) {
				return true
			}
		case "function_definition":
			if functionHasDecorator(cur, src, noGradContexts) {
				return true
			}
		}
		cur = cur.Parent()
	}
	return false
}

func withItemMatchesContext(withStmt *sitter.Node, src []byte, table map[string]bool) bool {
	count := int(withStmt.ChildCount())
	for i := 0; i < count; i++ {
		c := withStmt.Child(i)
		if c == nil || c.Type() != "with_clause" {
			continue
		}
		ic := int(c.ChildCount())
		for j := 0; j < ic; j++ {
			item := c.Child(j)
			if item == nil || item.Type() != "with_item" {
				continue
			}
			val := item.ChildByFieldName("value")
			if val == nil {
				val = item.NamedChild(0)
			}
			if val == nil || val.Type() != "call" {
				continue
			}
			callFn := val.ChildByFieldName("function")
			if callFn == nil {
				continue
			}
			name := strings.TrimSpace(string(src[callFn.StartByte():callFn.EndByte()]))
			if table[name] {
				return true
			}
		}
	}
	return false
}

// functionHasDecorator reports whether fn lives under a
// decorated_definition whose decorators include any name in table.
func functionHasDecorator(fn *sitter.Node, src []byte, table map[string]bool) bool {
	parent := fn.Parent()
	if parent == nil || parent.Type() != "decorated_definition" {
		return false
	}
	count := int(parent.ChildCount())
	for i := 0; i < count; i++ {
		c := parent.Child(i)
		if c == nil || c.Type() != "decorator" {
			continue
		}
		expr := c.NamedChild(0)
		if expr == nil {
			continue
		}
		var name string
		if expr.Type() == "call" {
			callFn := expr.ChildByFieldName("function")
			if callFn != nil {
				name = strings.TrimSpace(string(src[callFn.StartByte():callFn.EndByte()]))
			}
		} else {
			name = strings.TrimSpace(string(src[expr.StartByte():expr.EndByte()]))
		}
		if table[name] {
			return true
		}
	}
	return false
}
