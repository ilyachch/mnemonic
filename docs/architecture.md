# Architecture

## Purpose

This is the top-level entrypoint for the single-KB runtime architecture.
Supporting detail lives in [`docs/architecture-target.md`](./architecture-target.md) and the package-specific pages under [`docs/architecture/`](./architecture).

## Target Split

- Catalog context knows about many knowledge bases.
- Catalog context handles project management and selector resolution.
- Runtime context knows about exactly one selected knowledge base.
- Runtime context handles notes, search, tags, backlinks, reindex, doctor, web, and stdio for that KB.
- Maintenance context applies runtime operations to many knowledge bases by iterating catalog entries.
- Adapters parse UX and choose catalog, runtime, or maintenance paths.

## Selector Rules

For single-project runtime commands, selector precedence is:

1. positional `PROJECT`
2. `--project`
3. `MNEMONIC_PROJECT`
4. usage error

`--project` is a CLI/bootstrap boundary input. Runtime services must not accept it.

## Runtime Rule

- `RuntimeApp` is built from one resolved `KnowledgeBase`.
- `RootDir`, `StateDir`, and `IndexPath` are resolved before `RuntimeApp` is created.
- Runtime services do not know about registry access or project selection.

## `--all`

- `project reindex --all` and `project doctor --all` are maintenance operations.
- `--all` must not be combined with positional `PROJECT`.
- `--all` must not be combined with `--project`.
- `--all` uses maintenance services, not `RuntimeApp` directly from CLI.

## CLI Surface

```bash
mnemonic project init NAME
mnemonic project list
mnemonic project show PROJECT
mnemonic project import [PATH]
mnemonic project remove PROJECT

mnemonic --project work notes list
mnemonic --project work notes create --title "Hello"

mnemonic project reindex [PROJECT]
mnemonic project doctor [PROJECT]
mnemonic project reindex --all
mnemonic project doctor --all

mnemonic --project work web serve --port 8081
mnemonic --project work stdio
```

Root-level `mnemonic init` is not part of the target surface.

## Extension Rules

### Notes commands

- Add notes commands to the runtime path.
- Resolve the selected knowledge base before creating runtime services.
- Keep selector parsing in the adapter layer.
- Keep markdown persistence in store code and note behavior in service code.

### Stdio tools

- Register stdio tools against runtime services.
- Do not resolve project selectors in stdio registration.
- Keep transport mapping in the stdio adapter.

### Web endpoints

- Map HTTP requests to runtime services.
- Keep auth, request parsing, and response formatting in the web adapter.
- Do not let web handlers own registry resolution or SQLite opening.

## Responsibility Boundaries

- SQL lives in `internal/store/sqliteindex`.
- Markdown parsing lives in `internal/format/markdown`.
- Registry resolving lives in `internal/store/registry` and catalog services.
- Runtime services must not import `cobra`, `registry`, `MNEMONIC_PROJECT` handling, or project selector resolution.
- Adapters do not open SQLite directly.
- Adapters do not resolve project selectors directly.
