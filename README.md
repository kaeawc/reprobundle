# reprobundle — a static slicer that turns a failing eval or bug into a minimal repro

## What you're building

When an eval regresses or a customer reports a bad agent trace, the first hour of debugging is always the same: "what's the minimum set of code, prompts, configs, and inputs needed to reproduce this?" The answer is almost always discoverable from the source tree, but humans do it by hand. reprobundle does it statically.

Given (a) a failing eval ID or a saved trace, and (b) a repo, produce a directory containing only the files needed to reproduce: the agent's code, the transitive imports it actually uses, the prompt templates referenced, the tool schemas, the config values read, and a `repro.sh` that runs the failure.

Architecture mirrors Krit (Go-first, tree-sitter, capability-gated cross-file analysis, single-pass dispatch). Read `~/kaeawc/krit/CLAUDE.md` first — the dead-code detector and module/dependency graph are exactly what this tool generalizes.

## Core idea

Three slices, intersected:

1. **Code slice** — start from the eval's entry point or the trace's top frame, walk the call graph, keep only reachable functions/classes/modules. Same machinery as Krit's cross-file reference index, but in "keep" mode rather than "delete" mode.
2. **Prompt/tool slice** — every prompt template, tool schema, and agent definition the code slice references.
3. **Config/data slice** — every config key read by the code slice; every dataset/file path opened. For data files, optionally extract just the rows the trace touched.

Output: a self-contained directory + `repro.sh` + a `MANIFEST.md` listing what was included and why.

## Architecture

- **Go**, tree-sitter Python + TypeScript + Go.
- **Entry-point intake** — accept either a pytest test ID, an agent class name, or a JSONL trace file with frame info.
- **Call-graph walker** — bounded interprocedural reachability (subset of policyguard's engine; could share code).
- **Resource resolver** — for each `open(path)`, `load_template(...)`, `read_config(...)` call in the slice, statically resolve the path. Falls back to runtime tracing (capability-gated) when paths are dynamic.
- **Bundler** — copy files preserving the package layout the imports require; rewrite no source.
- **Capability gates** — `NeedsCallGraph` (always), `NeedsRuntimeTrace` (for dynamic paths), `NeedsDataExtractor` (to slice a dataset down to the relevant rows).
- **Outputs**: a directory, a tarball, optionally a Docker image with the venv pinned.

## MVP

1. Skeleton + tree-sitter Python.
2. Pytest entry-point intake.
3. Code slice (call graph) + simple prompt/template detector.
4. Static config-key resolver for one config library.
5. `MANIFEST.md` with file→reason mapping.
6. CI: take a public agent repo with failing tests, slice it, verify the slice still reproduces the failure.

## Stretch

- **Trace-driven mode** — given a saved Anthropic / OpenAI trace, pick out the exact agent, tools, prompts that ran; bundle just those.
- **Dataset slicing** — for a failing eval that uses a 10GB dataset, include only the rows the trace touched.
- **Repro replay** — bundled dir runs identically offline; pin model snapshots if available.
- **Diff between repros** — bundle two repros and statically diff them to highlight the smallest behavioral delta.
- **Public bundle scrubbing** — strip secrets and PII from the bundle so it's safe to share with vendors.

## Why this is the right shape

Repro is the unblocking step in almost every LLM-product debugging session, and it's almost always done manually. The tool that automates it earns trust immediately. The Krit pattern fits because the core engine is "polyglot static dependency analysis with capability-gated extras" — which is exactly Krit, with the output reshaped from "findings" to "a directory."

## Non-goals

- Inferring why the bug happened (different, downstream tool).
- Replaying against production services or stateful APIs.
- Patching the bug — reprobundle isolates, it doesn't fix.
