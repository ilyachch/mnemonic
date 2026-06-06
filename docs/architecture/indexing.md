# Indexing

## Purpose

The indexing layer provides deterministic search and graph queries over Markdown notes. It scans a project's notes, derives searchable structures, writes a fresh SQLite index, validates it, and atomically swaps it into place.

## Invariants

- Search is implemented with SQLite FTS5.
- The index is derived state and may be rebuilt from Markdown.
- Full rebuilds use a temporary database and swap it into place only after validation.
- Incompatible index schema versions are treated as `needs_reindex`, not migrated in place.
- Reindex operations are guarded by a project-scoped lock.

## Current flow

1. Resolve the project and note root.
2. Walk Markdown files with `internal/notes`.
3. Parse note metadata, tags, observations, and relations.
4. Build a fresh `index.new.sqlite` with schema version `1`.
5. Run SQLite `quick_check`.
6. Replace any old index files with the new index.

## Current query surface

- Full-text note search lives in `internal/search`.
- Backlinks use link rows through `internal/graph`.
- Tag listing and note listing are served from indexed/project state depending on the command.
- `project doctor` checks index presence, integrity, and schema compatibility.

## Related packages

- `internal/index`
- `internal/search`
- `internal/graph`
- `internal/cli` for `project reindex`, `project doctor`, `notes search`, and `tags list`
