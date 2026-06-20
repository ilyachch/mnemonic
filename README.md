# mnemonic

mnemonic is a local-first knowledge base, search tool, and multi-tenant MCP server for Markdown notes.

It provides:

- a CLI for project, note, and web authorization lifecycle management,
- a file-based registry for central and local project spaces,
- a disposable per-project SQLite index for search/backlinks/tags,
- an **HTTP Web MCP Server** utilizing the **Server-Sent Events (SSE)** transport with explicit token authorization.

## Principles

- Markdown files are the source of truth.
- The project registry is simple and fully file-based (no database).
- Index databases are disposable and can be rebuilt at any time.
- Web routing state is deterministic: projects are validated on boot and connections are pooled efficiently.
- Human-readable output is friendly; `--json` is automation-friendly.

## Feature Highlights

- Lightweight file-based registry with slug/UUID addressing.
- Full-text search using SQLite FTS5 and automatic backlinks generation.
- Multi-tenant HTTP SSE server engine with explicit user token isolation.
- Thread-safe, lazy-loading instance manager (`sync.RWMutex`) to minimize database connection footprints.
- Built-in administrative commands for web user provisioning and permission mapping.

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

### 1. Local Workspace Configuration

Initialize a local project layout:

```bash
mnemonic init my-notes --local
```

Create a note:

```bash
mnemonic notes create --project my-notes --title "System Architecture" --tag design --tag ops
```

Build the initial index:

```bash
mnemonic project reindex my-notes
```

### 2. Multi-Tenant Web Server Setup

Provision a new web user (this will generate and output a secure API token):

```bash
mnemonic web users add alice
```

_Save the printed token! Only its SHA-256 hash is committed to the authentication storage._

Grant the user access to your project:

```bash
mnemonic web perms grant alice my-notes --level rw
```

Start the HTTP Web MCP server:

```bash
export MNEMONIC_SUPERUSER_TOKEN="my-secure-root-token"
mnemonic web serve --port 8080 --projects my-notes
```

Your AI client can now establish an MCP session using the standard HTTP/SSE endpoints by providing the Bearer token:

- **SSE Connection:** `GET http://localhost:8080/mcp/my-notes/sse`
- **Client Messages:** `POST http://localhost:8080/mcp/my-notes/messages`

## Data Model and Paths

mnemonic strictly separates content, registry metadata, authentication states, and ephemeral cache paths:

- **Config:** `$XDG_CONFIG_HOME/mnemonic/config.toml`
- **Registry & Projects (`memories_home`):** `~/.mnemonic/` (default)
- **Web Authorization Database:** `$XDG_STATE_HOME/mnemonic/web_auth.sqlite` (configurable via `MNEMONIC_WEB_AUTH_DB` or `--auth-db`)
- **Per-project Indexes:** `$XDG_STATE_HOME/mnemonic/projects/<PROJECT_ID>/index.sqlite`
- **Locks:** `$XDG_STATE_HOME/mnemonic/projects/<PROJECT_ID>/locks/`

## CLI Reference

Top-level command scopes:

- `completion`: generate shell completion scripts.
- `config`: inspect effective config.
- `init`: initialize a project.
- `mcp`: run legacy MCP stdio adapter.
- `notes`: note operations.
- `project`: project operations.
- `tags`: tag listing.
- `version`: print build version.
- `web`: administrative web server controls (serving, users, perms).

### `web` command group

Administrative management for the HTTP Server-Sent Events architecture.

#### `web serve`

```bash
mnemonic web serve [--port 8080] [--projects common,code] [--auth-db /path/to/db]
```

Launches the HTTP web listener. Parses the project targets, mapping allowed routes immediately. If any target in the explicit list is physically missing from the registry, the server aborts initialization with exit code `3` (`CodeNotFound`). Project databases are opened lazily upon the first incoming client request to optimize memory.

#### `web users add`

```bash
mnemonic web users add <username> [--json]
```

Creates a new web consumer. Generates a cryptographically strong token, writing its SHA-256 digest to the authentication storage. Outputs the raw plaintext token precisely once.

#### `web users list`

```bash
mnemonic web users list [--json]
```

Lists registered web consumer names.

#### `web users revoke`

```bash
mnemonic web users revoke <username>
```

Permanently deletes the user record. Associated permission scopes are automatically purged via cascading foreign keys.

#### `web perms grant`

```bash
mnemonic web perms grant <username> <project-slug> --level [ro|rw]
```

Maps access rights for a user to a specific knowledge base slot.

