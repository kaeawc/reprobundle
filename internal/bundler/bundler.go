// Package bundler copies the sliced files into an output directory preserving
// the package layout the imports require, writes repro.sh, and emits a
// MANIFEST.md mapping each included file to the reason it was kept.
package bundler
