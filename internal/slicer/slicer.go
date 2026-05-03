// Package slicer walks the call graph from intake seeds and computes the
// reachable code, prompt, and config slices. Capability-gated: NeedsCallGraph
// (always), NeedsRuntimeTrace (dynamic paths), NeedsDataExtractor (dataset
// row slicing).
package slicer
