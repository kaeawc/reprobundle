# reprobundle

Go-first static slicer that turns a failing eval or bug into a minimal repro
directory. See `README.md` for the product spec; this file covers conventions.

Architecture mirrors [krit](https://github.com/kaeawc/krit): tree-sitter parse,
single-pass dispatch, capability-gated cross-file analysis. Read
`~/kaeawc/krit/CLAUDE.md` for the patterns reprobundle generalizes.

## Key Rules

- Keep all analyzer and slicer work in Go.
- After implementation changes, run `go build -o reprobundle ./cmd/reprobundle/ && go vet ./...`.
- Run `go test ./... -count=1` for full validation.
- Use tree-sitter AST/flat nodes for structural analysis; reach for regex only
  for line-oriented checks.
- Phases that need cross-file context declare a capability
  (`NeedsCallGraph`, `NeedsRuntimeTrace`, `NeedsDataExtractor`) so the
  dispatcher can build the matching index once.

## Project Structure

- `cmd/reprobundle/` — CLI entry point.
- `internal/cli/` — flag parsing and the top-level Run function.
- `internal/intake/` — entry-point intake (pytest test ID, agent class, JSONL trace).
- `internal/scanner/` — tree-sitter parsing and flat-AST helpers.
- `internal/slicer/` — call-graph walker and the code/prompt/config slices.
- `internal/resolver/` — static resource-path resolution; runtime fallback.
- `internal/bundler/` — file copying, `repro.sh` emission, `MANIFEST.md`.
- `tests/fixtures/` — input repos with expected slice outputs.

## Build & Validate

```bash
make build      # go build -o reprobundle ./cmd/reprobundle/
make vet        # go vet ./...
make test       # go test ./... -count=1
make ci         # build + vet + test
```
