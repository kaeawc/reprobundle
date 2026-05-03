package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"--version"}, "v0.0.1", &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "v0.0.1" {
		t.Fatalf("stdout = %q, want %q", got, "v0.0.1")
	}
}

func TestRequiresEntryAndOut(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(nil, "dev", &stdout, &stderr); err == nil {
		t.Fatal("expected error when --entry and --out are missing")
	}
}

func TestRunSliceEndToEnd(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "tests/test_thing.py", "import myapp.core\n\ndef test_thing():\n    myapp.core.run()\n")
	mustWrite(t, root, "myapp/__init__.py", "")
	mustWrite(t, root, "myapp/core.py", "import os\nfrom myapp.helper import h\n\ndef run():\n    h()\n")
	mustWrite(t, root, "myapp/helper.py", "def h():\n    pass\n")
	mustWrite(t, root, "unrelated.py", "x = 1\n")

	var stdout, stderr bytes.Buffer
	args := []string{
		"--repo", root,
		"--entry", "tests/test_thing.py::test_thing",
		"--out", filepath.Join(root, "out"),
	}
	if err := Run(args, "dev", &stdout, &stderr); err != nil {
		t.Fatalf("Run: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"entry: tests/test_thing.py :: test_thing",
		"files (4):",
		"myapp/__init__.py",
		"myapp/core.py",
		"myapp/helper.py",
		"tests/test_thing.py",
		"external (1):",
		"  os",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q\n--- output ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "unrelated.py") {
		t.Errorf("stdout should not list unrelated.py:\n%s", out)
	}
}

func TestRunSliceBadEntry(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := Run([]string{"--repo", root, "--entry", "no-separator", "--out", root}, "dev", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for malformed entry")
	}
	if !strings.Contains(err.Error(), "parse entry") {
		t.Fatalf("error %q does not mention parse entry", err)
	}
}

func TestRunSliceMissingFile(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := Run([]string{"--repo", root, "--entry", "missing.py::test_x", "--out", root}, "dev", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing entry file")
	}
	if !strings.Contains(err.Error(), "resolve seed") {
		t.Fatalf("error %q does not mention resolve seed", err)
	}
}

func mustWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}
