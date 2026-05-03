package v2

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kaeawc/pyhotlint/internal/oracle"
	"github.com/kaeawc/pyhotlint/internal/project"
)

// Finding is the per-rule analysis output.
type Finding struct {
	Rule     string   `json:"rule"`
	Category string   `json:"category"`
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Col      int      `json:"col"`
	EndLine  int      `json:"endLine"`
	EndCol   int      `json:"endCol"`
	Message  string   `json:"message"`
}

// Context is handed to each rule's Check callback.
//
// Project may be nil — version-drift rules guard with `c.Project ==
// nil` and skip when there's no project context.
//
// Oracle is non-nil but may be a Stub; rules check `r.Known` on each
// returned Result and skip the finding when the oracle declined to
// resolve the question.
//
// Root is the file's tree-sitter module root. Rules that need to scan
// the whole file (imports, global assignments, comments) walk from
// here rather than calling Parent() repeatedly from the dispatched
// node.
type Context struct {
	File    string
	Source  []byte
	Root    *sitter.Node
	Project *project.Project
	Oracle  oracle.Oracle
	rule    *Rule
	results *[]Finding
	cache   map[string]any
}

// NewContext wires a Context to the slice it should append findings to.
// Used by the dispatcher and by tests. proj may be nil; orc may be
// nil — Stub is substituted in that case.
func NewContext(file string, source []byte, root *sitter.Node, proj *project.Project, orc oracle.Oracle, results *[]Finding) *Context {
	if orc == nil {
		orc = oracle.Stub{}
	}
	return &Context{
		File:    file,
		Source:  source,
		Root:    root,
		Project: proj,
		Oracle:  orc,
		results: results,
		cache:   map[string]any{},
	}
}

// Cached memoizes per-file derived data so repeated rule callbacks
// against the same file do not re-walk the source. The cache lives
// for the lifetime of a single Run invocation; each new file gets a
// fresh Context with an empty cache.
//
// Build must be deterministic and pure with respect to the file's
// source — the result is cached forever within the file. Callers
// type-assert the return value to whatever type build returned.
//
// Convention: keys are namespaced by package, e.g. "async.from_imports",
// "server.prom_imports", "tensor.device_tracker". Avoid generic keys
// that could collide between rule packages.
func (c *Context) Cached(key string, build func() any) any {
	if v, ok := c.cache[key]; ok {
		return v
	}
	v := build()
	c.cache[key] = v
	return v
}

// SetRule is called by the dispatcher before each Check invocation so
// Emit attributes findings to the correct rule.
func (c *Context) SetRule(r *Rule) { c.rule = r }

// NodeText returns the source slice covered by n.
func (c *Context) NodeText(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return string(c.Source[n.StartByte():n.EndByte()])
}

// Emit attaches a finding to node n with the given message.
func (c *Context) Emit(n *sitter.Node, msg string) {
	if n == nil || c.rule == nil {
		return
	}
	start := n.StartPoint()
	end := n.EndPoint()
	*c.results = append(*c.results, Finding{
		Rule:     c.rule.ID,
		Category: c.rule.Category,
		Severity: c.rule.Severity,
		File:     c.File,
		Line:     int(start.Row) + 1,
		Col:      int(start.Column) + 1,
		EndLine:  int(end.Row) + 1,
		EndCol:   int(end.Column) + 1,
		Message:  msg,
	})
}
