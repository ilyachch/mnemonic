# mnemonic

`mnemonic` is a local-first personal knowledge base and search engine. It operates as a Command Line Interface (CLI) and a Model Context Protocol (MCP) server, allowing you to organize, link, and search Markdown-formatted notes.

Notes are indexed into a local SQLite database with FTS5 (Full-Text Search) for BM25-ranked queries. The application extracts structured frontmatter, inline metadata, backlinks, and exposes MCP interfaces (stdio and SSE) for AI assistant integration.

---

## Features

- **Local-first Architecture**: Plain Markdown files and an SQLite index stored in the user's workspace.
- **Markdown Frontmatter**: Every note carries YAML frontmatter with `mnemonic_note_id`, `slug`, `title`, `tags`, `summary`, `created_at`, `updated_at`, `type`, and `aliases`.
- **Unix Integer Timestamps**: `created_at` and `updated_at` are stored and returned as Unix epoch seconds (e.g. `1741737600`).
- **Slug-based Wiki-Links**: `[[target-slug]]` and `[[target-slug|Display Label]]` syntax for bidirectional linking. Standard markdown links are also supported: `[Label](target-slug.md)`.
- **Inline Metadata**:
  - Inline hashtags (`#tag`) extracted alongside frontmatter tags.
  - Observations in `- [category] text` bulleted lists under a `## Observations` heading.
  - Explicit relationships declared under a `## Relations` heading (`depends_on [[Target]]`, `relates_to [[Target]]`).
- **Multi-query Search**: Submit several FTS5 query variants per call; results are aggregated via Reciprocal Rank Fusion (RRF). Filter by tags (AND logic), creation time, and update time (absolute Unix timestamps or relative durations like `24h`).
- **Graph-aware Reranking**: Top results are multiplicatively boosted based on link connections to higher-ranked documents.
- **Related Notes**: Opt-in per-query retrieval of backlinks and forward links with `relation_type`, `source_kind`, and `direction`.
- **Batch Read**: Read multiple notes in a single `read_notes` call by providing an array of identifiers (note_id, slug, path, or title). Optional field selection controls payload size.
- **Repository Diagnostics**: `diagnose_notes` scans for invalid frontmatter, missing required fields, missing/invalid timestamps, duplicate slugs/aliases, unresolved or ambiguous wiki-links, and empty bodies. Optionally resolves broken links via search and suggests candidate targets.
- **MCP Transport**: stdio and HTTP/SSE transport with optional Bearer token authentication.

---

## Installation

```bash
go build -o mnemonic ./cmd/mnemonic
# Or install to your $GOPATH/bin
go install ./cmd/mnemonic
```

---

## Note Format

Every note is a Markdown file with YAML frontmatter. Timestamps are Unix epoch seconds as integers.

```markdown
---
mnemonic_note_id: "550e8400-e29b-41d4-a716-446655440000"
slug: my-first-note
title: My First Note
tags:
  - project
  - draft
summary: A brief description of the note content.
created_at: 1741737600
updated_at: 1741824000
type: note
aliases:
  - intro-note
---

## Summary
A high-level summary goes here.

## Notes
Main body content. Link to other notes with [[target-slug]] or [[target-slug|Alias]].
Standard markdown links: [text](target-slug.md).

## Observations
- [decision] Chose SQLite for offline search.
- [risk] Large repos may require index tuning.

## Relations
- depends_on [[deployment-guide]]
- relates_to [[project-roadmap]]
```

### Slug-based Wiki-Links

- Basic: `[[target-slug]]`
- Aliased: `[[target-slug|Display Label]]`

### Standard Markdown Links

- Labeled: `[Display Label](target-slug.md)`
- Empty label: `[](target-slug.md)`

Resolved links use the target note's slug as the identifier. A note is resolvable by slug, note_id, file path, or title.

---

## Configuration and Paths

`mnemonic` follows the XDG Base Directory Specification. Inspect active paths:

