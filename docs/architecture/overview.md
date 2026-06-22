# Architecture Overview

## Purpose

`mnemonic` is a local-first note system with two adapters: a Cobra CLI and a stdio MCP server. User data lives in Markdown files. Machine state lives in SQLite under XDG paths. The system is organized so Markdown remains the source of truth and disposable indexes can be rebuilt from files.

## Invariants

- Markdown notes are the source of truth.
- SQLite FTS5 is the only search engine in the current design.
- There is no background watcher or daemon-driven sync.
- Commands must not fall back to an implicit default project outside an explicit selector or a `.mnemonic` context.
- MCP serves one resolved project per process.
- XDG-managed paths own registry, state, cache, and lock files.

## Main packages

- `cmd/mnemonic` bootstraps the CLI.
- `internal/cli` implements commands and output contracts.
- `internal/app` wires config, paths, registry, and service dependencies.
- `internal/project` resolves project context from `.mnemonic`, `mnemonic.toml`, CLI flags, and environment.
- `internal/notes` owns Markdown file lifecycle operations.
- `internal/format/markdown` parses and renders note structure.
- `internal/index` and `internal/search` build and query the SQLite FTS5 index.
- `internal/registry` and `internal/registryschema` own global project metadata.
- `internal/adapter/stdio` exposes the same local project model through MCP tools.
- `internal/platform/fs` and `internal/platform/lock` provide atomic file writes and cross-process locking.

## Related packages

- `internal/graph` computes backlinks from indexed relationships.
- `internal/platform/config` and `internal/platform/paths` resolve configuration and XDG locations.
- `internal/testutil` supports deterministic tests.
