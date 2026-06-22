# MCP Server

## Purpose

The MCP server exposes the local note system to MCP clients over stdio. It reuses the same project resolution and note/index services as the CLI while keeping the process scoped to one project context.

## Invariants

- One MCP process serves one resolved project.
- Tool schemas must not accept a `project` argument.
- Read-only and write tools operate only within the startup project context.
- Tool behavior must follow the same note hashing, locking, trash, and indexing rules as the CLI.

## Current tool surface

The current server registers these tools:

- `list_notes`
- `list_tags`
- `search_notes`
- `read_note`
- `list_backlinks`
- `create_note`
- `edit_note`
- `delete_note`

## Current behavior

- Startup resolves the project from `--project`, `MNEMONIC_PROJECT`, or current-directory discovery.
- The server uses the Go MCP SDK stdio transport.
- Tool annotations distinguish read-only operations from destructive ones.
- Write tools use note hashes and file locks to reject stale or conflicting updates.

## Related packages

- `internal/adapter/stdio`
- `internal/project`
- `internal/notes`
- `internal/index`
- `internal/search`
- `internal/graph`
