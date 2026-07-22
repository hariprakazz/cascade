# cascade

Cascade tells you which packages in your monorepo actually need rebuilding after a change.

Most CI pipelines rebuild everything on every push. That's slow and wasteful.
Cascade builds a real dependency graph from your code and traces which packages
are truly affected — nothing more.

## how it works

change a file → cascade maps it to a package → walks the real dependency graph
(parsed from actual imports, not hand-maintained config) → outputs only the
packages affected, directly or transitively

## supported languages

- Go — parses real imports via `go/parser`, respects build constraints (`//go:build`),
  excludes `_test.go` files from the production dependency graph
- Rust — parses `Cargo.toml` `[dependencies]`

more languages planned (see roadmap below)

## usage

```bash
go run main.go --base=main
go run main.go --base=main --head=HEAD --format=json
```

flags:
- `--base` — base ref to diff against (default `HEAD~1`)
- `--head` — head ref (default `HEAD`)
- `--format` — `plain` (default) or `json`

## example output
## status

Core dependency-graph engine is real and tested — 10 passing unit tests cover
graph traversal (linear chains, cycles, disconnected packages), import parsing,
build-constraint filtering, and test-file exclusion. Building this in public
alongside my GSoC 2026 work on the D language's build tooling.

## roadmap

- [x] git diff-based change detection
- [x] real dependency graph from Go imports (not hand-maintained config)
- [x] build-constraint-aware parsing
- [x] Rust (`Cargo.toml`) support
- [ ] D (`dub.json`) support
- [ ] benchmark against Nx/Turborepo on a real monorepo
- [ ] GitHub Actions integration (`--format=github-matrix`)
- [ ] `--visualize` — print the dependency tree
