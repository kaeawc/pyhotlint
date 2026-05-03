package async

import (
	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

func init() {
	v2.Register(&v2.Rule{
		ID:          "cpu-bound-loop-in-event-loop",
		Category:    "async",
		Severity:    v2.SeverityWarning,
		Description: "A for-loop inside an async def with no await in its body blocks the event loop until completion. Move the work to a thread/process pool, or yield with `await asyncio.sleep(0)` to share the loop.",
		NodeTypes:   []string{"function_definition"},
		Confidence:  0.5,
		Check:       checkCPUBoundLoopInEventLoop,
	})
}

// checkCPUBoundLoopInEventLoop flags `for` loops nested directly in
// async-def bodies whose loop body contains zero `await` expressions.
// We deliberately stop at nested function/class/lambda boundaries so
// we don't blame an async wrapper for the contents of a sync helper
// declared inside it.
//
// This is a structural heuristic: we cannot prove the iterable is
// large or the body is heavy. Confidence is set accordingly (0.5).
// Users who legitimately want a tight loop (e.g., the iterable is
// known to be tiny) suppress with `# pyhotlint: ignore[...]`.
func checkCPUBoundLoopInEventLoop(ctx *v2.Context, fn *sitter.Node) {
	if !isAsyncFunction(fn) {
		return
	}
	body := fn.ChildByFieldName("body")
	if body == nil {
		return
	}
	walkSameAsyncScope(body, func(n *sitter.Node) {
		if n.Type() != "for_statement" {
			return
		}
		loopBody := forLoopBody(n)
		if loopBody == nil {
			return
		}
		if subtreeContainsAwait(loopBody) {
			return
		}
		ctx.Emit(n, "for-loop in async def with no await in its body blocks the event loop; offload to asyncio.to_thread or yield with `await asyncio.sleep(0)`")
	})
}

// forLoopBody returns the `block` child of a for_statement.
func forLoopBody(forStmt *sitter.Node) *sitter.Node {
	for i := 0; i < int(forStmt.ChildCount()); i++ {
		c := forStmt.Child(i)
		if c != nil && c.Type() == "block" {
			return c
		}
	}
	return forStmt.ChildByFieldName("body")
}
