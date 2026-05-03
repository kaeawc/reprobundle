// Package scanner parses source files with tree-sitter and exposes the flat
// AST helpers the slicer and resolver consume. Mirrors krit/internal/scanner
// in shape; reprobundle starts with Python and adds TypeScript/Go later.
package scanner

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// Language identifies which tree-sitter grammar parsed a file.
type Language int

const (
	LangUnknown Language = iota
	LangPython
)

func (l Language) String() string {
	switch l {
	case LangPython:
		return "python"
	default:
		return "unknown"
	}
}

// File is a parsed source file. The tree must be Closed when the caller is
// done with it.
type File struct {
	Path     string
	Source   []byte
	Language Language
	Tree     *sitter.Tree
}

// Close releases the tree-sitter tree.
func (f *File) Close() {
	if f.Tree != nil {
		f.Tree.Close()
		f.Tree = nil
	}
}

// Root returns the root node of the parsed tree.
func (f *File) Root() *sitter.Node {
	if f.Tree == nil {
		return nil
	}
	return f.Tree.RootNode()
}

// ParsePython parses src as Python. The returned File owns the tree.
func ParsePython(ctx context.Context, path string, src []byte) (*File, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, err := parser.ParseCtx(ctx, nil, src)
	if err != nil {
		return nil, fmt.Errorf("parse python %s: %w", path, err)
	}
	return &File{
		Path:     path,
		Source:   src,
		Language: LangPython,
		Tree:     tree,
	}, nil
}
