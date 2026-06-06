# `internal/index`

## What this package owns

This package owns per-project index paths, SQLite connection setup, schema creation, schema compatibility checks, integrity checks, reindex locking, note scanning for rebuilds, and atomic replacement of `index.sqlite`.

## What this package does not own

This package does not own global project metadata in `registry.sqlite`, CLI presentation, project selection, or Markdown parsing rules beyond consuming parsed note structures. It also does not own the higher-level search command contract.

## Important invariants

- `index.sqlite` is derived state and may be rebuilt from Markdown.
- Search storage is SQLite FTS5; do not add vector-specific indexing here.
- Full rebuilds write a temporary database, run validation, then swap into place.
- Incompatible schema versions are treated as `needs_rebuild`, not migrated in place.
- Reindex is guarded by a project-scoped lock to prevent concurrent rebuilds.

## Tests to update when changing this package

- `internal/index/schema_test.go`
- `internal/index/check_test.go`
- `internal/index/db_test.go`
- `internal/index/lock_test.go`
- `internal/index/paths_test.go`
- `internal/cli/project_reindex_test.go`
- `internal/cli/project_doctor_test.go`
- `internal/search/search_test.go` and `internal/graph/*_test.go` when indexed data shape changes
