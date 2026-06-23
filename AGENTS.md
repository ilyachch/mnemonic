# Mnemonic Developer Agent Guide

This document is intended for AI coding assistants and developers working on modifying, extending, or maintaining the `mnemonic` codebase. It outlines the architectural design, directory layout, coding standards, and common modification recipes.

---

## Architecture Overview

The codebase is structured around a Ports and Adapters (Hexagonal) architecture, separating business logic from CLI interaction, MCP transport layers, and file-based storage formats.

### Directory Structure

```
.
├── cmd/mnemonic/             # Entrypoint: bootstraps application and executes CLI
├── internal/
│   ├── adapter/
│   │   ├── cli/             # CLI command tree and validation (Cobra)
│   │   ├── stdio/           # MCP stdio server adapter & tool registrations
│   │   └── web/             # MCP HTTP/SSE server implementation
│   ├── app/                 # Dependency injection and runtime container wiring
│   ├── apperr/              # Structured error classification and exit codes
│   ├── domain/
│   │   ├── kb/              # Runtime knowledge base representation
│   │   └── slug/            # Core slugification algorithms
│   ├── format/
│   │   ├── manifest/        # TOML schemas for mnemonic.toml and pointer files
│   │   └── markdown/        # Goldmark-based AST parsers for notes, tags, and relations
│   ├── platform/            # Infrastructure-agnostic packages (fs, clock, lock, paths)
│   ├── service/             # Orchestration layers (catalogsvc, indexsvc, maintsvc, notesvc)
│   └── store/
│       ├── markdownstore/   # Core write operations and physical note management
│       ├── registry/        # Pointer-to-manifest project lookups
│       └── sqliteindex/     # SQLite database connection, schema, and queries
└── scripts/                 # System diagnostic and smoke-testing scripts
```

---

## Critical Lifecycle & Wiring Flow

1. **Bootstrapping (`app.Bootstrap`):**
   Configured in `internal/app/app.go`. Resolves user configurations, handles default path fallbacks, and initializes structural storage objects such as `registry.Store` and the parent `catalogsvc.Service`.
2. **Project Runtime (`app.RuntimeApp`):**
   When a project is targeted (via CLI or MCP), `Bootstrap.Runtime()` resolves the path configuration and returns a project-specific container containing `notesvc`, `searchsvc`, and `indexsvc` connected to the active project path.

---

## Core Development Standards

AI assistants must adhere to the following implementation details:

### 1. Deterministic Time and Identifiers

Do **not** use `time.Now()` or external random UUID generators directly in domain models or service writes.

- Always retrieve time and UUIDs through the platform interfaces to support predictable unit testing.
- Use `clock.NowUTC()` or dependency-injected clock functions.
- Generate UUIDs via `idgen.NewUUID()`.
- Refer to `internal/platform/clock/clock.go` for details.

### 2. Error Classification & Exit Codes

Wrap failures in `apperr.Error` values to ensure the CLI exits with the appropriate status code.

- Prefer helper builders such as `apperr.CLIUsage()`, `apperr.NotFound()`, `apperr.Unsafe()`, or `apperr.Corrupted()`.
- Avoid naked panic statements; let errors bubble back to adapter boundary execution runs.

### 3. Safe File Modification

- Use `internal/platform/fs/atomic.go` for writing markdown documents or metadata.
- Mutating files directly via `os.WriteFile` bypasses the staging-and-atomic-swap lifecycle, which can lead to file corruption on crashes.

### 4. Lock Acquisition Invariants

- Modifications to Markdown notes (`Store.Create`, `Store.Edit`, `Store.Delete`) require acquiring the write lock via `internal/platform/lock`.
- Rebuilding indices requires acquiring `rebuildLockPath` under the active state directory.
- Always release locks using deferred guards.

### 5. Pure Go SQLite Driver

- This project uses `modernc.org/sqlite` instead of `cgo`-dependent drivers.
- Ensure SQL dialect-specific operations are fully compatible with sqlite3 specifications.
- Connection setup is defined in `internal/store/sqliteindex/store.go` (`openDB`).

---

## How to Extend the Application

### Adding a New CLI Command

1. Locate the target area under `internal/adapter/cli/`.
2. Define your new command variable:
   ```go
   var myNewCmd = &cobra.Command{
       Use:   "my-command [ARGS]",
       Short: "Brief description",
       RunE: func(cmd *cobra.Command, args []string) error {
           // Execute logic
           return nil
       },
   }
   ```
3. Register the command within `buildCommandTree()` in `command_tree.go` and configure any required flags on the clone copy inside that file.

### Adding a New MCP Tool

1. Define input and output parameter structures in `internal/adapter/stdio/tools.go`.
2. Register the tool function under either `RegisterReadOnly` or `RegisterWrite`:
   ```go
   func RegisterMyNewTool(server *sdkmcp.Server, deps Dependencies) {
       sdkmcp.AddTool(server, &sdkmcp.Tool{
           Name:        "my_new_tool",
           Description: "Instructions for the LLM",
       }, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input MyInput) (*sdkmcp.CallToolResult, MyOutput, error) {
           // Connect to runtime services through deps
           return nil, MyOutput{}, nil
       })
   }
   ```

### Modifying Markdown Parsing Logic

If you need to change how elements (tags, links, observations) are extracted from markdown files:

1. Do not place layout detection code inside raw stores or services.
2. Modify or add parser interfaces inside the `internal/format/markdown/` package.
3. Test processing outputs against AST configurations in isolation using Goldmark nodes.
