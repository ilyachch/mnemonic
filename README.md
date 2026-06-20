# mnemonic

mnemonic is a local-first knowledge base and search tool for Markdown notes.

It provides:
- a CLI for project and note lifecycle management,
- a file-based registry for central and local project spaces,
- a disposable per-project SQLite index for search/backlinks/tags,
- an MCP stdio server for AI-tool integrations.

## Principles

- Markdown files are the source of truth.
- The project registry is simple and fully file-based (no database).
- Index databases are disposable and can be rebuilt at any time.
- Project operations are explicit and path-safe.
- Human-readable output is friendly; `--json` is automation-friendly.

## Feature Highlights

- Lightweight file-based registry with slug/UUID addressing.
- Seamless workspace management via central and local layout strategies.
- Full-text search using SQLite FTS5.
- Backlinks graph generated from indexed notes.
- Safe note editing with conditional updates (`--if-match`).
- App-agnostic MCP adapter over stdio.

## Installation

Build from source:

```bash
git clone https://github.com/ilyachch/mnemonic.git
cd mnemonic
go build ./cmd/mnemonic
```

Install binary into Go bin:

```bash
go install github.com/ilyachch/mnemonic/cmd/mnemonic@latest
```

## Quick Start

1. Initialize a project (local layout):

```bash
mnemonic init my-notes --local
```

2. Create a note:

```bash
mnemonic notes create --project my-notes --title "System Architecture" --tag design --tag ops
```

3. Build or refresh the index:

```bash
mnemonic project reindex my-notes
```

4. Search notes:

```bash
mnemonic notes search "architecture" --project my-notes
```

5. Show note content:

```bash
mnemonic notes show system-architecture --project my-notes
```

## Data Model and Paths

mnemonic strictly separates content, registry metadata, and ephemeral cache paths:

- Config: `$XDG_CONFIG_HOME/mnemonic/config.toml`
- Registry & Projects (`memories_home`): `~/.mnemonic/` (default)
- Per-project Indexes: `$XDG_STATE_HOME/mnemonic/projects/<PROJECT_ID>/index.sqlite`
- Locks: `$XDG_STATE_HOME/mnemonic/projects/<PROJECT_ID>/locks/`

Note markdown files live directly in your project workspaces, while SQLite indexes and file locks stay out of your repository trees under `$XDG_STATE_HOME`.

## Project Types

mnemonic supports two clean project layouts depending on where you want to keep your notes.

### regular (central)

This is the default mode when you run `mnemonic init NAME` without extra flags.

- Stores the project manifest `mnemonic.toml` and all markdown notes under `~/.mnemonic/<slug>/`.
- Useful for keeping notes in a central, structured directory without introducing files into your local coding repositories.

### local

Use `--local` when you want the project content to live directly inside your local repository or workspace.

- Stores Markdown notes under `.mnemonic-memories/<slug>` inside your current working directory.
- Creates a global pointer file `~/.mnemonic/<slug>.toml` under your central home, containing the absolute path to the local manifest.
- Perfect for keeping documentation self-contained alongside repository code.

---

## CLI Reference

Top-level commands:

- `completion`: generate shell completion scripts.
- `config`: inspect effective config.
- `init`: initialize a project.
- `mcp`: run MCP stdio adapter.
- `notes`: note operations.
- `project`: project operations.
- `tags`: tag listing.
- `version`: print build version.

All commands support `--json` global output mode.

### init

Create a project from the current working directory.

```bash
mnemonic init NAME [--local]
```

- Default mode creates a regular (central) project.
- `--local` stores notes under `.mnemonic-memories/<slug>` in the workspace and registers a global pointer file.

### project commands

#### project list

```bash
mnemonic project list
```

Scans the registry home and displays active projects. If a project is corrupt (JSON syntax issue or mismatched slug) or a local project's path is missing, it marks them appropriately:
- `[CORRUPTED]` (mismatched configurations)
- `[ORPHANED/MISSING]` (local workspace path moved or deleted)

#### project show

```bash
mnemonic project show NAME_OR_UUID
```

Prints detailed project metadata, location info, and internal file paths.

#### project reindex

