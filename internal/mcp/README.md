# `internal/mcp`

## What this package owns

This package owns the stdio MCP server wrapper, tool registration, tool schemas, MCP-facing request and response structs, and the mapping from MCP tool calls to local note, search, and graph operations.

## What this package does not own

This package does not own core note storage, project schema rules, search indexing internals, or CLI command behavior. It should adapt existing services into MCP, not reimplement them.

## Important invariants

- One MCP process serves one resolved project context.
- Tool schemas must not accept a `project` argument.
- Read-only versus destructive hints must match actual tool behavior.
- Write tools must respect note hashes, locks, and delete-mode semantics from core note services.
- Tool registration and schema snapshots should stay stable unless the product surface intentionally changes.

## Tests to update when changing this package

- `internal/mcp/server_test.go`
- `internal/mcp/testdata/read_only_tools.snapshot.json`
- `internal/cli/mcp_test.go` if CLI launch behavior changes
- relevant `internal/notes`, `internal/search`, or `internal/graph` tests when MCP-exposed behavior depends on their contracts
