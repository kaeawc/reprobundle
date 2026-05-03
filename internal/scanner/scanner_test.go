package scanner

import (
	"context"
	"testing"
)

func TestParsePythonSmoke(t *testing.T) {
	src := []byte("def hello(name):\n    return 'hi ' + name\n")
	f, err := ParsePython(context.Background(), "hello.py", src)
	if err != nil {
		t.Fatalf("ParsePython: %v", err)
	}
	defer f.Close()

	root := f.Root()
	if root == nil {
		t.Fatal("root node is nil")
	}
	if root.Type() != "module" {
		t.Fatalf("root.Type() = %q, want %q", root.Type(), "module")
	}
	if root.HasError() {
		t.Fatalf("parse produced ERROR nodes: %s", root.String())
	}
	if got, want := f.Language.String(), "python"; got != want {
		t.Fatalf("Language = %q, want %q", got, want)
	}
}

func TestParsePythonReportsSyntaxErrors(t *testing.T) {
	src := []byte("def broken(:\n")
	f, err := ParsePython(context.Background(), "broken.py", src)
	if err != nil {
		t.Fatalf("ParsePython returned error on recoverable syntax: %v", err)
	}
	defer f.Close()
	if !f.Root().HasError() {
		t.Fatal("expected HasError() on malformed input")
	}
}
