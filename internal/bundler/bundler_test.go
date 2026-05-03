package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kaeawc/reprobundle/internal/intake"
	"github.com/kaeawc/reprobundle/internal/slicer"
)

func newSrc(files map[string]string) fstest.MapFS {
	out := fstest.MapFS{}
	for p, body := range files {
		out[p] = &fstest.MapFile{Data: []byte(body)}
	}
	return out
}

func TestWriteCopiesFilesAndPreservesLayout(t *testing.T) {
	src := newSrc(map[string]string{
		"tests/test_x.py":   "def test_x(): pass\n",
		"myapp/__init__.py": "",
		"myapp/core.py":     "def run(): pass\n",
	})
	out := t.TempDir()

	b := FromWalk(src, "/repo", intake.Seed{
		Kind: intake.KindPytest, File: "tests/test_x.py", Symbol: "test_x",
	}, slicer.FileSet{
		Files:    []string{"myapp/__init__.py", "myapp/core.py", "tests/test_x.py"},
		External: []string{"os"},
	})

	if err := Write(out, b); err != nil {
		t.Fatalf("Write: %v", err)
	}

	for path, want := range map[string]string{
		"tests/test_x.py":   "def test_x(): pass\n",
		"myapp/__init__.py": "",
		"myapp/core.py":     "def run(): pass\n",
	} {
		got, err := os.ReadFile(filepath.Join(out, path))
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestWriteEmitsManifest(t *testing.T) {
	src := newSrc(map[string]string{
		"a/b/__init__.py": "",
		"a/b/x.py":        "x=1\n",
	})
	out := t.TempDir()
	b := FromWalk(src, "/repo", intake.Seed{
		Kind: intake.KindPytest, File: "a/b/x.py", Class: "TestX", Symbol: "test_y", Param: "case-1",
	}, slicer.FileSet{
		Files:    []string{"a/b/__init__.py", "a/b/x.py"},
		External: []string{"requests"},
	})

	if err := Write(out, b); err != nil {
		t.Fatalf("Write: %v", err)
	}

	manifest, err := os.ReadFile(filepath.Join(out, "MANIFEST.md"))
	if err != nil {
		t.Fatalf("read MANIFEST.md: %v", err)
	}
	got := string(manifest)
	for _, want := range []string{
		"# Reprobundle manifest",
		"Source repo: /repo",
		"Entry: a/b/x.py::TestX::test_y[case-1]",
		"## Files (2)",
		"`a/b/__init__.py` — parent package __init__",
		"`a/b/x.py` — entry point",
		"## External (1)",
		"`requests`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("manifest missing %q\n--- manifest ---\n%s", want, got)
		}
	}
}

func TestWriteEmitsExecutableReproScript(t *testing.T) {
	src := newSrc(map[string]string{"t.py": ""})
	out := t.TempDir()
	b := FromWalk(src, "/repo", intake.Seed{
		Kind: intake.KindPytest, File: "t.py", Symbol: "test_x",
	}, slicer.FileSet{Files: []string{"t.py"}})

	if err := Write(out, b); err != nil {
		t.Fatalf("Write: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(out, "repro.sh"))
	if err != nil {
		t.Fatalf("read repro.sh: %v", err)
	}
	if !strings.Contains(string(body), `pytest -x "t.py::test_x"`) {
		t.Errorf("repro.sh missing pytest invocation:\n%s", body)
	}
	info, err := os.Stat(filepath.Join(out, "repro.sh"))
	if err != nil {
		t.Fatalf("stat repro.sh: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("repro.sh is not executable: %v", info.Mode())
	}
}

func TestPytestTargetClassAndParam(t *testing.T) {
	cases := []struct {
		seed intake.Seed
		want string
	}{
		{intake.Seed{File: "t.py", Symbol: "test_x"}, "t.py::test_x"},
		{intake.Seed{File: "t.py", Class: "TestX", Symbol: "test_y"}, "t.py::TestX::test_y"},
		{intake.Seed{File: "t.py", Symbol: "test_x", Param: "1-foo"}, "t.py::test_x[1-foo]"},
		{intake.Seed{File: "t.py", Class: "TestX", Symbol: "test_y", Param: "p"}, "t.py::TestX::test_y[p]"},
	}
	for _, tc := range cases {
		if got := pytestTarget(tc.seed); got != tc.want {
			t.Errorf("pytestTarget(%+v) = %q, want %q", tc.seed, got, tc.want)
		}
	}
}

func TestWriteRejectsBadInputs(t *testing.T) {
	src := newSrc(map[string]string{"x.py": ""})
	cases := []struct {
		name string
		out  string
		b    Bundle
	}{
		{"empty out dir", "", Bundle{SrcFS: src, Files: []string{"x.py"}}},
		{"nil src fs", t.TempDir(), Bundle{Files: []string{"x.py"}}},
		{"empty files", t.TempDir(), Bundle{SrcFS: src}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Write(tc.out, tc.b); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestWriteRejectsPathEscape(t *testing.T) {
	src := newSrc(map[string]string{"x.py": ""})
	out := t.TempDir()
	b := Bundle{SrcFS: src, Files: []string{"../etc/passwd"}, Seed: intake.Seed{File: "x.py", Symbol: "t"}}
	if err := Write(out, b); err == nil {
		t.Fatal("expected error for path escape")
	}
}
