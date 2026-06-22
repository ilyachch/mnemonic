# Package Map

## `internal/domain`

- Pure domain types and small value objects.
- Holds selected knowledge base data and slug-related helpers.
- No storage, CLI, environment, or transport concerns.

## `internal/platform`

- Low-level platform helpers.
- Owns config, paths, filesystem, locking, clock, id generation, and build metadata.
- No business use cases and no selector resolution.

## `internal/format`

- Markdown parsing and rendering.
- Owns frontmatter, wikilinks, tags, observations, and relations.
- No registry resolving, selector lookup, CLI, or transport code.

## `internal/store`

- Low-level persistence and filesystem-backed storage.
- Owns registry scanning, markdown file operations, and SQLite index operations.
- No CLI flags, environment selector lookup, web server logic, or application orchestration.

## `internal/service`

- Business use cases and orchestration.
- Includes catalog, runtime, indexing, and maintenance services.
- Runtime services operate on an already resolved knowledge base and do not accept project selectors.

## `internal/app`

- Bootstrap construction and runtime wiring.
- Builds runtime apps and composes services and stores.
- No business logic, SQL queries, or markdown parsing details.

## `internal/adapter`

- Transport and UX boundary code.
- Includes CLI, web, and stdio adapters.
- Parses flags, reads environment inputs, formats output, maps exit codes, and chooses catalog/runtime/maintenance paths.
- Does not own SQL queries, registry scanning, markdown persistence, or runtime business logic.
