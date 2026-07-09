# Mnemonic Developer Guide

This document is intended for developers and automated design systems working on modifying, extending, or maintaining the `mnemonic` codebase. It describes architectural decisions, directory structure, coding standards, and rules for making changes.

---

## Architecture Overview

The codebase is organized around the principles of ports and adapters (hexagonal architecture), which isolates the core business logic from the CLI interface, MCP transport layers, and physical document storage.

### Directory Structure

```
.
├── cmd/mnemonic/             # Entry point: container initialization and CLI launch
├── internal/
│   ├── adapter/
│   │   ├── cli/             # Cobra command tree, flag validation, and output formatting
│   │   ├── stdio/           # MCP stdio server adapter and tool registration
│   │   └── web/             # HTTP/SSE MCP server implementation
│   ├── app/                 # Dependency injection, initialization of global and runtime containers
│   ├── apperr/              # Structured application errors and process exit codes
│   ├── domain/
│   │   ├── kb/              # Runtime model representing the active knowledge base
│   │   └── slug/            # Algorithms for generating symbolic identifiers (slugs)
│   ├── format/
│   │   ├── manifest/        # TOML schemas for mnemonic.toml and pointer files
│   │   └── markdown/        # Goldmark AST parsers for notes, tags, and relations
│   ├── platform/            # Infrastructure components (filesystem, clock, locks, paths)
│   ├── service/             # Service layers (catalogsvc, indexsvc, maintsvc, notesvc, searchsvc)
│   └── store/
│       ├── markdownstore/   # Physical markdown file management, read/write operations
│       ├── registry/        # File-based project registry (central and local)
│       └── sqliteindex/     # Connection, schema, and queries for the SQLite index database
└── scripts/                 # Diagnostic scripts and smoke tests
```

---

## Lifecycle and Component Binding

1. **Initialization (`app.Bootstrap`):**
   Implemented in `internal/app/app.go`. This step resolves the user's configuration, calculates default paths, and initializes the project registry store (`registry.Store`) and the catalog service (`catalogsvc.Service`).
2. **Runtime Container (`app.RuntimeApp`):**
   When accessing a specific project, the `b.Runtime()` method returns an isolated `RuntimeApp` container. This container holds the `notesvc`, `searchsvc`, and `indexsvc` services configured to operate within the selected project directory.

---

## Coding Standards and Invariants

When making changes, you must strictly adhere to the following requirements:

### 1. Deterministic Time and Identifiers

Direct usage of `time.Now()` or external UUID generators within services or domain models is prohibited.

- Use time and UUID generation interfaces to ensure testability.
- Retrieve the current time using `clock.NowUTC()`.
- Generate UUIDs using `idgen.NewUUID()`.
- Interface definitions can be found in `internal/platform/clock/clock.go`.

### 2. Error Classification and Exit Codes

Errors must be wrapped in the `apperr.Error` struct to return the correct exit codes in the CLI.

- Use the helper constructors: `apperr.CLIUsage()`, `apperr.NotFound()`, `apperr.Unsafe()`, `apperr.Corrupted()`, `apperr.IO()`, `apperr.Ambiguous()`.
- **Do not classify errors by message text.** Classification must use type assertions (`errors.As`) or `apperr.Code`. Code that inspects `err.Error()` with `HasPrefix`, `HasSuffix`, or `Contains` to determine the error category is prohibited.
- Avoid direct calls to `panic()`; errors should be handled at the adapter boundaries.

### 3. Safe File Writes

- Writing Markdown files and metadata must be performed via calls to `internal/platform/fs/atomic.go`.
- Direct writing via `os.WriteFile` without a temporary buffer and a `Sync()` call on the parent directory is prohibited, as it can lead to file corruption if the system terminates abruptly.

### 4. Locking Invariants

- Any mutation of notes in the storage requires acquiring an exclusive write lock via `internal/platform/lock` with the name `write`.
- Rebuilding the search index requires acquiring the `reindex.lock` in the active project's state directory.
- Releasing locks must always be handled in a `defer` block.

### 5. Pure Go SQLite Driver

- The project uses the `modernc.org/sqlite` driver to avoid CGO dependencies.
- All SQL queries must remain compatible with the SQLite3 specification.

### 6. Index Schema Validation

- `ValidateSchema()` in `internal/store/sqliteindex/schema_check.go` performs **structural** checking only (table and column presence via `PRAGMA table_info`).
- It does **not** use version numbers, `PRAGMA user_version`, or any migration-like mechanisms.
- The index is a disposable derived artifact — incompatibility is resolved by an explicit `mnemonic project reindex`, never automatically.

### 7. Minimal Frontmatter Model

The `Note` struct (`internal/format/markdown/note.go`) does **not** carry `CreatedAt`/`UpdatedAt` fields. Timestamps are resolved at runtime:

- **Indexer** (`sqliteindex/scan.go`): uses `fs.GetFileTimes` to obtain `btime`/`mtime`. `CreatedAt` = YAML `frontmatter["created_at"]` if present, else `btime`, else `mtime`. `UpdatedAt` = YAML `frontmatter["updated_at"]` if present, else `mtime`.
- **read_notes / show**: `ShowResult.UpdatedAt` (system mtime) is the fallback when YAML timestamps are absent.
- **RenderNote**: omits empty optional fields; `created_at`/`updated_at` are in `removedFrontmatterKeys` and are stripped from legacy files on write.
- **Hydrate**: only writes `mnemonic_note_id`; `title` and `slug` are derived dynamically via `Note.GetOrDeriveTitle(relPath)` and `Note.GetOrDeriveSlug(relPath)`.
- **Diagnostics**: `KindMissingTimestamp` and `KindInvalidTimestamp` no longer exist.

---

## Guidelines for Extending Code

### Running Tests and Linters

1. After making changes, run the full test suite: `go test ./...`.
2. Check the code style using the linter: `golangci-lint run`.

### Adding a New CLI Command

1. Create a command file in `internal/adapter/cli/`.
2. Define the `cobra.Command` structure:
   ```go
   var myNewCmd = &cobra.Command{
       Use:   "my-command [ARGS]",
       Short: "Brief description",
       RunE: func(cmd *cobra.Command, args []string) error {
           // Processing logic
           return nil
       },
   }
   ```
3. Register the command within `buildCommandTree()` inside `command_tree.go`.

### Adding a New MCP Tool

1. Describe the input and output parameter structures in `internal/adapter/stdio/tools.go`.
2. Register the tool depending on its impact on state (read or write):
   ```go
   func RegisterMyNewTool(server *sdkmcp.Server, deps Dependencies) {
       sdkmcp.AddTool(server, &sdkmcp.Tool{
           Name:        "my_new_tool",
           Description: "Purpose description for the AI model",
       }, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input MyInput) (*sdkmcp.CallToolResult, MyOutput, error) {
           // Invocation logic for internal services from deps
           return nil, MyOutput{}, nil
       })
   }
   ```

### Modifying Markdown Parsing

If you need to change how tags, links, or observations are extracted:

1. Document parsing logic must not reside in the storage or service layers.
2. Apply changes within the `internal/format/markdown/` package.
3. Test Goldmark AST node manipulation in isolation within the package's respective test files.
