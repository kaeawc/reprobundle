// Package resolver statically resolves resource references (open(path),
// load_template, read_config) for files in the slice. Falls back to runtime
// tracing for dynamic paths when the NeedsRuntimeTrace capability is granted.
package resolver
