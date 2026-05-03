// Package slicer walks the call graph from intake seeds and computes the
// reachable code, prompt, and config slices. Capability-gated: NeedsCallGraph
// (always), NeedsRuntimeTrace (dynamic paths), NeedsDataExtractor (dataset
// row slicing).
package slicer

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/kaeawc/reprobundle/internal/scanner"
)

// FileSet is the result of walking imports out from an entry point. Files
// is sorted, deduplicated, and contains every repo-local .py file the
// entry transitively imports (including the entry itself). External lists
// the dotted module names for imports that resolved as external or could
// not be resolved at all; the bundler omits these but the manifest cites
// them so users know what the bundled slice still depends on.
type FileSet struct {
	Files    []string
	External []string
}

// WalkFiles starts at entry and walks transitive Python imports through
// resolver, returning every repo-local file in the closure plus the
// modules that fell outside it. resolver.FS is used for file reads, so the
// caller must construct it against the same root.
func WalkFiles(ctx context.Context, resolver *scanner.PyResolver, entry string) (FileSet, error) {
	if resolver == nil || resolver.FS == nil {
		return FileSet{}, fmt.Errorf("slicer: nil resolver or FS")
	}
	if entry == "" {
		return FileSet{}, fmt.Errorf("slicer: empty entry path")
	}

	seen := map[string]bool{}
	external := map[string]bool{}
	queue := []string{entry}

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return FileSet{}, err
		}
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true

		src, err := fs.ReadFile(resolver.FS, current)
		if err != nil {
			return FileSet{}, fmt.Errorf("slicer: read %s: %w", current, err)
		}
		file, err := scanner.ParsePython(ctx, current, src)
		if err != nil {
			return FileSet{}, fmt.Errorf("slicer: parse %s: %w", current, err)
		}
		imports, ierr := scanner.Imports(file)
		file.Close()
		if ierr != nil {
			return FileSet{}, fmt.Errorf("slicer: imports %s: %w", current, ierr)
		}

		for _, imp := range imports {
			res := resolver.ResolveImport(current, imp)
			switch res.Kind {
			case scanner.ResolutionModule, scanner.ResolutionPackage:
				if !seen[res.Path] {
					queue = append(queue, res.Path)
				}
				// Python needs every parent package's __init__.py
				// to import the resolved file at runtime, so the
				// bundle has to ship them too.
				for _, init := range parentInits(resolver.FS, res.Path) {
					if !seen[init] {
						queue = append(queue, init)
					}
				}
			default:
				if name := externalName(imp); name != "" {
					external[name] = true
				}
			}
		}
	}

	files := keys(seen)
	sort.Strings(files)
	ext := keys(external)
	sort.Strings(ext)
	return FileSet{Files: files, External: ext}, nil
}

// externalName picks the dotted name to record for an unresolved import.
// For relative imports we synthesise a "<dots><module>" string so the
// manifest preserves the importer's view of the name; relative-and-empty
// imports (which would just be ".") are dropped.
func externalName(imp scanner.Import) string {
	if imp.Relative > 0 {
		dots := ""
		for i := 0; i < imp.Relative; i++ {
			dots += "."
		}
		switch {
		case imp.Module != "" && imp.Name != "" && imp.Name != "*":
			return dots + imp.Module + "." + imp.Name
		case imp.Module != "":
			return dots + imp.Module
		case imp.Name != "" && imp.Name != "*":
			return dots + imp.Name
		default:
			return ""
		}
	}
	switch {
	case imp.Module != "" && imp.Name != "" && imp.Name != "*":
		return imp.Module + "." + imp.Name
	case imp.Module != "":
		return imp.Module
	case imp.Name != "" && imp.Name != "*":
		return imp.Name
	default:
		return ""
	}
}

// parentInits returns the __init__.py files for every directory between
// the file and the FS root, in root-first order. Files that don't exist
// (the test directory, the project root) are skipped.
func parentInits(fsys fs.FS, file string) []string {
	dir := path.Dir(file)
	if dir == "." || dir == "/" || dir == "" {
		return nil
	}
	parts := strings.Split(dir, "/")
	var out []string
	for i := 1; i <= len(parts); i++ {
		candidate := path.Join(path.Join(parts[:i]...), "__init__.py")
		if info, err := fs.Stat(fsys, candidate); err == nil && !info.IsDir() {
			out = append(out, candidate)
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
