# ADR 0005: Registry in SQLite

## Context

The application needs a machine-local catalog of projects, their physical locations, and status fields such as whether an index exists or needs rebuild. This data benefits from structured queries, schema versioning, and transactional updates.

## Decision

The global project registry is stored in `registry.sqlite`. Schema changes are handled through an explicit migration runner, while disposable per-project indexes remain separate SQLite files with rebuild semantics.

## Consequences

- Project metadata can be queried and updated transactionally.
- Registry versioning can evolve independently from project indexes.
- Operational commands such as `project list`, `project show`, `project discover`, and `project reindex` can share one canonical metadata store.
- New project metadata should prefer the registry database instead of ad hoc sidecar files when the data is machine-managed and query-oriented.