#### `web perms revoke`

```bash
mnemonic web perms revoke <username> <project-slug>
```

Removes targeted knowledge base accessibility fields for the selected user.

### `project` commands

#### `project list`

```bash
mnemonic project list
```

Scans the registry home and displays active projects. If a project is corrupt or a local project's path is missing, it marks them appropriately:

- `[CORRUPTED]` (mismatched configurations)
- `[ORPHANED/MISSING]` (local workspace path moved or deleted)

#### `project reindex`

```bash
mnemonic project reindex [NAME_OR_UUID]
mnemonic project reindex --all
```

- with selector: rebuild one project index.
- with `--all`: rebuild all active projects.

### `notes` commands

#### `notes list`

```bash
mnemonic notes list --project PROJECT
```

#### `notes show`

```bash
mnemonic notes show SELECTOR --project PROJECT
```

Selector can be a note UUID, slug, path, or title.

#### `notes edit`

```bash
mnemonic notes edit SELECTOR --project PROJECT --if-match <content_hash> --append "text"
```

Enforces optimistic concurrency using custom SHA-256 hashes generated across note contents.

## Configuration

Typical config file location: `$XDG_CONFIG_HOME/mnemonic/config.toml`

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
    subgraph Interfaces ["1. Interfaces & Transports"]
        direction TB
        CLI("💻 CLI Admin\n(internal/cli)"):::interface
        WEB("🌐 Web Server (HTTP/SSE)\n(internal/web)"):::interface
        STDIO("🤖 Stdio Adapter\n(Legacy MCP CLI)"):::interface
    end

    %% 2. Core & Server Management
    App{"⚙️ Web Server Manager\n(Auth, Lazy-Loading,\nInstance Routing)"}:::core

    %% 3. Business Logic
    subgraph Services ["2. Business Services"]
        direction TB
        Auth("🔑 Web Auth Store"):::logic
        Proj("📁 Project Space"):::logic
        Note["📝 Notes Engine"]:::logic
    end

    %% 4. File System
    subgraph Storage ["3. Files & Layout Trees"]
        direction TB
        MD["📄 Notes (*.md)\n(Source of Truth)"]:::storage
        Reg["🗂️ Registry Manifests\n(~/.mnemonic/*.toml)"]:::storage
    end

    %% 5. Databases
    subgraph DBs ["4. Isolated Storage (SQLite)"]
        direction TB
        AuthDB[("🔒 web_auth.sqlite\n(Tokens & Perms)")]:::db
        FTS[("⚡ index.sqlite\n(Pooled per Project)")]:::db
    end

    %% Flow connections
    CLI --> App
    WEB --> App
    STDIO --> Proj

    App --> Auth
    App --> Proj
    App --> Note

    Auth --> AuthDB
    Proj --> Reg
    Note --> MD
    Note -. Scanned by .-> FTS
```

### Detailed Architecture

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
    subgraph Entry ["1. Presentation Layer (Transports)"]
        direction LR
        CLI("💻 CLI Controls\n(internal/cli)"):::entry
        WebServe("🌐 HTTP SSE Router\n(internal/web)"):::entry
    end

    subgraph Core ["2. Access & Lifecycle Security"]
        direction LR
        Manager{"⚙️ Server Manager\n(Lazy Thread Cache)"}:::core
        AuthStore("🔑 Auth Subsystem\n(internal/webauth)"):::core
    end

    subgraph Domain ["3. Domain layer (Business Objects)"]
        direction LR
        Proj("📁 Project Layout"):::domain
        Notes("📝 Notes Logic"):::domain
        Search("🔍 FTS Search Engine"):::domain
    end

    subgraph Engines ["4. File Parsing Processing"]
        direction LR
        MD("🛠️ Markdown AST"):::engine
        Idx("⚡ Reindexer Engine"):::engine
    end

    subgraph Infra ["5. Infrastructure & Storage"]
        direction LR
        AuthDB[("🔒 web_auth.sqlite")]:::db
        IndexDB[("🗄️ index.sqlite")]:::db
        Sys("💾 Locks & Filesystem"):::infra
    end

    %% Flow: Web Routing & Guarding
    WebServe --> Manager
    Manager --> AuthStore
    AuthStore --> AuthDB

    %% Flow: Instance Lazy Hydration
    Manager -. Creates pool .-> IndexDB
    Manager --> Proj
    Manager --> Notes

    %% Flow: Processing Boundary
    Notes --> MD
    Notes --> Sys
    Search --> Idx
    Idx --> IndexDB
```
