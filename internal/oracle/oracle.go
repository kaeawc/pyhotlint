// Package oracle hosts PyOracle: a Python subprocess attached to the
// project's interpreter that answers questions the source-level
// analyzer cannot — tensor device, nn.Module subclassing, full FQN
// resolution. Capability-gated; rules with NeedsOracle set in their
// Capabilities consume it via ctx.Oracle.
//
// The package exposes three implementations:
//
//	Stub        — always returns Unknown. Used when no Python
//	              interpreter is discoverable.
//	Subprocess  — real Python subprocess with newline-delimited JSON
//	              over stdin/stdout.
//	Fake        — programmable in-memory implementation for tests.
//
// All three satisfy the Oracle interface and are safe to call from
// multiple goroutines (Subprocess serializes).
package oracle

// Result carries an answer plus a Known flag. Unknown means the oracle
// could not (or chose not to) resolve the question; rules must treat
// Unknown the same as "no oracle wired" and skip.
type Result struct {
	Known bool
	Value string
}

// Unknown is the zero-value response.
var Unknown = Result{}

// Oracle is the query surface used by rules.
type Oracle interface {
	// DeviceOf resolves the device a tensor expression sits on
	// (e.g. "cpu", "cuda:0"). Returns Unknown when undecidable.
	DeviceOf(expr string) Result

	// SubclassesNNModule reports whether a fully qualified class name
	// is a torch.nn.Module subclass. Returns Unknown when the import
	// could not be resolved.
	SubclassesNNModule(qualname string) Result

	// Close releases any underlying subprocess. Safe to call multiple
	// times; subsequent calls are no-ops.
	Close() error
}

// Stub is the no-Python-interpreter implementation: every query
// returns Unknown.
type Stub struct{}

func (Stub) DeviceOf(string) Result           { return Unknown }
func (Stub) SubclassesNNModule(string) Result { return Unknown }
func (Stub) Close() error                     { return nil }

// New returns an Oracle. Convenience for tests: returns a Stub.
func New() Oracle { return Stub{} }
