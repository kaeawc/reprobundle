package cli

import (
	"bytes"
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
