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
type Context struct {
	File    string
	Source  []byte
	Project *project.Project
	Oracle  oracle.Oracle
	rule    *Rule
	results *[]Finding
}

// NewContext wires a Context to the slice it should append findings to.
// Used by the dispatcher and by tests. proj may be nil; orc may be
// nil — Stub is substituted in that case.
func NewContext(file string, source []byte, proj *project.Project, orc oracle.Oracle, results *[]Finding) *Context {
	if orc == nil {
		orc = oracle.Stub{}
	}
	return &Context{File: file, Source: source, Project: proj, Oracle: orc, results: results}
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
