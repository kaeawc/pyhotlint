package server

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// prometheusMetricCtors is the set of Prometheus client metric class
// names. Receiver/owner proof: we additionally require that the call
// resolves to the prometheus_client module (via `from prometheus_client
// import X` or `import prometheus_client[ as alias]`), so a
// user-defined `Counter` class is not flagged.
var prometheusMetricCtors = map[string]bool{
	"Counter":   true,
	"Gauge":     true,
	"Histogram": true,
	"Summary":   true,
	"Info":      true,
	"Enum":      true,
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "metric-not-labeled",
		Category:    "server",
		Severity:    v2.SeverityWarning,
		Description: "Prometheus metric created without `labelnames=...`; the metric cannot be split per service / route in dashboards.",
		NodeTypes:   []string{"call"},
		Confidence:  0.85,
		Check:       checkMetricNotLabeled,
	})
}

// promImports describes the prometheus_client import shape of a file.
//
//	bareCtors:   local-name -> original-class-name. Captures
//	             `from prometheus_client import Counter` (Counter -> Counter)
//	             and `from prometheus_client import Counter as Ctr`
//	             (Ctr -> Counter). The original name is what we
//	             match against prometheusMetricCtors.
//	moduleNames: local-name -> nothing. The local binding under which
//	             the prometheus_client module is reachable as a dotted
//	             receiver, including aliases (`import prometheus_client
//	             as pc` records "pc").
type promImports struct {
	bareCtors   map[string]string
	moduleNames map[string]struct{}
}

func checkMetricNotLabeled(ctx *v2.Context, call *sitter.Node) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil {
		return
	}
	imp := getPromImports(ctx)
	if !callTargetsPromCtor(fnExpr, ctx.Source, imp) {
		return
	}
	args := call.ChildByFieldName("arguments")
	if args != nil && hasKeywordArg(args, ctx.Source, "labelnames") {
		return
	}
	ctx.Emit(call, "prometheus metric created without labelnames; cannot split by service or route")
}

// getPromImports memoizes the per-file prometheus_client import shape
// on Context. Without the cache we walked the entire AST once per
// `call` node — O(file_size × call_count). After: O(file_size).
func getPromImports(ctx *v2.Context) *promImports {
	return ctx.Cached("server.prom_imports", func() any {
		return collectPromImports(ctx.Root, ctx.Source)
	}).(*promImports)
}

// callTargetsPromCtor reports whether fnExpr is a call to one of the
// Prometheus metric constructors, given the file's import shape.
func callTargetsPromCtor(fnExpr *sitter.Node, src []byte, imp *promImports) bool {
	switch fnExpr.Type() {
	case "identifier":
		original, ok := imp.bareCtors[string(src[fnExpr.StartByte():fnExpr.EndByte()])]
		if !ok {
			return false
		}
		return prometheusMetricCtors[original]
	case "attribute":
		obj := fnExpr.ChildByFieldName("object")
		leaf := fnExpr.ChildByFieldName("attribute")
		if obj == nil || leaf == nil {
			return false
		}
		if !prometheusMetricCtors[string(src[leaf.StartByte():leaf.EndByte()])] {
			return false
		}
		objText := strings.TrimSpace(string(src[obj.StartByte():obj.EndByte()]))
		_, ok := imp.moduleNames[objText]
		return ok
	}
	return false
}

// hasKeywordArg reports whether args (an `argument_list` node) carries
// a keyword argument whose name matches `wanted`.
func hasKeywordArg(args *sitter.Node, src []byte, wanted string) bool {
	for i := 0; i < int(args.NamedChildCount()); i++ {
		c := args.NamedChild(i)
		if c == nil || c.Type() != "keyword_argument" {
			continue
		}
		nameNode := c.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		if string(src[nameNode.StartByte():nameNode.EndByte()]) == wanted {
			return true
		}
	}
	return false
}

// collectPromImports walks `import_from_statement` and `import_statement`
// nodes once. We do this per Check call rather than once per file
// because v2 has no per-file hook yet; the ASTs are small and this is
// O(file). Worth caching when the dispatcher gains a per-file callback.
func collectPromImports(root *sitter.Node, src []byte) *promImports {
	imp := &promImports{
		bareCtors:   map[string]string{},
		moduleNames: map[string]struct{}{},
	}
	if root == nil {
		return imp
	}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "import_from_statement":
			collectFromImport(n, src, imp)
		case "import_statement":
			collectImport(n, src, imp)
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			walk(c)
		}
	}
	walk(root)
	return imp
}

func collectFromImport(stmt *sitter.Node, src []byte, imp *promImports) {
	module := stmt.ChildByFieldName("module_name")
	if module == nil {
		return
	}
	if strings.TrimSpace(string(src[module.StartByte():module.EndByte()])) != "prometheus_client" {
		return
	}
	for i := 0; i < int(stmt.NamedChildCount()); i++ {
		c := stmt.NamedChild(i)
		if c == nil || c == module {
			continue
		}
		switch c.Type() {
		case "dotted_name":
			name := strings.TrimSpace(string(src[c.StartByte():c.EndByte()]))
			imp.bareCtors[name] = name
		case "aliased_import":
			recordFromAlias(c, src, imp.bareCtors)
		}
	}
}

func collectImport(stmt *sitter.Node, src []byte, imp *promImports) {
	for i := 0; i < int(stmt.NamedChildCount()); i++ {
		c := stmt.NamedChild(i)
		if c == nil {
			continue
		}
		switch c.Type() {
		case "dotted_name":
			name := strings.TrimSpace(string(src[c.StartByte():c.EndByte()]))
			if name == "prometheus_client" {
				imp.moduleNames[name] = struct{}{}
			}
		case "aliased_import":
			recordImportedModuleAlias(c, src, imp.moduleNames)
		}
	}
}

// recordFromAlias handles `from MOD import NAME as ALIAS`: the local
// binding is ALIAS, and the original class name is NAME. Both must be
// preserved so the call site `ALIAS(...)` can be matched against the
// metric-ctor whitelist.
func recordFromAlias(node *sitter.Node, src []byte, into map[string]string) {
	name := node.ChildByFieldName("name")
	alias := node.ChildByFieldName("alias")
	if name == nil || alias == nil {
		return
	}
	originalName := strings.TrimSpace(string(src[name.StartByte():name.EndByte()]))
	aliasName := strings.TrimSpace(string(src[alias.StartByte():alias.EndByte()]))
	into[aliasName] = originalName
}

// recordImportedModuleAlias handles `import prometheus_client as pc`:
// we only care if the underlying module name is prometheus_client.
func recordImportedModuleAlias(node *sitter.Node, src []byte, into map[string]struct{}) {
	name := node.ChildByFieldName("name")
	alias := node.ChildByFieldName("alias")
	if name == nil || alias == nil {
		return
	}
	if strings.TrimSpace(string(src[name.StartByte():name.EndByte()])) != "prometheus_client" {
		return
	}
	into[strings.TrimSpace(string(src[alias.StartByte():alias.EndByte()]))] = struct{}{}
}
