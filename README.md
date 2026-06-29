# mnemonic

`mnemonic` is a local-first personal knowledge base and search engine. Operating as a Command Line Interface (CLI) and a Model Context Protocol (MCP) server, it allows you to organize, link, and search Markdown-formatted notes.

The application indexes documents into a local SQLite database, using FTS5 (Full-Text Search) virtual tables to rank results via the BM25 algorithm. It extracts structured inline metadata, tracks backlinks, and exposes MCP interfaces (via both stdio and SSE) for integration with external AI assistants and compatible clients.

---

## Features

- **Local-first Architecture:** Data is stored as plain Markdown files and standard SQLite databases inside the user's workspace.
- **Dual Project Configurations:**
  - **Central:** Stored directly within the configured home directory for notes.
  - **Local:** Pointer files linking arbitrary directories (such as a Git repository) containing a `mnemonic.toml` file to the global registry.
- **Metadata and Relationship Extraction:**
  - Parses standard YAML Frontmatter (`mnemonic_note_id`, `slug`, `tags`, etc.).
  - Detects inline hashtags (`#tag`).
  - Parses bulleted lists of observations with explicit categorization: `- [category] content`.
  - Supports wikilink syntax `[[Target Note]]` or aliased links `[[Target Note|Alias]]`.
  - Parses explicitly declared relationships under a `## Relations` heading.
- **SQLite FTS5 Search Index:** Indexes titles, tags, content, and observations for fast local searches.
- **Model Context Protocol (MCP) Support:** Provides tools for safely reading, searching, updating, and managing notes from compatible MCP clients.
- **SSE Server:** Exposes the MCP API over HTTP SSE transport with optional Bearer token authentication.

---

## Installation

To compile and install the CLI from source, clone the repository and run:

```bash
go build -o mnemonic ./cmd/mnemonic
# Or install to your $GOPATH/bin
go install ./cmd/mnemonic
```

---

## Configuration and Paths

`mnemonic` adheres to the XDG Base Directory Specification. You can inspect the active paths on your system using:

```bash
mnemonic config show
```

### Configuration File Lookup Order:

1. CLI flag `--config-file`.
2. Environment variable `MNEMONIC_CONFIG_FILE`.
3. `$XDG_CONFIG_HOME/mnemonic/config.toml` (typically `~/.config/mnemonic/config.toml`).

### Configuration Schema (`config.toml`):

```toml
version = 1

[paths]
# Absolute or relative path to store central projects.
# If empty, defaults to ~/.mnemonic.
memories_home = "~/.mnemonic"

[notes]
# Note deletion behavior. Supported: "trash"
delete_behavior = "trash"
trash_dir_name = ".trash"

[index]
# Enable/disable full-text search, WAL mode, and database busy timeout
fts = true
wal = true
busy_timeout_ms = 5000

[output]
# JSON formatting output settings in the CLI
json_pretty = true

[logging]
level = "info"
```

### Configuration Schema (`mnemonic.toml`):

```toml
version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000" # Project UUID
name = "My Personal Wiki"
slug = "personal"
type = "local" # Use "local" or omit for central projects
markdown_format_version = 1

# Optional project description for MCP tool context
description = "..."

# Optional custom instructions for the MCP server
custom_instructions = "..."

created_at = 2024-11-20T12:00:00Z
updated_at = 2024-11-21T15:30:00Z

[layout]
# File globs to include in indexing
notes_glob = ["**/*.md"]
# Paths excluded from indexing
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
app_version = "v0.1.0"
```

---

## Quick Start

### 1. Initialize a Project

You can initialize a project globally (central) or within your current working directory (local).

```bash
# Central project (saved to ~/.mnemonic/personal)
mnemonic project init personal --description "My personal thoughts and logs"

# Local project (in current directory, registers a pointer)
mnemonic project init my-repo --local
```

### 2. Create and Edit Notes

Create a note via the CLI:

```bash
mnemonic notes create --title "My First Note" --tag "project" --tag "draft"
```

Edit a note (query by ID, title, path, or slug):

```bash
mnemonic notes edit "my-first-note" --append "\n- [todo] complete the setup instructions."
```

### 3. Search and View

Query your knowledge base:

```bash
# Full-text search
mnemonic notes search "todo instructions"

# Filter search results by tag
mnemonic notes search "complete" --tag "project"

# List all tags
mnemonic tags list
```

Display the contents of a note:

```bash
mnemonic notes show "my-first-note"
```

---

## Model Context Protocol (MCP) Integration

`mnemonic` can run as an MCP host to expose your local knowledge base to compatible AI assistants and clients.

### Stdio Protocol

Run the stdio adapter inside your MCP client configuration:

```bash
mnemonic stdio --project personal
```

#### Example Client Configuration

Add the following config to your compatible MCP client settings file (usually in JSON format):

```json
{
  "mcpServers": {
    "mnemonic": {
      "command": "mnemonic",
      "args": ["stdio", "--project", "personal"]
    }
  }
}
```

### HTTP/SSE Web Server

Access the MCP API over SSE endpoints.

```bash
export MNEMONIC_PROJECT_TOKEN="your-secure-token"
mnemonic web serve --port 8080
```

The server handles:

- `GET /sse` (initial SSE connection establishment)
- `POST /messages` (MCP protocol message delivery)

If `MNEMONIC_PROJECT_TOKEN` is set, requests require a Bearer authorization token.

### Read-Only Mode

Using the `--read-only` flag (or setting the environment variable `MNEMONIC_READ_ONLY=true`) launches the server with diagnostics and read tools only, preventing any modifications to note files.
