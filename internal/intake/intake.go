// Package intake resolves a user-supplied entry point (pytest test ID, agent
// class name, or JSONL trace) into the seed locations the slicer walks from.
package intake

// Kind identifies which form of entry point produced a Seed.
type Kind int

const (
	KindUnknown Kind = iota
	KindPytest
)

func (k Kind) String() string {
	switch k {
	case KindPytest:
		return "pytest"
	default:
		return "unknown"
	}
}

// Seed is a starting point for the slicer's call-graph walk. File is the
// source file containing the entry symbol; Class is the enclosing class for
// method-based tests (empty for module-level functions); Symbol is the name
// of the function or method the slicer treats as the root frame.
type Seed struct {
	Kind   Kind
	File   string
	Class  string
	Symbol string
	// Param is the pytest parametrize ID, e.g. "0-foo" from
	// `test_foo[0-foo]`. Not used for resolution; kept so the bundler can
	// surface it in the manifest.
	Param string
}
