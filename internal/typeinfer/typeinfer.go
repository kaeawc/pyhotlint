// Package typeinfer is the source-level type tracker — same role as Krit's
// internal/typeinfer. MVP stub: exposes the API surface so rules can be
// written against it; real implementation comes later.
package typeinfer

// Resolver tracks types of locals across a single file's AST.
type Resolver struct{}

// New returns an empty Resolver.
func New() *Resolver { return &Resolver{} }

// TypeOf returns the inferred type of expr, or "" when unknown.
func (r *Resolver) TypeOf(expr string) string { return "" }
