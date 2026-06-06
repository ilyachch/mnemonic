# `internal/project`

## What this package owns

This package owns project schemas and project context resolution. It parses and validates `.mnemonic` and `mnemonic.toml`, initializes and imports projects, discovers manifests, resolves the active project from selector or current directory, and computes note-root paths for supported project kinds.

## What this package does not own

This package does not own global config loading, XDG path resolution, registry persistence, note file mutation, index rebuilding, or MCP transport concerns. It should describe projects, not execute note/index workflows directly.

## Important invariants

- `.mnemonic` and `mnemonic.toml` require explicit `version = 1`.
- `.mnemonic` supports `local` and `regular`; `mnemonic.toml` supports `regular` and `detached`.
- Project resolution order is CLI selector, `MNEMONIC_PROJECT`, then nearest `.mnemonic`.
- No implicit default project is allowed when resolution is absent or ambiguous.
- Project path helpers must preserve the distinction between repository-rooted and memories-home-rooted projects.

## Tests to update when changing this package

- `internal/project/mnemonic_file_test.go`
- `internal/project/manifest_test.go`
- `internal/project/resolve_test.go`
- `internal/project/find_test.go`
- `internal/project/init_test.go`
- `internal/project/import_test.go`
- `internal/project/discover_test.go`
- `internal/cli/*project*_test.go` when command behavior depends on project resolution
