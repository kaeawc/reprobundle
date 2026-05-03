package scanner

import (
	"context"
	"testing"
)

func parse(t *testing.T, src string) *File {
	t.Helper()
	f, err := ParsePython(context.Background(), "t.py", []byte(src))
	if err != nil {
		t.Fatalf("ParsePython: %v", err)
	}
	t.Cleanup(f.Close)
	return f
}

func TestImports(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Import
	}{
		{
			name: "import simple",
			src:  "import os\n",
			want: []Import{{Module: "os"}},
		},
		{
			name: "import dotted with alias",
			src:  "import os.path as op\n",
			want: []Import{{Module: "os.path", Alias: "op"}},
		},
		{
			name: "import multiple comma",
			src:  "import os, sys as s\n",
			want: []Import{
				{Module: "os"},
				{Module: "sys", Alias: "s"},
			},
		},
		{
			name: "from absolute",
			src:  "from collections import OrderedDict, deque as dq\n",
			want: []Import{
				{Module: "collections", Name: "OrderedDict"},
				{Module: "collections", Name: "deque", Alias: "dq"},
			},
		},
		{
			name: "from relative one dot bare",
			src:  "from . import sibling\n",
			want: []Import{{Module: "", Name: "sibling", Relative: 1}},
		},
		{
			name: "from relative two dots with subpkg",
			src:  "from ..pkg.sub import thing as t\n",
			want: []Import{{Module: "pkg.sub", Name: "thing", Alias: "t", Relative: 2}},
		},
		{
			name: "from with parenthesised list",
			src:  "from typing import (\n    List,\n    Dict as D,\n)\n",
			want: []Import{
				{Module: "typing", Name: "List"},
				{Module: "typing", Name: "Dict", Alias: "D"},
			},
		},
		{
			name: "wildcard",
			src:  "from foo import *\n",
			want: []Import{{Module: "foo", Name: "*"}},
		},
		{
			name: "nested in function still collected",
			src:  "def f():\n    import json\n    return json\n",
			want: []Import{{Module: "json"}},
		},
		{
			name: "no imports",
			src:  "x = 1\n",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := parse(t, tc.src)
			got, err := Imports(f)
			if err != nil {
				t.Fatalf("Imports: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d imports, want %d: %+v", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				g := got[i]
				if g.Module != w.Module || g.Name != w.Name || g.Alias != w.Alias || g.Relative != w.Relative {
					t.Errorf("import[%d] = %+v, want %+v", i, g, w)
				}
				if g.StartByte == g.EndByte {
					t.Errorf("import[%d] has empty source range", i)
				}
			}
		})
	}
}

func TestImportsLocalName(t *testing.T) {
	cases := []struct {
		imp  Import
		want string
	}{
		{Import{Module: "os"}, "os"},
		{Import{Module: "os.path"}, "os"},
		{Import{Module: "os.path", Alias: "op"}, "op"},
		{Import{Module: "collections", Name: "deque"}, "deque"},
		{Import{Module: "collections", Name: "deque", Alias: "dq"}, "dq"},
	}
	for _, tc := range cases {
		if got := tc.imp.LocalName(); got != tc.want {
			t.Errorf("(%+v).LocalName() = %q, want %q", tc.imp, got, tc.want)
		}
	}
}

func TestImportsRejectsNonPython(t *testing.T) {
	if _, err := Imports(nil); err == nil {
		t.Fatal("expected error for nil file")
	}
	f := &File{Path: "x.go", Language: LangUnknown}
	if _, err := Imports(f); err == nil {
		t.Fatal("expected error for non-python file")
	}
}