```bash
mnemonic config show
```

### Configuration File (`config.toml`)

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

### Project Manifest (`mnemonic.toml`)

```toml
version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "My Personal Wiki"
slug = "personal"
type = "local"
markdown_format_version = 1

description = "Project description for MCP tool context"
custom_instructions = "Custom MCP server instructions"

created_at = 1741737600
updated_at = 1741824000

[format]
links_style = "wiki"

[layout]
notes_glob = ["**/*.md"]
ignore = ["mnemonic.toml", ".trash/**"]

[generator]
app = "mnemonic"
app_version = "v0.1.0"
```

---

## Quick Start

### Initialize a Project

```bash
# Central project (stored under ~/.mnemonic/personal)
mnemonic project init personal --description "My personal thoughts and logs"

# Local project (pointer to current directory)
mnemonic project init my-repo --local
```

### Create and Edit Notes

```bash
# Create
mnemonic notes create --title "Deployment Guide" --tag "ops" --tag "reference"

# Append content
mnemonic notes edit "deployment-guide" --append "\n- [todo] verify rollback procedure."

# Replace entire body (requires content_hash from read_notes)
mnemonic notes edit "deployment-guide" --replace-body "New body text" --if-match-hash "abc123..."

# Merge frontmatter
mnemonic notes edit "deployment-guide" --set type=decision

# Set or replace tags
mnemonic notes edit "deployment-guide" --set-tags ops --set-tags reference

# Clear all tags
mnemonic notes edit "deployment-guide" --clear-tags

# Set or replace aliases
mnemonic notes edit "deployment-guide" --set-aliases deploy-intro

# Clear all aliases
mnemonic notes edit "deployment-guide" --clear-aliases

# Combine clear and set in one call (both are tags/aliases mode)
mnemonic notes edit "deployment-guide" --clear-tags --set-aliases current
```

### Display a Note

```bash
mnemonic notes show "deployment-guide"
mnemonic notes show "550e8400-e29b-41d4-a716-446655440000"
mnemonic notes show "notes/deployment-guide.md"
```

### Search

```bash
# Full-text search with multiple query variants
mnemonic notes search -q "deployment" -q "rollout" -q "release"

# Filter by tag (AND)
mnemonic notes search -q "pipeline" -t "ops" -t "reference"

# Time filters with relative durations
mnemonic notes search -q "design" --created-since "7d"
mnemonic notes search -q "incident" --updated-since "24h"

# Time filters with absolute Unix timestamps
mnemonic notes search -q "migration" --created-after 1741737600 --created-before 1741824000

# Include related notes (backlinks and forward links)
mnemonic notes search -q "architecture" --include-related

# Debug mode (shows path, score, content_hash)
mnemonic notes search -q "config" --debug

# Paginate
mnemonic notes search -q "notes" --limit 50
```

### List All Notes

```bash
mnemonic notes list
mnemonic notes list --limit 10
```

### List Tags

```bash
mnemonic tags list
```

### Repository Diagnostics

```bash
# Full scan
mnemonic project doctor

# Filter by kind
mnemonic project doctor --kinds unresolved_link,ambiguous_link

# Paginate
mnemonic project doctor --limit 20

# With candidate suggestions for broken links
mnemonic project doctor --include-suggestions
```

### Rebuild Index

The SQLite index is a disposable artifact derived from the Markdown notes. Incompatible index schemas are not automatically migrated or rebuilt — the application reports `index is invalid` and asks the user to run `mnemonic project reindex`.

```bash
mnemonic project reindex
```

### Run Health Checks

```bash
mnemonic doctor
```

---

## Model Context Protocol (MCP) Integration

`mnemonic` exposes your knowledge base as MCP tools for AI assistants and clients.

### Stdio

```bash
mnemonic stdio --project personal
```

Client configuration:

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

### HTTP/SSE

```bash
export MNEMONIC_PROJECT_TOKEN="your-secure-token"
mnemonic web serve --port 8080
```

