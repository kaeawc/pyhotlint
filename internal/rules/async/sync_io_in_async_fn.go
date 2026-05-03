// Package async holds rules in the "async correctness" taxonomy.
package async

import (
	"strings"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	sitter "github.com/smacker/go-tree-sitter"
)

// blockingCalls maps a fully qualified dotted call name to the human
// description used in the finding message. The match is on the literal
// dotted text of the call's `function` field — receiver/owner proof per
// the rule guardrails. We do not match `mytimer.sleep` against
// `time.sleep`, only `time.sleep` itself or `sleep` from `from time import sleep`
// (handled separately below via the import-tracking pass).
var blockingCalls = map[string]string{
	"time.sleep":              "time.sleep blocks the event loop; use asyncio.sleep",
	"requests.get":            "requests.get is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.post":           "requests.post is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.put":            "requests.put is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.patch":          "requests.patch is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.delete":         "requests.delete is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.head":           "requests.head is sync HTTP; use httpx.AsyncClient or aiohttp",
	"requests.request":        "requests.request is sync HTTP; use httpx.AsyncClient or aiohttp",
	"urllib.request.urlopen":  "urllib.request.urlopen blocks; use an async HTTP client",
	"subprocess.run":          "subprocess.run blocks; use asyncio.create_subprocess_exec",
	"subprocess.call":         "subprocess.call blocks; use asyncio.create_subprocess_exec",
	"subprocess.check_call":   "subprocess.check_call blocks; use asyncio.create_subprocess_exec",
	"subprocess.check_output": "subprocess.check_output blocks; use asyncio.create_subprocess_exec",
}

// bareBlockingCalls names sync-only callables that may appear unqualified
// (typically due to `from X import Y`). We treat a bare call to one of
// these as suspicious only when the corresponding `from` import is in
// the file. `open()` is treated specially since it is a builtin.
var bareBlockingCalls = map[string]struct {
	from string
	msg  string
}{
	"sleep":   {from: "time", msg: "time.sleep blocks the event loop; use asyncio.sleep"},
	"urlopen": {from: "urllib.request", msg: "urlopen blocks; use an async HTTP client"},
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "sync-io-in-async-fn",
		Category:    "async",
		Severity:    v2.SeverityWarning,
		Description: "Calls a blocking sync API inside an async def, stalling the event loop.",
		NodeTypes:   []string{"function_definition"},
		Confidence:  0.9,
		Check:       checkSyncIOInAsyncFn,
	})
}

func checkSyncIOInAsyncFn(ctx *v2.Context, fn *sitter.Node) {
	if !isAsyncFunction(fn) {
		return
	}
	body := fn.ChildByFieldName("body")
	if body == nil {
		return
	}

	imports := collectFromImports(rootOf(fn), ctx.Source)

	walkSameAsyncScope(body, func(n *sitter.Node) {
		if n.Type() != "call" {
			return
		}
		fnExpr := n.ChildByFieldName("function")
		if fnExpr == nil {
			return
		}
		text := ctx.NodeText(fnExpr)
		if msg, ok := blockingCalls[text]; ok {
			ctx.Emit(n, msg)
			return
		}
		// Bare-name calls (from-imports). Match only when the
		// corresponding `from MODULE import NAME` is in the file.
		if fnExpr.Type() == "identifier" {
			if entry, ok := bareBlockingCalls[text]; ok {
				if imports[entry.from+"."+text] {
					ctx.Emit(n, entry.msg)
				}
			}
			if text == "open" {
				ctx.Emit(n, "open() blocks; use aiofiles or asyncio.to_thread")
			}
		}
	})
}

// isAsyncFunction reports whether a function_definition node is `async def`.
// In tree-sitter Python the `async` modifier is an unnamed `async` token
// child preceding the `def` keyword.
func isAsyncFunction(fn *sitter.Node) bool {
	count := int(fn.ChildCount())
	for i := 0; i < count; i++ {
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

// walkSameAsyncScope visits descendant nodes of body that belong to the
// same async scope — i.e. it does NOT descend into nested
// function_definition, class_definition, or lambda nodes. A `time.sleep`
// inside a sync nested function is fine; flagging it would be a false
// positive per the rule guardrails.
func walkSameAsyncScope(body *sitter.Node, visit func(*sitter.Node)) {
	var rec func(n *sitter.Node)
	rec = func(n *sitter.Node) {
		t := n.Type()
		// Stop at nested scopes that establish their own async-ness.
		if t == "function_definition" || t == "class_definition" || t == "lambda" {
			return
		}
		visit(n)
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			rec(c)
		}
	}
	count := int(body.ChildCount())
	for i := 0; i < count; i++ {
		c := body.Child(i)
		if c == nil {
			continue
		}
		rec(c)
	}
}

// collectFromImports returns a set of "module.name" entries gathered
// from `from MODULE import NAME[, NAME...]` statements in the file. We
// normalize aliasing only minimally: `from time import sleep as s`
// records `time.sleep` plus tracks the alias separately if it ever
// matters; for MVP we ignore aliases.
func collectFromImports(root *sitter.Node, src []byte) map[string]bool {
	out := map[string]bool{}
	if root == nil {
		return out
	}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "import_from_statement" {
			module := n.ChildByFieldName("module_name")
			if module != nil {
				modText := strings.TrimSpace(string(src[module.StartByte():module.EndByte()]))
				count := int(n.ChildCount())
				for i := 0; i < count; i++ {
					c := n.Child(i)
					if c == nil || c.Type() != "dotted_name" {
						continue
					}
					if c == module {
						continue
					}
					name := strings.TrimSpace(string(src[c.StartByte():c.EndByte()]))
					if name != "" {
						out[modText+"."+name] = true
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

// rootOf walks parents until it finds the module root.
func rootOf(n *sitter.Node) *sitter.Node {
	for n != nil && n.Parent() != nil {
		n = n.Parent()
	}
	return n
}
