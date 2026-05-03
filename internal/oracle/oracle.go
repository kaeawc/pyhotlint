// Package oracle will host the PyOracle subprocess: a Python interpreter
// attached to the project venv, queried over JSON-RPC for facts the
// source-level analyzer cannot see (tensor device, nn.Module subclassing,
// resolved import targets). MVP stub: every query returns Unknown.
package oracle

// Result carries an answer plus a confidence flag. Unknown means the
// oracle could not (or chose not to) resolve the question.
type Result struct {
	Known bool
	Value string
}

// Unknown is the sentinel returned by the stub.
var Unknown = Result{Known: false}

// Oracle is the query surface. Stub implementation returns Unknown for
// every question.
type Oracle struct{}

// New returns a stub Oracle. The real one will start a Python subprocess.
func New() *Oracle { return &Oracle{} }

// DeviceOf returns the device a tensor expression resolves to, or
// Unknown when undecidable.
func (o *Oracle) DeviceOf(expr string) Result { return Unknown }

// SubclassesNNModule reports whether a class is an nn.Module subclass.
func (o *Oracle) SubclassesNNModule(qualName string) Result { return Unknown }
