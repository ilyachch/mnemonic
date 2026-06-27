# mnemonic

`mnemonic` is a local-first personal knowledge base and search engine. It operates as a Command Line Interface (CLI) tool and a Model Context Protocol (MCP) server, allowing you to manage, link, and search Markdown-formatted notes.

The application indexes documents into a local SQLite database utilizing FTS5 (Full-Text Search) for queries, extracts structured inline data, tracks explicit backlinks, and provides both standard stdio-based and SSE-based MCP interfaces for LLM integrations.

---

## Features

- **Local-First Architecture:** Keeps your data stored in plain text Markdown files and standard SQLite databases under your user directory.
- **Two Project Configurations:**
  - **Central:** Maintained directly under the configured home path.
  - **Local:** Pointers mapping a custom folder (such as a Git repository) containing a `mnemonic.toml` file to the registry.
- **Extracted Structural Metadata:**
  - Standard YAML Frontmatter parsing (`mnemonic_note_id`, `slug`, `tags`, etc.).
  - Inline hashtag detection (`#tag`).
  - Bullet-point observations with explicit categorizations: `- [category] content`.
  - Classic Wiki-link syntax `[[Target Note]]` or aliased links `[[Target Note|Alias]]`.
  - Explicit dependency declaration blocks via `## Relations` headers.
- **SQLite FTS5 Search Index:** Indexes title, tags, bodies, and observations to run fast local search.
- **Model Context Protocol (MCP) Support:** Provides tools for AI models to safely read, search, append to, and manage notes.
- **SSE Web Server:** Exposes the MCP server over HTTP SSE endpoints with Bearer Token authorization options.

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

`mnemonic` adheres to the XDG Base Directory Specification. You can discover your system-specific paths by running:

```bash
mnemonic config show
```

### Precedence for Config File Search:

1. `--config-file` CLI override flag.
2. `MNEMONIC_CONFIG_FILE` environment variable.
3. `$XDG_CONFIG_HOME/mnemonic/config.toml` (typically `~/.config/mnemonic/config.toml`).

### Configuration Schema (`config.toml`):

```toml
version = 1

[paths]
# Absolute or home-relative path to store central projects.
# Defaults to ~/.mnemonic if left empty.
memories_home = "~/.mnemonic"

[notes]
# Note deletion behavior. Supported values: "trash"
delete_behavior = "trash"
trash_dir_name = ".trash"

[index]
# Enables/disables full-text search, WAL mode, and locks busy timeouts
fts = true
wal = true
busy_timeout_ms = 5000

[output]
# Formats JSON outputs with indentation when using CLI flags
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
type = "local" # Use "local" or omit/leave empty for central projects
markdown_format_version = 1

# Optional description for the project, which will be injected into tools descriptions.
description = "..."

# Optional custom instructions, which will be injected into mcp server instructions.
custom_instructions = "..."

created_at = 2024-11-20T12:00:00Z
updated_at = 2024-11-21T15:30:00Z

[layout]
# File globs to index as notes
notes_glob = ["**/*.md"]
# Paths to exclude from index operations
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
app_version = "v0.1.0"
```

---

## Getting Started

### 1. Initialize a Project

You can initialize a project globally (central) or inside your current working directory (local).

```bash
# Central project (saved in ~/.mnemonic/personal)
mnemonic project init personal --description "My personal thoughts and logs"

# Local project (stored in current directory, registers a pointer)
mnemonic project init my-repo --local
```

### 2. Add or Edit Notes

To create a note with the CLI:

```bash
mnemonic notes create --title "My First Note" --tag "project" --tag "draft"
```

To edit an existing note (resolving by ID, title, path, or slug):

```bash
mnemonic notes edit "my-first-note" --append "\n- [todo] complete the setup instructions."
```

### 3. Search and View

Query your knowledge base:

```bash
# Full text search
mnemonic notes search "todo instructions"

# Filter by tag
mnemonic notes search "complete" --tag "project"

# List tags
mnemonic tags list
```

Show raw file content or structural representations:

```bash
mnemonic notes show "my-first-note"
```

---

## Model Context Protocol (MCP) Integration

`mnemonic` can be used as an MCP host to supply your local knowledge base directly to LLMs (such as Cursor, Windsurf, or Claude Desktop).

### Command-line Stdio Adapter

Run the stdio adapter inside your client configuration:

```bash
mnemonic stdio --project personal
```

#### Claude Desktop Configuration Example

Add the following to your `claude_desktop_config.json`:

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

### HTTP/SSE Server

Expose the MCP API over SSE endpoints.

```bash
export MNEMONIC_PROJECT_TOKEN="your-secure-token"
mnemonic web serve --port 8080
```

This serves:

- `GET /sse` (Initial SSE connection)
- `POST /messages` (MCP protocol messaging endpoint)

Using standard Bearer Authorization if `MNEMONIC_PROJECT_TOKEN` is declared.

### Read-Only Constraints

Add the `--read-only` flag (or set the `MNEMONIC_READ_ONLY=true` environment variable) to run the server with only diagnostic and querying tools enabled, protecting files from write mutations.
