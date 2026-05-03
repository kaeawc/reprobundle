package intake

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParsePytestID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Seed
	}{
		{
			name: "module-level function",
			in:   "tests/test_foo.py::test_thing",
			want: Seed{Kind: KindPytest, File: "tests/test_foo.py", Symbol: "test_thing"},
		},
		{
			name: "class method",
			in:   "tests/test_foo.py::TestThing::test_method",
			want: Seed{Kind: KindPytest, File: "tests/test_foo.py", Class: "TestThing", Symbol: "test_method"},
		},
		{
			name: "parametrize ID",
			in:   "tests/test_foo.py::test_thing[case-1]",
			want: Seed{Kind: KindPytest, File: "tests/test_foo.py", Symbol: "test_thing", Param: "case-1"},
		},
		{
			name: "class method with parametrize",
			in:   "pkg/tests/t.py::TestX::test_y[a-b-c]",
			want: Seed{Kind: KindPytest, File: "pkg/tests/t.py", Class: "TestX", Symbol: "test_y", Param: "a-b-c"},
		},
		{
			name: "leading whitespace trimmed",
			in:   "  a.py::t  ",
			want: Seed{Kind: KindPytest, File: "a.py", Symbol: "t"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePytestID(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ParsePytestID(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParsePytestIDErrors(t *testing.T) {
	cases := map[string]string{
		"empty":                "",
		"no separator":         "tests/test_foo.py",
		"missing path":         "::test_foo",
		"non-py path":          "tests/foo.txt::test_foo",
		"missing symbol":       "tests/test_foo.py::",
		"too many segments":    "a.py::B::c::d",
		"unterminated bracket": "a.py::test_x[oops",
		"invalid class":        "a.py::1Bad::test_x",
		"invalid symbol":       "a.py::1bad",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePytestID(in); err == nil {
				t.Fatalf("expected error for %q", in)
			}
		})
	}
}

func TestResolveSuccess(t *testing.T) {
	fsys := fstest.MapFS{
		"tests/test_foo.py": &fstest.MapFile{Data: []byte("def test_thing(): pass\n")},
	}
	seed, err := Resolve(fsys, Seed{Kind: KindPytest, File: "tests/test_foo.py", Symbol: "test_thing"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if seed.File != "tests/test_foo.py" {
		t.Fatalf("File = %q, want %q", seed.File, "tests/test_foo.py")
	}
}

func TestResolveCleansPath(t *testing.T) {
	fsys := fstest.MapFS{
		"tests/test_foo.py": &fstest.MapFile{Data: []byte("x")},
	}
	seed, err := Resolve(fsys, Seed{Kind: KindPytest, File: "tests/./test_foo.py", Symbol: "t"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if seed.File != "tests/test_foo.py" {
		t.Fatalf("File = %q, want cleaned form", seed.File)
	}
}

func TestResolveErrors(t *testing.T) {
	fsys := fstest.MapFS{
		"tests/test_foo.py": &fstest.MapFile{Data: []byte("x")},
		"some/dir/file.py":  &fstest.MapFile{Data: []byte("x")},
	}
	cases := []struct {
		name    string
		seed    Seed
		wantSub string
	}{
		{"missing", Seed{File: "tests/missing.py"}, "does not exist"},
		{"escapes root", Seed{File: "../etc/passwd.py"}, "escapes repository root"},
		{"absolute path", Seed{File: "/etc/passwd.py"}, "escapes repository root"},
		{"is directory", Seed{File: "some/dir"}, "is a directory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(fsys, tc.seed)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err, tc.wantSub)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	if KindPytest.String() != "pytest" {
		t.Fatalf("KindPytest.String() = %q", KindPytest.String())
	}
	if KindUnknown.String() != "unknown" {
		t.Fatalf("KindUnknown.String() = %q", KindUnknown.String())
	}
}
