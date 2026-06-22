# Storage Model

## Purpose

The storage model separates human-authored content from machine-managed state. Markdown files are durable data. SQLite files and locks are operational state that can be recreated or updated safely.

## Invariants

- User-authored note content is stored in Markdown files.
- `registry.sqlite` is the global machine-local catalog of projects.
- `index.sqlite` is per-project derived state and can be rebuilt.
- Schema/version handling must not silently rewrite Markdown notes.
- Registry, indexes, caches, and locks must live under XDG-managed directories.

## Storage layout

- Config lives in `config.toml` under the config home.
- The registry database lives under the data home.
- Per-project index databases live under the state home, keyed by project UUID.
- Local project notes live under the repository's configured memories path.
- Non-local project notes live under the configured memories home and include `mnemonic.toml`.
- Deleted notes may be moved into a project-local `.trash/` directory unless hard delete is requested.

## Current databases

### Registry database

`registry.sqlite` tracks projects, locations, and status fields such as `index_present` and `needs_reindex`. It uses a migration runner in `internal/registryschema`.

### Index database

Each project index stores notes, tags, observations, links, and an FTS5 virtual table. Incompatible indexes are marked for rebuild rather than migrated in place.

## Related packages

- `internal/platform/config` and `internal/platform/paths`
- `internal/registry` and `internal/registryschema`
- `internal/index`
- `internal/notes`
- `internal/platform/fs` and `internal/platform/lock`
