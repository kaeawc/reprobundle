package scanner

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ResolutionKind classifies the outcome of resolving an Import.
type ResolutionKind int

const (
	// ResolutionUnresolved means no candidate file was found and the
	// module is not known to be external. Callers should treat this as a
	// recoverable warning, not a fatal error.
	ResolutionUnresolved ResolutionKind = iota
	// ResolutionModule means the import maps to a single .py file under
	// one of the search roots.
	ResolutionModule
	// ResolutionPackage means the import maps to a package directory and
	// Path is the package's __init__.py.
	ResolutionPackage
	// ResolutionExternal means the module exists outside any search root
	// (a third-party dependency or stdlib). The bundler skips these but
	// the manifest still records that the import was seen.
	ResolutionExternal
)

func (k ResolutionKind) String() string {
	switch k {
	case ResolutionModule:
		return "module"
	case ResolutionPackage:
		return "package"
	case ResolutionExternal:
		return "external"
	default:
		return "unresolved"
	}
}

// Resolution is the outcome of mapping an Import to a file under the repo.
type Resolution struct {
	Kind   ResolutionKind
	Path   string
	Reason string
}

// PyResolver resolves Python module references to repo-relative file paths.
//
// FS is the repository filesystem. SearchRoots is the ordered list of
// repo-relative directories to treat as sys.path entries; the empty list
// defaults to []string{"."}, i.e. only the repo root.
type PyResolver struct {
	FS          fs.FS
	SearchRoots []string
}

// NewPyResolver constructs a resolver rooted at fsys with the default
// search root list ([]string{"."}).
func NewPyResolver(fsys fs.FS) *PyResolver {
	return &PyResolver{FS: fsys, SearchRoots: []string{"."}}
}

// ResolveImport maps an Import to a file under one of the search roots, or
// reports that the import is external/unresolved. importerPath must be the
// repo-relative path of the file that contains the import; it is only used
// for relative imports (Relative > 0).
//
// For `from X import Y` the resolver prefers a submodule resolution
// (X/Y.py or X/Y/__init__.py) and falls back to the package itself (X.py
// or X/__init__.py) if Y is not a submodule.
func (r *PyResolver) ResolveImport(importerPath string, imp Import) Resolution {
	if r == nil || r.FS == nil {
		return Resolution{Reason: "nil resolver"}
	}

	dotted, err := r.dottedFor(importerPath, imp)
	if err != nil {
		return Resolution{Reason: err.Error()}
	}

	roots := r.SearchRoots
	if len(roots) == 0 {
		roots = []string{"."}
	}

	if imp.Name != "" && imp.Name != "*" {
		// Try submodule first.
		submodule := joinDotted(dotted, imp.Name)
		if res := r.lookupAcrossRoots(submodule, roots); res.Kind != ResolutionUnresolved {
			return res
		}
		// Fall back to the package/module itself.
		if dotted != "" {
			if res := r.lookupAcrossRoots(dotted, roots); res.Kind != ResolutionUnresolved {
				return res
			}
		}
		return Resolution{Reason: fmt.Sprintf("from %q import %q: no matching file under %v", dotted, imp.Name, roots)}
	}

	if dotted == "" {
		return Resolution{Reason: "empty module path"}
	}
	if res := r.lookupAcrossRoots(dotted, roots); res.Kind != ResolutionUnresolved {
		return res
	}
	return Resolution{Reason: fmt.Sprintf("module %q: no matching file under %v", dotted, roots)}
}

// dottedFor turns an Import into its absolute dotted form relative to the
// search roots. For absolute imports this is simply imp.Module. For
// relative imports it walks up importerPath by imp.Relative levels and
// appends imp.Module.
func (r *PyResolver) dottedFor(importerPath string, imp Import) (string, error) {
	if imp.Relative == 0 {
		return imp.Module, nil
	}
	if importerPath == "" {
		return "", fmt.Errorf("relative import requires importer path")
	}

	importer := path.Clean(importerPath)
	stripped := strings.TrimSuffix(importer, ".py")
	if stripped == importer {
		return "", fmt.Errorf("importer %q is not a .py file", importerPath)
	}
	// Drop the trailing segment in either case: for non-__init__ files
	// it's the module name, for __init__ files it's "__init__". The
	// remaining parts are the importer's package path.
	parts := strings.Split(stripped, "/")
	parts = parts[:len(parts)-1]

	// One dot means "this package" (no further pops); each extra dot
	// pops one more directory.
	pop := imp.Relative - 1
	if pop > len(parts) {
		return "", fmt.Errorf("relative import level %d exceeds package depth", imp.Relative)
	}
	parts = parts[:len(parts)-pop]

	if imp.Module != "" {
		parts = append(parts, strings.Split(imp.Module, ".")...)
	}
	return strings.Join(parts, "."), nil
}

// lookupAcrossRoots tries to resolve a dotted module path under each
// search root in order and returns the first hit.
func (r *PyResolver) lookupAcrossRoots(dotted string, roots []string) Resolution {
	for _, root := range roots {
		if res := r.lookupUnderRoot(dotted, root); res.Kind != ResolutionUnresolved {
			return res
		}
	}
	return Resolution{}
}

func (r *PyResolver) lookupUnderRoot(dotted, root string) Resolution {
	if dotted == "" {
		return Resolution{}
	}
	rel := strings.ReplaceAll(dotted, ".", "/")
	candidates := []struct {
		path string
		kind ResolutionKind
	}{
		{joinPath(root, rel+".py"), ResolutionModule},
		{joinPath(root, rel, "__init__.py"), ResolutionPackage},
	}
	for _, c := range candidates {
		if exists(r.FS, c.path) {
			return Resolution{Kind: c.kind, Path: c.path}
		}
	}
	return Resolution{}
}

func exists(fsys fs.FS, p string) bool {
	info, err := fs.Stat(fsys, p)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func joinDotted(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func joinPath(root string, parts ...string) string {
	all := append([]string{root}, parts...)
	cleaned := path.Join(all...)
	return strings.TrimPrefix(cleaned, "./")
}
