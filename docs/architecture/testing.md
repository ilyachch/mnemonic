# Testing Strategy

## Purpose

The test strategy keeps the repository behavior stable across project resolution, note parsing, indexing, CLI flows, and MCP tools. Tests are primarily package-local Go tests with focused fixtures and deterministic clocks.

## Invariants

- Every behavior change should be covered by nearby package tests.
- Routine verification uses quiet `go test` runs.
- Race testing is reserved for phase boundaries or final verification.
- Tests should prefer deterministic fixtures and helper constructors over ad hoc file layouts when helpers exist.

## Current test layout

- `internal/cli` contains integration-style command tests.
- `internal/project`, `internal/config`, and `internal/registry` cover schema parsing, validation, and migration behavior.
- `internal/notes` covers file lifecycle operations, hashing, locking, and delete modes.
- `internal/markdown` covers frontmatter, tags, observations, relations, wiki-links, and rendering.
- `internal/index`, `internal/search`, and `internal/graph` cover rebuilds and query behavior.
- `internal/mcp` covers stdio transport, tool registration, and tool semantics.

## Common test practices

- Use temporary directories and XDG environment overrides.
- Use `internal/testutil` clocks for deterministic timestamps and UUIDs when the scenario needs stable outputs.
- Verify Markdown content and `mtime` when a feature must not rewrite source notes.
- Treat green `go test` results as authoritative when editor diagnostics are stale.

## Related packages

- `internal/testutil`
- all package-local `*_test.go` files under `internal/`
