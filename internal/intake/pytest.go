package intake

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ParsePytestID parses a pytest test ID into a Seed. Accepted forms:
//
//	tests/test_foo.py::test_function
//	tests/test_foo.py::TestClass::test_method
//	tests/test_foo.py::TestClass::test_method[param-id]
//
// The path component is kept verbatim (the caller decides how to interpret
// relative vs absolute). Parameter IDs are stored in Seed.Param.
func ParsePytestID(id string) (Seed, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Seed{}, fmt.Errorf("empty pytest test ID")
	}
	parts := strings.Split(id, "::")
	if len(parts) < 2 {
		return Seed{}, fmt.Errorf("pytest test ID %q: expected at least one '::' separator", id)
	}

	path := parts[0]
	if path == "" {
		return Seed{}, fmt.Errorf("pytest test ID %q: missing file path", id)
	}
	if !strings.HasSuffix(path, ".py") {
		return Seed{}, fmt.Errorf("pytest test ID %q: path %q must end in .py", id, path)
	}

	seed := Seed{Kind: KindPytest, File: path}

	switch len(parts) {
	case 2:
		seed.Symbol = parts[1]
	case 3:
		seed.Class = parts[1]
		seed.Symbol = parts[2]
	default:
		return Seed{}, fmt.Errorf("pytest test ID %q: too many '::' segments", id)
	}

	if seed.Symbol == "" {
		return Seed{}, fmt.Errorf("pytest test ID %q: missing function or method name", id)
	}
	if seed.Class != "" && !isIdent(seed.Class) {
		return Seed{}, fmt.Errorf("pytest test ID %q: invalid class name %q", id, seed.Class)
	}

	if i := strings.IndexByte(seed.Symbol, '['); i >= 0 {
		if !strings.HasSuffix(seed.Symbol, "]") {
			return Seed{}, fmt.Errorf("pytest test ID %q: unterminated parametrize bracket", id)
		}
		seed.Param = seed.Symbol[i+1 : len(seed.Symbol)-1]
		seed.Symbol = seed.Symbol[:i]
	}
	if !isIdent(seed.Symbol) {
		return Seed{}, fmt.Errorf("pytest test ID %q: invalid symbol name %q", id, seed.Symbol)
	}

	return seed, nil
}

// Resolve verifies that the seed's file exists under repoRoot and rewrites
// Seed.File to the cleaned path relative to repoRoot. The repoRoot itself is
// resolved against fsys; pass os.DirFS(root) for the on-disk case.
func Resolve(fsys fs.FS, seed Seed) (Seed, error) {
	clean := filepath.ToSlash(filepath.Clean(seed.File))
	if strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(clean) {
		return seed, fmt.Errorf("intake: file %q escapes repository root", seed.File)
	}
	info, err := fs.Stat(fsys, clean)
	if err != nil {
		return seed, fmt.Errorf("intake: %w", err)
	}
	if info.IsDir() {
		return seed, fmt.Errorf("intake: file %q is a directory", clean)
	}
	seed.File = clean
	return seed, nil
}

// ResolveOnDisk is a convenience wrapper that builds an os.DirFS from
// repoRoot. Returns the absolute repoRoot alongside the resolved seed so
// downstream phases can keep both forms.
func ResolveOnDisk(repoRoot string, seed Seed) (Seed, string, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return seed, "", fmt.Errorf("intake: resolve repo root: %w", err)
	}
	if info, err := os.Stat(abs); err != nil {
		return seed, "", fmt.Errorf("intake: repo root: %w", err)
	} else if !info.IsDir() {
		return seed, "", fmt.Errorf("intake: repo root %q is not a directory", abs)
	}
	resolved, err := Resolve(os.DirFS(abs), seed)
	return resolved, abs, err
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}
