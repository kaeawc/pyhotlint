package oracle

// Fake is an in-memory Oracle for tests. Programmable per-method.
type Fake struct {
	Devices    map[string]Result // expr -> Result
	NNModules  map[string]Result // qualname -> Result
	CloseError error             // returned by Close
}

// NewFake returns a Fake with empty maps.
func NewFake() *Fake {
	return &Fake{
		Devices:   map[string]Result{},
		NNModules: map[string]Result{},
	}
}

// DeviceOf returns the canned response, or Unknown when absent.
func (f *Fake) DeviceOf(expr string) Result {
	if r, ok := f.Devices[expr]; ok {
		return r
	}
	return Unknown
}

// SubclassesNNModule returns the canned response, or Unknown when absent.
func (f *Fake) SubclassesNNModule(qualname string) Result {
	if r, ok := f.NNModules[qualname]; ok {
		return r
	}
	return Unknown
}

// Close returns f.CloseError. Safe to call multiple times.
func (f *Fake) Close() error { return f.CloseError }
