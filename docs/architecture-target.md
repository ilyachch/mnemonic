# Architecture Target

## Purpose

This page defines the target-state split for the single-KB runtime refactor.

## Catalog Context

- Knows about many knowledge bases.
- Handles project management and selector resolution.
- Owns registry scanning and enumeration.

## Runtime Context

- Knows about exactly one selected knowledge base.
- Handles notes, search, tags, backlinks, reindex, doctor, web, and stdio for that KB.
- Does not accept project selectors.

## Maintenance Context

- Applies runtime operations to many knowledge bases by iterating catalog entries.
- Aggregates per-project results.
- Uses runtime services, not direct CLI orchestration.

## Adapter Boundaries

- CLI parses flags and chooses catalog, runtime, or maintenance paths.
- Web maps HTTP requests to runtime services.
- Stdio maps MCP tools to runtime services.
- Adapters do not resolve project selectors directly.
- Adapters do not open SQLite directly.

## Single-KB RuntimeApp Rule

- `RuntimeApp` is built from one resolved `KnowledgeBase`.
- `RootDir`, `StateDir`, and `IndexPath` are resolved before `RuntimeApp` is created.
- Runtime services do not know about registry access or project selection.

## Selector Rule

For single-project runtime commands, selector precedence is:

1. positional `PROJECT`
2. `--project`
3. `MNEMONIC_PROJECT`
4. usage error

`--project` is a boundary input for adapters only. Runtime services must not accept it.

## `--all` Rule

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

## Notes

- Root-level `mnemonic init` is not part of the target surface.
- This is a target-state reference, not an implementation guide.
