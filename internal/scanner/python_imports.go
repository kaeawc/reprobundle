package scanner

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Import is one bound name produced by a Python import statement.
//
// Each comma-separated clause becomes its own Import. So
// `from collections import OrderedDict, deque as dq` produces two records,
// both with Module="collections", Relative=0, distinguished by Name/Alias.
type Import struct {
	// Module is the dotted module path from the source side of the
	// statement, after stripping any leading dots for relative imports.
	// For `from . import x` this is the empty string.
	Module string

	// Name is the symbol pulled out by a from-import. Empty for plain
	// `import X` or `import X.Y` statements (use Module instead).
	Name string

	// Alias is the local binding name, or empty if no alias was used.
	Alias string

	// Relative is the leading-dot count for relative imports.
	// 0 means absolute, 1 means `.`, 2 means `..`, and so on.
	Relative int

	// StartByte and EndByte point at the originating statement node so
	// downstream tools can report diagnostics with source ranges.
	StartByte uint32
	EndByte   uint32
}

// LocalName returns the name the import binds in the importing module.
// For `import x.y` it's "x"; for `import x.y as z` it's "z"; for
// `from a import b` it's "b"; for `from a import b as c` it's "c".
func (i Import) LocalName() string {
	if i.Alias != "" {
		return i.Alias
	}
	if i.Name != "" {
		return i.Name
	}
	if dot := strings.IndexByte(i.Module, '.'); dot >= 0 {
		return i.Module[:dot]
	}
	return i.Module
}

// Imports walks a parsed Python file and returns every binding produced by
// its top-level import statements. Imports nested inside functions, classes,
// or conditional blocks are included as well — the slicer treats every
// reachable import as live.
func Imports(f *File) ([]Import, error) {
	if f == nil {
		return nil, fmt.Errorf("scanner.Imports: nil file")
	}
	if f.Language != LangPython {
		return nil, fmt.Errorf("scanner.Imports: %s is not a Python file", f.Path)
	}
	root := f.Root()
	if root == nil {
		return nil, nil
	}
	var out []Import
	walk(root, f.Source, &out)
	return out, nil
}

func walk(n *sitter.Node, src []byte, out *[]Import) {
	switch n.Type() {
	case "import_statement":
		collectImport(n, src, out)
		return
	case "import_from_statement":
		collectImportFrom(n, src, out)
		return
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		walk(n.NamedChild(i), src, out)
	}
}

func collectImport(n *sitter.Node, src []byte, out *[]Import) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "dotted_name":
			*out = append(*out, Import{
				Module:    child.Content(src),
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		case "aliased_import":
			module, alias := splitAliased(child, src)
			*out = append(*out, Import{
				Module:    module,
				Alias:     alias,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		}
	}
}

func collectImportFrom(n *sitter.Node, src []byte, out *[]Import) {
	relative := 0
	module := ""
	source := n.NamedChild(0)
	if source == nil {
		return
	}
	switch source.Type() {
	case "dotted_name":
		module = source.Content(src)
	case "relative_import":
		relative, module = parseRelative(source, src)
	default:
		return
	}

	for i := 1; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "dotted_name":
			*out = append(*out, Import{
				Module:    module,
				Name:      child.Content(src),
				Relative:  relative,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		case "aliased_import":
			name, alias := splitAliased(child, src)
			*out = append(*out, Import{
				Module:    module,
				Name:      name,
				Alias:     alias,
				Relative:  relative,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		case "wildcard_import":
			*out = append(*out, Import{
				Module:    module,
				Name:      "*",
				Relative:  relative,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		}
	}
}

func parseRelative(n *sitter.Node, src []byte) (int, string) {
	level := 0
	module := ""
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "import_prefix":
			level = len(child.Content(src))
		case "dotted_name":
			module = child.Content(src)
		}
	}
	return level, module
}

func splitAliased(n *sitter.Node, src []byte) (module, alias string) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "dotted_name":
			module = child.Content(src)
		case "identifier":
			alias = child.Content(src)
		}
	}
	return
}
