package scanner

import (
	"testing"
	"testing/fstest"
)

func mapFS(files ...string) fstest.MapFS {
	out := fstest.MapFS{}
	for _, f := range files {
		out[f] = &fstest.MapFile{Data: []byte("")}
	}
	return out
}

func TestResolveAbsoluteModule(t *testing.T) {
	r := NewPyResolver(mapFS("pkg/sub/x.py", "pkg/__init__.py", "pkg/sub/__init__.py"))
	got := r.ResolveImport("entry.py", Import{Module: "pkg.sub.x"})
	if got.Kind != ResolutionModule || got.Path != "pkg/sub/x.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveAbsolutePackage(t *testing.T) {
	r := NewPyResolver(mapFS("pkg/__init__.py", "pkg/sub/__init__.py"))
	got := r.ResolveImport("entry.py", Import{Module: "pkg.sub"})
	if got.Kind != ResolutionPackage || got.Path != "pkg/sub/__init__.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveFromImportPrefersSubmodule(t *testing.T) {
	// from pkg import sub -> pkg/sub.py wins over pkg/__init__.py.
	r := NewPyResolver(mapFS("pkg/__init__.py", "pkg/sub.py"))
	got := r.ResolveImport("entry.py", Import{Module: "pkg", Name: "sub"})
	if got.Kind != ResolutionModule || got.Path != "pkg/sub.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveFromImportFallsBackToPackage(t *testing.T) {
	// from pkg import some_name -> some_name not a submodule, so resolve to pkg.
	r := NewPyResolver(mapFS("pkg/__init__.py"))
	got := r.ResolveImport("entry.py", Import{Module: "pkg", Name: "some_name"})
	if got.Kind != ResolutionPackage || got.Path != "pkg/__init__.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveRelativeOneDotBare(t *testing.T) {
	// from . import sibling, importer at pkg/mod.py -> pkg/sibling.py
	r := NewPyResolver(mapFS("pkg/__init__.py", "pkg/mod.py", "pkg/sibling.py"))
	got := r.ResolveImport("pkg/mod.py", Import{Name: "sibling", Relative: 1})
	if got.Kind != ResolutionModule || got.Path != "pkg/sibling.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveRelativeFromInit(t *testing.T) {
	// from . import x, importer at pkg/__init__.py -> pkg/x.py
	r := NewPyResolver(mapFS("pkg/__init__.py", "pkg/x.py"))
	got := r.ResolveImport("pkg/__init__.py", Import{Name: "x", Relative: 1})
	if got.Kind != ResolutionModule || got.Path != "pkg/x.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveRelativeTwoDots(t *testing.T) {
	// from ..util import helper, importer at pkg/sub/mod.py -> pkg/util/helper.py
	r := NewPyResolver(mapFS(
		"pkg/__init__.py",
		"pkg/sub/__init__.py",
		"pkg/sub/mod.py",
		"pkg/util/__init__.py",
		"pkg/util/helper.py",
	))
	got := r.ResolveImport("pkg/sub/mod.py", Import{Module: "util", Name: "helper", Relative: 2})
	if got.Kind != ResolutionModule || got.Path != "pkg/util/helper.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveRelativeOverflow(t *testing.T) {
	r := NewPyResolver(mapFS("pkg/mod.py"))
	got := r.ResolveImport("pkg/mod.py", Import{Name: "x", Relative: 5})
	if got.Kind != ResolutionUnresolved || got.Reason == "" {
		t.Fatalf("expected unresolved with reason, got %+v", got)
	}
}

func TestResolveCustomSearchRoot(t *testing.T) {
	// Common src/ layout.
	r := &PyResolver{FS: mapFS("src/pkg/__init__.py", "src/pkg/x.py"), SearchRoots: []string{"src"}}
	got := r.ResolveImport("src/pkg/x.py", Import{Module: "pkg.x"})
	if got.Kind != ResolutionModule || got.Path != "src/pkg/x.py" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveSearchRootOrdering(t *testing.T) {
	r := &PyResolver{
		FS:          mapFS("first/m.py", "second/m.py"),
		SearchRoots: []string{"first", "second"},
	}
	got := r.ResolveImport("entry.py", Import{Module: "m"})
	if got.Path != "first/m.py" {
		t.Fatalf("expected first root to win, got %+v", got)
	}
}

func TestResolveUnresolved(t *testing.T) {
	r := NewPyResolver(mapFS("pkg/__init__.py"))
	got := r.ResolveImport("entry.py", Import{Module: "missing"})
	if got.Kind != ResolutionUnresolved {
		t.Fatalf("expected unresolved, got %+v", got)
	}
	if got.Reason == "" {
		t.Fatal("expected non-empty Reason on unresolved")
	}
}

func TestResolveNilGuards(t *testing.T) {
	var r *PyResolver
	if got := r.ResolveImport("a.py", Import{Module: "x"}); got.Kind != ResolutionUnresolved {
		t.Fatalf("nil receiver should return unresolved, got %+v", got)
	}
	r2 := &PyResolver{}
	if got := r2.ResolveImport("a.py", Import{Module: "x"}); got.Kind != ResolutionUnresolved {
		t.Fatalf("nil FS should return unresolved, got %+v", got)
	}
}

func TestResolutionKindString(t *testing.T) {
	cases := map[ResolutionKind]string{
		ResolutionUnresolved: "unresolved",
		ResolutionModule:     "module",
		ResolutionPackage:    "package",
		ResolutionExternal:   "external",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", k, got, want)
		}
	}
}
