// Package v2 provides the unified rule interface for pyhotlint.
//
// A Rule declares the tree-sitter node types it cares about and the
// capabilities the dispatcher must provide. The dispatcher walks each
// parsed file once and routes matching nodes to each rule's Check
// callback. Mirrors Krit's v2 registry, scoped to Python.
package v2

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// Capabilities declares what the dispatcher must provide to a rule.
type Capabilities uint32

const (
	// NeedsProject requests project metadata (pyproject + lockfile) on Context.
	NeedsProject Capabilities = 1 << iota
	// NeedsTypeInfer requests source-level type inference on Context.
	NeedsTypeInfer
	// NeedsOracle requests the PyOracle subprocess. Opt-in; expensive.
	NeedsOracle
	// NeedsLinePass marks this rule as a line-scanning rule (receives lines, not nodes).
	NeedsLinePass
)

// Severity controls how findings are surfaced.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// FixLevel is the autofix safety tier, mirroring Krit.
type FixLevel uint8

const (
	FixNone FixLevel = iota
	FixCosmetic
	FixIdiomatic
	FixSemantic
)

// Rule is the dispatchable unit of analysis.
type Rule struct {
	ID          string
	Category    string
	Severity    Severity
	Description string
	// NodeTypes is the set of tree-sitter node type names this rule wants
	// to receive. The dispatcher routes each node of these types to Check.
	// A line rule (NeedsLinePass) leaves NodeTypes nil.
	NodeTypes  []string
	Needs      Capabilities
	Fix        FixLevel
	Confidence float64
	// Check runs on each node whose type matches NodeTypes.
	Check func(ctx *Context, node *sitter.Node)
}

var registered []*Rule

// Register adds r to the global rule registry. Call from an init().
func Register(r *Rule) {
	if r == nil {
		return
	}
	registered = append(registered, r)
}

// All returns a snapshot of the registered rules.
func All() []*Rule {
	out := make([]*Rule, len(registered))
	copy(out, registered)
	return out
}