Endpoints: `GET /sse`, `POST /messages`. If `MNEMONIC_PROJECT_TOKEN` is set, requests require a `Bearer` authorization header.

### Read-Only Mode

`--read-only` (or `MNEMONIC_READ_ONLY=true`) disables `create_note`, `edit_note`, `delete_note`, and `rebuild_index`, exposing only search, read, and diagnostic tools.

---

## MCP Tool Reference

### `search_notes`
Multi-query FTS5 search with time and tag filters, graph-aware reranking, and optional related-note retrieval.

| Parameter | Type | Description |
|---|---|---|
| `queries` | `[]string` | FTS5 query strings; max 8, 500 Unicode chars each (submit phrasing variants) |
| `tags` | `[]string` | Filter by tags (AND logic) |
| `created_before` | `int64` | Unix timestamp, upper bound for `created_at` |
| `created_after` | `int64` | Unix timestamp, lower bound for `created_at` |
| `updated_before` | `int64` | Unix timestamp, upper bound for `updated_at` |
| `updated_after` | `int64` | Unix timestamp, lower bound for `updated_at` |
| `created_since` | `string` | Relative duration (e.g. `"24h"`, `"7d"`) |
| `updated_since` | `string` | Relative duration (e.g. `"24h"`, `"7d"`) |
| `limit` | `int` | Max results (1–100, default 10) |
| `include_related` | `bool` | Include `related_notes` array per hit |
| `debug` | `bool` | Expose `path`, `score`, `content_hash` |

Results include `note_id`, `slug`, `title`, `snippet`, `summary`, `tags`, `matched_queries` (original query strings that matched), and optionally `related_notes`, `path`, `score`, `content_hash`.

### `read_notes`
Batch-read notes by an array of identifiers (max 50). Returns a `notes` array, a `missing` array for unresolvable identifiers, and an `issues` array with per-selector errors (ambiguous, corrupted, io_error, internal).

| Parameter | Type | Description |
|---|---|---|
| `identifiers` | `[]string` | note_ids, slugs, paths, or titles |
| `fields` | `[]string` | Select fields: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at. Timestamps are Unix seconds. |
| `max_body_chars` | `int` | Truncate body to N characters (0–100000) |

### `diagnose_notes`
Scan for metadata issues, broken links, and content problems.

| Parameter | Type | Description |
|---|---|---|
| `kinds` | `[]string` | Filter by kind (see list below) |
| `limit` | `int` | Issues per page (1–200, default 50) |
| `cursor` | `int` | Zero-based page offset (>= 0) |
| `include_suggestions` | `bool` | Resolve broken links via search |

Diagnostic kinds: `invalid_frontmatter`, `missing_required_field`, `missing_summary`, `missing_timestamp`, `invalid_timestamp`, `duplicate_slug`, `duplicate_alias`, `unresolved_link`, `ambiguous_link`, `empty_body`.

### `list_notes`
List all notes with pagination (`limit`, `cursor`).

### `list_tags`
List all tags with usage counts. `limit`: 0 = no explicit limit, positive = max results, negative = validation error.

### `list_backlinks`
List notes that link to a given note (`identifier`, `limit`). `limit`: 0 = no explicit limit, positive = max results, negative = validation error.

### `create_note`
Create a note with `title`, `body`, and `tags`.

### `edit_note`
Edit a note by `identifier` with one of `append`, `replace_body` (requires `if_match_hash`), `merge_frontmatter`, typed `tags`, or typed `aliases`.

`tags` and `aliases` are presence-aware:
- field absent → do not modify
- empty array `[]` → clear the field
- non-empty array `["a", "b"]` → replace the field with the given list

`tags` and `aliases` must not be passed through `merge_frontmatter`.

### `delete_note`
Delete or trash a note by `identifier`. `hard_delete` requires `if_match_hash`.

### `rebuild_index`
Rebuild the full-text search index from scratch.

### `doctor`
Run index and content health checks.
