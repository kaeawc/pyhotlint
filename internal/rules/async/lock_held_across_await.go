package async

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// lockConstructors enumerates the call expressions that bind a name to
// a lock instance. Receiver/owner proof — we do not match a bare
// `Lock()` call, only the dotted form, to avoid collisions with
// user-defined Lock classes.
var lockConstructors = map[string]bool{
	"threading.Lock":  true,
	"threading.RLock": true,
	"asyncio.Lock":    true,
	"asyncio.RLock":   true,
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "lock-held-across-await",
		Category:    "async",
		Severity:    v2.SeverityWarning,
		Description: "A threading or asyncio Lock is held across an await; risk of deadlock or starvation. If intentional, justify with an inline comment.",
		NodeTypes:   []string{"function_definition"},
		Confidence:  0.8,
		Check:       checkLockHeldAcrossAwait,
	})
}

func checkLockHeldAcrossAwait(ctx *v2.Context, fn *sitter.Node) {
	if !isAsyncFunction(fn) {
		return
	}
	body := fn.ChildByFieldName("body")
	if body == nil {
		return
	}
	lockNames := getLockBindings(ctx)

	walkSameAsyncScope(body, func(n *sitter.Node) {
		if n.Type() != "with_statement" {
			return
		}
		if !withHoldsLock(n, ctx.Source, lockNames) {
			return
		}
		withBody := withStatementBody(n)
		if withBody == nil || !subtreeContainsAwait(withBody) {
			return
		}
		if hasInlineComment(n, ctx.Source) {
			return
		}
		ctx.Emit(n, "lock held across await; release before awaiting or document why this is safe with an inline comment")
	})
}

// collectLockBindings scans the file for `<name> = <ctor>()` patterns
// where ctor is one of the known lock constructors.
func collectLockBindings(root *sitter.Node, src []byte) map[string]bool {
	out := map[string]bool{}
	if root == nil {
		return out
	}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "assignment" {
			left := n.ChildByFieldName("left")
			right := n.ChildByFieldName("right")
			if left != nil && right != nil && right.Type() == "call" {
				callFn := right.ChildByFieldName("function")
				if callFn != nil {
					ctor := strings.TrimSpace(string(src[callFn.StartByte():callFn.EndByte()]))
					if lockConstructors[ctor] && left.Type() == "identifier" {
						varName := strings.TrimSpace(string(src[left.StartByte():left.EndByte()]))
						out[varName] = true
					}
				}
			}
		}
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			walk(c)
		}
	}
	walk(root)
	return out
}

// getLockBindings memoizes collectLockBindings on Context. Walks the
// file once instead of once per async function_definition node.
func getLockBindings(ctx *v2.Context) map[string]bool {
	return ctx.Cached("async.lock_bindings", func() any {
		return collectLockBindings(ctx.Root, ctx.Source)
	}).(map[string]bool)
}

// withHoldsLock reports whether any with-item in the with-statement
// resolves to a known lock instance.
func withHoldsLock(withStmt *sitter.Node, src []byte, lockNames map[string]bool) bool {
	for i := 0; i < int(withStmt.ChildCount()); i++ {
		c := withStmt.Child(i)
		if c == nil || c.Type() != "with_clause" {
			continue
		}
		if withClauseHoldsLock(c, src, lockNames) {
			return true
		}
	}
	return false
}

func withClauseHoldsLock(clause *sitter.Node, src []byte, lockNames map[string]bool) bool {
	for j := 0; j < int(clause.ChildCount()); j++ {
		item := clause.Child(j)
		if item == nil || item.Type() != "with_item" {
			continue
		}
		val := item.ChildByFieldName("value")
		if val == nil {
			val = item.NamedChild(0)
		}
		if withItemHoldsLock(val, src, lockNames) {
			return true
		}
	}
	return false
}

func withItemHoldsLock(val *sitter.Node, src []byte, lockNames map[string]bool) bool {
	if val == nil {
		return false
	}
	switch val.Type() {
	case "call":
		callFn := val.ChildByFieldName("function")
		if callFn == nil {
			return false
		}
		name := strings.TrimSpace(string(src[callFn.StartByte():callFn.EndByte()]))
		return lockConstructors[name]
	case "identifier":
		name := strings.TrimSpace(string(src[val.StartByte():val.EndByte()]))
		return lockNames[name]
	}
	return false
}

// withStatementBody returns the `block` child of a with_statement.
func withStatementBody(withStmt *sitter.Node) *sitter.Node {
	count := int(withStmt.ChildCount())
	for i := 0; i < count; i++ {
		c := withStmt.Child(i)
		if c != nil && c.Type() == "block" {
			return c
		}
	}
	return withStmt.ChildByFieldName("body")
}

// subtreeContainsAwait reports whether root has any await expression in
// its subtree, stopping at nested function/class/lambda boundaries.
func subtreeContainsAwait(root *sitter.Node) bool {
	if root == nil {
		return false
	}
	found := false
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if found {
			return
		}
		t := n.Type()
		if t == "function_definition" || t == "class_definition" || t == "lambda" {
			return
		}
		if t == "await" {
			found = true
			return
		}
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			walk(c)
		}
	}
	walk(root)
	return found
}

// hasInlineComment reports whether the line containing withStmt's first
// byte has a `#` comment marker. Naive scan that respects single- and
// double-quoted strings; good enough for MVP.
func hasInlineComment(withStmt *sitter.Node, src []byte) bool {
	start := int(withStmt.StartByte())
	if start >= len(src) {
		return false
	}
	end := start
	for end < len(src) && src[end] != '\n' {
		end++
	}
	line := src[start:end]
	inSingle, inDouble := false, false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case !inDouble && ch == '\'':
			inSingle = !inSingle
		case !inSingle && ch == '"':
			inDouble = !inDouble
		case !inSingle && !inDouble && ch == '#':
			return true
		}
	}
	return false
}
