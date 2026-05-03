package slicer

import (
	"context"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/kaeawc/reprobundle/internal/scanner"
)

func newFS(files map[string]string) fstest.MapFS {
	out := fstest.MapFS{}
	for path, body := range files {
		out[path] = &fstest.MapFile{Data: []byte(body)}
	}
	return out
}

func TestWalkFilesLinear(t *testing.T) {
	fsys := newFS(map[string]string{
		"entry.py":   "import a\n",
		"a.py":       "from b import thing\n",
		"b.py":       "x = 1\n",
		"unused.py":  "x = 1\n",
	})
	got, err := WalkFiles(context.Background(), scanner.NewPyResolver(fsys), "entry.py")
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	want := []string{"a.py", "b.py", "entry.py"}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %v, want %v", got.Files, want)
	}
	if len(got.External) != 0 {
		t.Fatalf("External = %v, want []", got.External)
	}
}

func TestWalkFilesPackages(t *testing.T) {
	fsys := newFS(map[string]string{
		"entry.py":             "from pkg import sub\n",
		"pkg/__init__.py":      "",
		"pkg/sub/__init__.py":  "from . import helper\n",
		"pkg/sub/helper.py":    "x = 1\n",
	})
	got, err := WalkFiles(context.Background(), scanner.NewPyResolver(fsys), "entry.py")
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	// pkg/__init__.py is included as a parent __init__ of pkg/sub.
	want := []string{"entry.py", "pkg/__init__.py", "pkg/sub/__init__.py", "pkg/sub/helper.py"}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %v, want %v", got.Files, want)
	}
}

func TestWalkFilesIncludesParentInits(t *testing.T) {
	fsys := newFS(map[string]string{
		"entry.py":              "import pkg.sub.deep\n",
		"pkg/__init__.py":       "",
		"pkg/sub/__init__.py":   "",
		"pkg/sub/deep/__init__.py": "",
	})
	got, err := WalkFiles(context.Background(), scanner.NewPyResolver(fsys), "entry.py")
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	want := []string{"entry.py", "pkg/__init__.py", "pkg/sub/__init__.py", "pkg/sub/deep/__init__.py"}
	if !reflect.DeepEqual(got.Files, want) {
		t.Fatalf("Files = %v, want %v", got.Files, want)
	}
}

func TestWalkFilesCycle(t *testing.T) {
	fsys := newFS(map[string]string{
		"a.py": "import b\n",
		"b.py": "import a\n",
	})
	got, err := WalkFiles(context.Background(), scanner.NewPyResolver(fsys), "a.py")
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	if !reflect.DeepEqual(got.Files, []string{"a.py", "b.py"}) {
		t.Fatalf("Files = %v, want [a.py b.py]", got.Files)
	}
}

func TestWalkFilesRecordsExternal(t *testing.T) {
	fsys := newFS(map[string]string{
		"entry.py": "import os\nimport requests\nfrom mypkg.local import thing\n",
		"mypkg/__init__.py":   "",
		"mypkg/local.py":      "x = 1\n",
	})
	got, err := WalkFiles(context.Background(), scanner.NewPyResolver(fsys), "entry.py")
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	if !reflect.DeepEqual(got.Files, []string{"entry.py", "mypkg/__init__.py", "mypkg/local.py"}) {
		t.Fatalf("Files = %v", got.Files)
	}
	if !reflect.DeepEqual(got.External, []string{"os", "requests"}) {
		t.Fatalf("External = %v, want [os requests]", got.External)
	}
}

func TestWalkFilesEntryMissing(t *testing.T) {
	_, err := WalkFiles(context.Background(), scanner.NewPyResolver(newFS(nil)), "nope.py")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestWalkFilesNilGuards(t *testing.T) {
	if _, err := WalkFiles(context.Background(), nil, "a.py"); err == nil {
		t.Fatal("expected error for nil resolver")
	}
	if _, err := WalkFiles(context.Background(), scanner.NewPyResolver(newFS(nil)), ""); err == nil {
		t.Fatal("expected error for empty entry")
	}
}

func TestWalkFilesContextCancel(t *testing.T) {
	fsys := newFS(map[string]string{"a.py": "import b\n", "b.py": "import a\n"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := WalkFiles(ctx, scanner.NewPyResolver(fsys), "a.py"); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestExternalNameForms(t *testing.T) {
	cases := []struct {
		imp  scanner.Import
		want string
	}{
		{scanner.Import{Module: "os"}, "os"},
		{scanner.Import{Module: "collections", Name: "deque"}, "collections.deque"},
		{scanner.Import{Module: "foo", Name: "*"}, "foo"},
		{scanner.Import{Module: "pkg", Name: "x", Relative: 2}, "..pkg.x"},
		{scanner.Import{Name: "x", Relative: 1}, ".x"},
		{scanner.Import{Relative: 1}, ""},
	}
	for _, tc := range cases {
		if got := externalName(tc.imp); got != tc.want {
			t.Errorf("externalName(%+v) = %q, want %q", tc.imp, got, tc.want)
		}
	}
}
