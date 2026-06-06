# mnemonic

`mnemonic` is a local-first personal knowledge base, indexer, and search engine. It operates as both a command-line interface (CLI) and a Model Context Protocol (MCP) server, designed to parse Markdown notes, track wiki-links, catalog metadata, and offer fast search capabilities using an embedded SQLite database.

## Key Features

- **Local-First Markdown Indexing**: Automatically scans directories for Markdown notes, parses structured YAML frontmatter, and updates a local relational SQLite database.
- **Model Context Protocol (MCP) Support**: Exposes read and write capabilities (`list_notes`, `read_note`, `search_notes`, `create_note`, `edit_note`, etc.) to LLM environments (such as Claude Desktop) using standard JSON-RPC over `stdio`.
- **Relational Graph & Backlinks**: Extracts standard wiki-links (`[[Target Note]]`) and typed relations (e.g., `depends_on`, `relates_to` listed under a `## Relations` section) to build a graph of your notes.
- **Deduplicated Tagging & Observations**: Indexes tags from frontmatter YAML, inline hashtags (e.g., `#tag`), and categorizes structured observations (e.g., `- [decision] Description #tag`).
- **Full-Text Search (FTS5)**: Leverages SQLite FTS5 with unicode61 tokenization for responsive, query-based search of note contents, titles, and tags.
- **Data Integrity & Safety**: Uses flock-based file locking, atomic file staging to prevent corruption on partial writes, and SQLite WAL (Write-Ahead Logging) mode.

---

## Installation

To build `mnemonic` from source, ensure you have Go (1.21 or later) installed:

```bash
git clone https://github.com/ilyachch/mnemonic.git
cd mnemonic
go build ./cmd/mnemonic
```

To install directly to your `$GOPATH/bin`:

```bash
go install github.com/ilyachch/mnemonic/cmd/mnemonic@latest
```

---

## Getting Started

### 1. Initialize a Project
Create a new project in your current directory. Choosing `--local` places your markdown files in a hidden directory (`.mnemonic-memories/<project-name>`) inside your current folder:

```bash
mnemonic init my-notes --local
```

### 2. Create Your First Note
Create a note with a title and optional tags:

```bash
mnemonic notes create --project my-notes --title "System Architecture" --tag "design" --tag "ops"
```

You can also pass content via standard input or from a file:

```bash
echo -e "## Summary\nThis outlines the database structure." | mnemonic notes create --project my-notes --title "Database Schema" --stdin
```

### 3. Rebuild the Search Index
`mnemonic` decouples indexing from editing to optimize CLI operations. Run `reindex` to sync your directory changes with the SQLite search database:

```bash
mnemonic project reindex
```

### 4. Search and Inspect Notes
Once indexed, you can perform full-text queries:

```bash
mnemonic notes search "database structure" --project my-notes
```

Show a note's complete contents:

```bash
mnemonic notes show "system-architecture" --project my-notes
```

---

## CLI Usage

`mnemonic` commands generally offer both human-readable and `--json` structured outputs.

### Project Management
- `mnemonic project list`: Lists all registered projects along with their disk paths, index states, and configuration info.
- `mnemonic project show <NAME_OR_UUID>`: Shows metadata details for a registered project.
- `mnemonic project doctor`: Runs structural integrity diagnostic checks on the registry, project manifests, SQLite schemas, notes, and temporary file states.
- `mnemonic project discover`: Scans your global memories directory for standalone project configurations and registers them.
- `mnemonic project import <PATH>`: Registers an existing project found at the specified path.
- `mnemonic project remove <NAME_OR_UUID>`: Safely unregisters a project. Pass `--delete-markdown` to also purge the associated text notes.

### Note Management
- `mnemonic notes list`: Lists note metadata in the current project (excluding trashed items).
- `mnemonic notes show <SELECTOR>`: Displays note metadata, frontmatter, and contents. Selectors can be a UUID, slug, path, or title.
- `mnemonic notes edit <SELECTOR>`: Appends body text or updates specific frontmatter fields safely.
- `mnemonic notes delete <SELECTOR>`: Moves a note to the project's `.trash` directory or permanently deletes it with `--hard --yes`.
- `mnemonic notes backlinks <SELECTOR>`: Queries the database to list all other notes referencing the targeted note.
- `mnemonic tags list`: Groups and counts note tags.

---

## Model Context Protocol (MCP) Server

You can configure `mnemonic` as an assistant tool for LLMs. The server communicates via standard I/O (stdio).

### Run the Server
Select your active project to start serving:

```bash
mnemonic mcp --project my-notes
```

### Integration with Claude Desktop
To interface `mnemonic` with Claude Desktop, add the tool configuration to your `claude_desktop_config.json` (typically located in `%APPDATA%\Claude` on Windows or `~/Library/Application Support/Claude` on macOS):

```json
{
  "mcpServers": {
    "mnemonic": {
      "command": "mnemonic",
      "args": ["mcp", "--project", "my-notes"],
      "env": {
        "MNEMONIC_MEMORIES_HOME": "/path/to/your/memories/home"
      }
    }
  }
}
```

### Available MCP Tools
Once connected, the client gains access to the following tools:
- `list_notes`: Retrieve paginated note listings.
- `read_note`: Query a note's metadata and body using a selector.
- `search_notes`: Run full-text FTS5 search queries.
- `list_backlinks`: Map references pointing back to a note.
- `list_tags`: Lists tags with their document counts.
- `create_note`: Writes a new note to disk.
- `edit_note`: Safely performs appends, body replacements, or frontmatter edits.
- `delete_note`: Soft-trashes or hard-deletes files.

---

## Configuration

`mnemonic` follows the XDG Base Directory Specification. It determines configuration files using the following precedence order:

1. Explicit command line flag: `--config <path>`
2. Environment Variable: `MNEMONIC_CONFIG_FILE`
3. Environment Variable: `MNEMONIC_CONFIG_HOME/mnemonic/config.toml`
4. Standard XDG config path: `$XDG_CONFIG_HOME/mnemonic/config.toml` (defaulting to `~/.config/mnemonic/config.toml`)

An example `config.toml` file:

```toml
version = 1

[paths]
# Explicitly direct where your non-local notes reside
memories_home = "~/.mnemonic"

[notes]
# Options: "trash" (default) or "delete"
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
