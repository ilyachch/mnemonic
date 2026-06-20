# Agent Development Guide (AGENTS.md)

This document defines the constraints, invariants, and style guidelines for AI agents and human developers contributing to `mnemonic`.

## Tech Stack & Core Libraries

- **Language**: Go (current standard library features).
- **CLI Framework**: Cobra (`github.com/spf13/cobra`).
- **Markdown Engine**: Goldmark (`github.com/yuin/goldmark`) with AST traversal.
- **YAML/TOML Parsing**: `gopkg.in/yaml.v3` (for frontmatter) and `github.com/pelletier/go-toml/v2` (for config/manifests/pointers).
- **Database**: Pure Go SQLite (`modernc.org/sqlite`) used strictly for per-project search indexes. The project registry is completely file-based and does not use a database.

## Workflows

- **Memory first**: Before starting any work, check project memory or note directories for relevant facts.
- **Use just as tasks runner**: For running tests, linters, builds, and other development tasks, use `just`.

## Core Architectural Invariants

1. **Markdown is the Source of Truth**: Never store canonical note content exclusively in SQLite. Databases (indexes) are fully disposable and can be rebuilt via `project reindex`.
2. **No Silent Changes to User Notes**: Schema modifications or index rebuilds must never silently overwrite or modify on-disk `.md` files without an explicit edit/delete command from the user.
3. **No Implicit Default Project**: There is no fallback project. If a command cannot resolve project context from flags, environment variables (`MNEMONIC_PROJECT`), or local path discovery, it must fail with an error.
4. **Pure File-Based Registry**: No global SQL database is allowed to manage project listings. The registry is composed of directories (for central projects) and `.toml` pointer files (for local projects) stored under `memories_home`.
5. **Fail-Fast Validation**: The resolver must validate that the pointer name or directory name matches the internal `slug` defined in `mnemonic.toml`. Mismatches or orphaned pointers must cause commands to fail immediately with descriptive errors.

## Directories & XDG Layout

- Configuration -> `XDG_CONFIG_HOME/mnemonic/config.toml`
- File-Based Registry (`memories_home`) -> `~/.mnemonic/` (default).
  - Central projects: stored under `~/.mnemonic/<slug>/` (containing `mnemonic.toml` and notes).
  - Local projects: registered via `~/.mnemonic/<slug>.toml` pointing to the workspace manifest.
- Per-project Indexes -> `XDG_STATE_HOME/mnemonic/projects/<UUID>/index.sqlite`
- Locks -> `XDG_STATE_HOME/mnemonic/projects/<UUID>/locks/`

## Safe File Modification & Concurrency

- **Atomic Writes**: Always use `internal/fs.AtomicWriteFile` when writing Markdown notes or TOML files to prevent partial writes.
- **Write-Locking**: Before performing create, edit, or delete operations on a project's notes, acquire the project-scoped lock using `internal/notes.acquireWriteLock`.
- **Reindex Locking**: Reindexing is guarded by a separate `reindex` lock using `internal/index.acquireReindexLock`.

## Error Handling & Exit Codes

Always use typed application errors from `internal/app` to maintain predictable CLI exit codes:

- `CodeSuccess` = 0
- `CodeInternal` = 1 (Standard/unclassified errors)
- `CodeCLIUsage` = 2 (Usage or syntax issues)
- `CodeNotFound` = 3 (Missing notes, configurations, or projects)
- `CodeAmbiguous` = 4 (Multiple projects resolved, duplicate slugs, or name conflicts)
- `CodeUnsafe` = 5 (Lock contention, hash mismatch on conditional edits)
- `CodeCorrupted` = 6 (SQLite index corruption detected by `quick_check`)

## Testing Guidelines

- **No Shared State**: Tests must call `testutil.CleanEnvForTest(t)` at the beginning to isolate the environment inside a temporary directory.
- **Deterministic Clocks**: Do not use `time.Now()` directly in packages that perform business logic or write files. Inject the custom clock (`project.Clock`) or use `project.SetClock` in tests to ensure deterministic timestamps and UUIDs.
- **Command Testing**: Use `executeCommand` helpers in `internal/cli` to verify CLI command output and exit codes.
```