```bash
mnemonic project reindex [NAME_OR_UUID]
mnemonic project reindex --all
```

- with selector: rebuild one project.
- with `--all`: rebuild all active projects.
- without args and without `--all`: rebuild all pending projects.

#### project doctor

```bash
mnemonic project doctor [NAME_OR_UUID]
```

Runs health diagnostics verifying registry pointer structures, manifest schemas, index integrity, and duplicate note checks.

#### project import

```bash
mnemonic project import PATH
```

Imports an existing project manifest from a specified directory path and registers a pointer file for it.

#### project remove

```bash
mnemonic project remove SLUG
mnemonic project remove --wipe SLUG
```

Behavior:
- `remove SLUG`: deletes the registry metadata (pointer TOML or central manifest) and the associated state index. Notes remain intact on disk.
- `remove --wipe SLUG`: deletes registry metadata, index, and recursively removes the physical folder containing your markdown notes.

### notes commands

#### notes list

```bash
mnemonic notes list --project PROJECT
```

Lists notes in the selected project.

#### notes show

```bash
mnemonic notes show SELECTOR --project PROJECT
```

Shows a resolved note (selector can be UUID, slug, path, or title).

#### notes create

```bash
mnemonic notes create --project PROJECT --title "Title" [--tag TAG ...]
mnemonic notes create --project PROJECT --title "Title" --stdin
mnemonic notes create --project PROJECT --title "Title" --body-file body.md
```

#### notes edit

```bash
mnemonic notes edit SELECTOR --project PROJECT --append "text"
mnemonic notes edit SELECTOR --project PROJECT --body-file body.md
mnemonic notes edit SELECTOR --project PROJECT --set key=value
mnemonic notes edit SELECTOR --project PROJECT --if-match <content_hash> --append "text"
```

#### notes delete

```bash
mnemonic notes delete SELECTOR --project PROJECT
mnemonic notes delete SELECTOR --project PROJECT --dry-run
mnemonic notes delete SELECTOR --project PROJECT --hard --yes
```

#### notes search

```bash
mnemonic notes search QUERY --project PROJECT [--tag TAG] [--limit N]
```

#### notes backlinks

```bash
mnemonic notes backlinks SELECTOR --project PROJECT
```

### tags commands

```bash
mnemonic tags list --project PROJECT
```

Returns aggregated tag counts from index.

## Shell Completion

Generate completion scripts:

```bash
mnemonic completion bash
mnemonic completion zsh
mnemonic completion fish
mnemonic completion powershell
```

Project-aware completion is available for:
- `project show/remove/reindex/doctor` positional selectors,
- `--project` flag in notes and tags commands.

## MCP Server (App-Agnostic)

mnemonic provides an MCP stdio adapter and can be integrated with any MCP-compatible client.

Run server for a project:

```bash
mnemonic mcp --project my-notes
```

To configure, point your MCP client's generic stdio server command to `mnemonic mcp --project <slug>`.

## Configuration

Use:

```bash
mnemonic config show
```

to inspect effective config and resolved paths.

Typical config file location: `$XDG_CONFIG_HOME/mnemonic/config.toml`

Example:

```toml
version = 1

[paths]
memories_home = "~/.mnemonic"

[notes]
delete_behavior = "trash"
trash_dir_name = ".trash"

[index]
fts = true
wal = true
busy_timeout_ms = 5000

[output]
json_pretty = true

[logging]
level = "info"
```

## Typical Workflows

Index refresh after external file changes:

```bash
mnemonic project reindex
```

Rebuild all projects:

```bash
mnemonic project reindex --all
```

Safe note edit with optimistic concurrency:

```bash
mnemonic notes show my-note --project my-notes --json
mnemonic notes edit my-note --project my-notes --if-match <hash> --append "\nUpdate"
```

## Development

### High-Level Architecture

