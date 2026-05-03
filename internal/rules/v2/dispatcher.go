package v2

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// Run executes rules against a parsed file. Each rule sees every node
// whose type matches its NodeTypes set. The walk is a single preorder
// traversal of the named-and-anonymous tree.
func Run(rules []*Rule, file string, source []byte, root *sitter.Node) []Finding {
	var findings []Finding
	if root == nil {
		return findings
	}

	byType := make(map[string][]*Rule, len(rules))
	for _, r := range rules {
		if r.Needs&NeedsLinePass != 0 {
			continue // line rules are not node-dispatched
		}
		for _, nt := range r.NodeTypes {
			byType[nt] = append(byType[nt], r)
		}
	}
	if len(byType) == 0 {
		return findings
	}

	ctx := NewContext(file, source, &findings)
	walk(root, func(n *sitter.Node) {
		rs := byType[n.Type()]
		for _, r := range rs {
			ctx.SetRule(r)
			r.Check(ctx, n)
		}
	})
	return findings
}

// walk performs a preorder traversal over named children using a tree
// cursor. Cheap and avoids recursion blow-up on deep ASTs.
func walk(root *sitter.Node, visit func(*sitter.Node)) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	visit(cursor.CurrentNode())
	if !cursor.GoToFirstChild() {
		return
	}
	for {
		visit(cursor.CurrentNode())
		if cursor.GoToFirstChild() {
			continue
		}
		for !cursor.GoToNextSibling() {
			if !cursor.GoToParent() {
				return
			}
		}
	}
}
