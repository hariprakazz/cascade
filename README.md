# cascade

Cascade tells you which packages in your monorepo actually need rebuilding after a change.

Most CI pipelines rebuild everything on every push. That's slow and wasteful.
Cascade reads your dependency graph and figures out what's truly affected — nothing more.

## how it works

change one file → cascade traces which packages depend on it → outputs only those

## usage

```bash
go run main.go
```

## example output
affected packages:
auth
api
dashboard

## status

building this in public alongside my GSoC 2026 project. rough edges expected.