```mermaid
flowchart LR
    %% Styling
    classDef interface fill:#e3f2fd,stroke:#1565c0,stroke-width:2px,color:#000
    classDef core fill:#fff3e0,stroke:#e65100,stroke-width:2px,color:#000
    classDef logic fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px,color:#000
    classDef storage fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#000
    classDef db fill:#ffebee,stroke:#c62828,stroke-width:2px,color:#000

    %% 1. External Interfaces
    subgraph Interfaces ["1. Interfaces (Input)"]
        direction TB
        CLI("💻 CLI\n(For Humans)"):::interface
        MCP("🤖 MCP Server\n(For LLMs / AI)"):::interface
    end

    %% 2. Core
    App{"⚙️ App Container\n(DI, Config,\nPath Resolution)"}:::core

    %% 3. Business Logic
    subgraph Services ["2. Business Services"]
        direction TB
        Proj("📁 Project Manager"):::logic
        Note["📝 Notes Manager"]:::logic
        Idx("🔍 Indexer & Search"):::logic
    end

    %% 4. File System
    subgraph Storage ["3. Files (Disk)"]
        direction TB
        Conf["⚙️ config.toml\n(Settings)"]:::storage
        MD["📄 Notes (*.md)\n(Source of Truth)"]:::storage
        Reg["🗂️ Registry Manifests\n(~/.mnemonic/*.toml)"]:::storage
    end

    %% 5. Databases
    subgraph DBs ["4. Databases (SQLite)"]
        direction TB
        FTS[("⚡ Index DB\n(FTS5 Search & Graphs)")]:::db
    end

    %% Flow connections
    CLI --> App
    MCP --> App

    App --> Proj
    App --> Note
    App --> Idx

    %% Storage connections
    App -. Reads .-> Conf
    Proj --> Reg
    Note --> MD
    Idx --> FTS
    Idx -. Parses .-> MD
```

## Detailed Architecture

```mermaid
flowchart TD
    %% Styling
    classDef entry fill:#e3f2fd,stroke:#1e88e5,stroke-width:2px,color:#000,rx:8px,ry:8px
    classDef core fill:#fff8e1,stroke:#ffb300,stroke-width:2px,color:#000,rx:8px,ry:8px
    classDef domain fill:#f3e5f5,stroke:#8e24aa,stroke-width:2px,color:#000,rx:8px,ry:8px
    classDef engine fill:#e8f5e9,stroke:#43a047,stroke-width:2px,color:#000,rx:8px,ry:8px
    classDef infra fill:#eceff1,stroke:#546e7a,stroke-width:2px,color:#000,rx:8px,ry:8px
    classDef db fill:#ffebee,stroke:#e53935,stroke-width:2px,color:#000,rx:8px,ry:8px

    %% Layers
    subgraph Entry ["1. Presentation Layer (Entry Points)"]
        direction LR
        CLI("💻 CLI\n(internal/cli)"):::entry
        MCP("🤖 MCP Tools\n(internal/mcp)"):::entry
    end

    subgraph Core ["2. Application Core (Bootstrapping)"]
        direction LR
        App{"⚙️ App Container\n(internal/app)"}:::core
        Paths("📂 Paths & Config\n(internal/paths, config)"):::core
    end

    subgraph Domain ["3. Domain Layer (Business Features)"]
        direction LR
        Proj("📁 Project\n(internal/project)"):::domain
        Notes("📝 Notes\n(internal/notes)"):::domain
        Search("🔍 Search & Graph\n(internal/search, graph)"):::domain
    end

    subgraph Engines ["4. Processing Engines"]
        direction LR
        MD("🛠️ Markdown\n(internal/markdown)"):::engine
        Idx("⚡ Indexer\n(internal/index)"):::engine
    end

    subgraph Infra ["5. Infrastructure & Adapters"]
        direction LR
        Reg("🗂️ Registry Files\n(internal/registry)"):::infra
        Sys("💾 FS & Locks\n(internal/fs, lock)"):::infra
        DB[("🗄️ SQLite\n(Databases)")]:::db
    end

    %% Flow: Bootstrapping
    CLI & MCP --> App
    App --> Paths
    App --> Reg

    %% Flow: Feature Execution
    CLI & MCP --> Proj
    CLI & MCP --> Notes
    CLI & MCP --> Search

    %% Flow: Domain to Engines & Infra
    Proj --> Reg
    Proj --> Paths

    Notes --> MD
    Notes --> Sys

    Search --> Idx

    %% Flow: Engines to Infra
    Idx --> MD
    Idx --> DB
```
