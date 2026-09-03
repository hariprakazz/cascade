# cascade

Cascade tells you which packages in your monorepo actually need rebuilding after a change.

Most CI pipelines rebuild everything on every push. That's slow and wasteful.
Cascade builds a real dependency graph from your code and traces which packages
are truly affected, nothing more.

## how it works

change a file, cascade maps it to a package, walks the real dependency graph
(parsed from actual imports, not hand-maintained config), outputs only the
packages affected, directly or transitively

## supported languages

- Go, parses real imports via `go/parser`, respects build constraints (`//go:build`),
  excludes `_test.go` files from the production dependency graph
- Rust, parses `Cargo.toml` `[dependencies]`
- D, parses `dub.json` `dependencies`

## correctness

Cascade handles the git edge cases that break naive diff-based detection:

- renamed files (`git diff --name-status`, not `--name-only`, so both old and
  new paths get mapped to their packages)
- merge commits (`--base` defaults to `HEAD~1`, which is ambiguous on a merge
  commit since it has two parents; cascade resolves the actual merge-base
  instead)
- shallow clones (CI checkouts are often `--depth 1`; cascade detects this and
  runs `git fetch --unshallow` automatically before diffing)
- rebases and detached HEAD, verified safe, since a plain `git diff` between
  two resolved commits doesn't care how they got there

The dependency graph is not hardcoded to this repo. `modulePrefix` is derived
from the target repo's own `go.mod`, and `--packages-dir` lets you point
cascade at any repo's actual package layout.

## usage

```bash
go run main.go --base=main
go run main.go --base=main --head=HEAD --format=json
go run main.go --base=main --packages-dir=apps
```

flags:
- `--base`, base ref to diff against (default `HEAD~1`)
- `--head`, head ref (default `HEAD`)
- `--format`, `plain` (default) or `json`
- `--packages-dir`, directory containing packages, relative to repo root (default `packages`)

## benchmark

`benchmark/generate.sh` builds a reproducible 11-package multi-language fixture
repo (Go, Rust, D) with real cross-package dependency chains and real commit
history. Anyone can regenerate it and verify these numbers independently:

```bash
./benchmark/generate.sh /tmp/bench-repo
```

| change | affected | skipped | time |
|---|---|---|---|
| core package (imported by nearly everything) | 9 / 11 | 2 | 7µs |
| leaf + mid-tier package | 3 / 11 | 8 | 6µs |

The point isn't that cascade always skips most packages. It's that it
correctly identifies exactly which ones matter, every time, including the
honest case where a core dependency change legitimately affects almost
everything.

## status

19 passing unit tests cover graph traversal, import parsing across all three
languages, build-constraint filtering, test-file exclusion, and the git edge
cases above. Building this in public alongside my GSoC 2026 work on the D
language's build tooling.

## roadmap

- [x] git diff-based change detection
- [x] real dependency graph from Go imports (not hand-maintained config)
- [x] build-constraint-aware parsing
- [x] Rust (`Cargo.toml`) support
- [x] D (`dub.json`) support
- [x] rename, merge-commit, and shallow-clone correctness
- [x] benchmark against a reproducible multi-language fixture (real public repo matching cascade's model does not exist yet)
- [ ] `--format=github-matrix` and GitHub Actions integration
- [ ] `--visualize`, print the dependency tree
