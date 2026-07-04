# Mnemonic AI Prompts & Tool Specifications

This document provides a reference of all prompts, system instructions, and tool descriptions configured in the `mnemonic` MCP server.

---

## 1. System Instructions (Global Context)

Depending on the operational mode (standard or read-only), `mnemonic` sends specific guiding instructions to the connected AI Client during session initialization. The link format instruction adapts to the project's `format.links_style` setting (`wiki` or `regular`).

### Standard Mode Instructions

```text
You MUST use the mnemonic tools as your primary long-term memory.
- Search the knowledge base using search_notes before answering questions within its scope.
- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
- Batch-read all selected notes in one read_notes call.
- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
- Link related notes using [[target-slug|Display Label]].
```

With `links_style = "regular"`:
```text
- Link related notes using [Display Label](target-slug.md).
```

### Read-Only Mode Instructions

```text
You MUST use the mnemonic tools as your primary long-term memory.
- Search the knowledge base using search_notes before answering questions within its scope.
- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
- Batch-read all selected notes in one read_notes call.
- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
- This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.
```

### Dynamic Meta-Information Append

If the project configuration (`mnemonic.toml`) contains optional fields, they are automatically appended to the selected instruction set in the following format:

```text
[Base System Instructions chosen above]

Project Description:
[Value of description field]

Custom Instructions:
[Value of custom_instructions field]
```

---

## 2. MCP Tools Directory

Below is the exact list of tools exposed to the Model Context Protocol client, including their registration names, descriptions, and structural schema.

### Read Tools (Available in all modes)

#### `list_notes`

- **Description:** `List all notes in this knowledge base.`
- **Arguments:**
  - `limit` (integer, optional): Maximum notes to return per page (defaults to 20).
  - `cursor` (string, optional): Pagination offset.

#### `read_notes`

- **Description:** Batch-read one or more notes by note_id, slug, path, or title. note_id, slug, and title are always returned.
- **Arguments:**
  - `identifiers` ([]string, required): Note IDs, slugs, file paths, or titles to resolve.
  - `fields` ([]string, optional): Optional fields to include. Valid values: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at.
  - `max_body_chars` (int, optional): Truncate body to this many characters.

#### `search_notes`

- **Description:** Multi-query full-text search with time filters, tag filters, and graph-aware reranking. Results aggregated via Reciprocal Rank Fusion.
- **Arguments:**
  - `queries` ([]string, optional): FTS5 query variants for multi-query search.
  - `tags` ([]string, optional): Filter results by tags (AND logic).
  - `created_before` / `created_after` (int64, optional): Unix timestamp filters.
  - `updated_before` / `updated_after` (int64, optional): Unix timestamp filters.
  - `created_since` / `updated_since` (string, optional): Relative duration filters (e.g. "24h", "7d").
  - `limit` (integer, optional): Max results (defaults to 10).
  - `include_related` (boolean, optional): Include related notes (links and backlinks).
  - `debug` (boolean, optional): Include score, path, and content_hash in output.

#### `list_tags`

- **Description:** `List tags in this knowledge base.`
- **Arguments:**
  - `limit` (integer, optional): Max tags to return.

#### `list_backlinks`

- **Description:** `List backlinks for a note.`
- **Arguments:**
  - `identifier` (string, required): Note ID, slug, or title to find references to.
  - `limit` (integer, optional): Limit results.

#### `diagnose_notes`

- **Description:** Scan notes for issues: invalid frontmatter, missing fields, missing timestamps, invalid timestamps, duplicate slugs/aliases, unresolved/ambiguous links, empty bodies.
- **Arguments:**
  - `kinds` ([]string, optional): Filter by diagnostic kind.
  - `limit` (integer, optional): Page size (default 50).
  - `cursor` (integer, optional): Pagination offset.
  - `include_suggestions` (boolean, optional): Search for link target candidates.

#### `doctor`

- **Description:** Run index and content health checks at project level.
- **Arguments:** None.

---

### Write Tools (Disabled in Read-Only mode)

#### `create_note`

- **Description:** `Create a new note.`
- **Arguments:**
  - `title` (string, required): The title of the note.
  - `body` (string, optional): Note markdown content.
  - `tags` (array of strings, optional): Tags to insert into the YAML frontmatter.

#### `edit_note`

- **Description:** `Edit an existing note.`
- **Arguments:**
  - `identifier` (string, required): The note selector (ID, slug, path, or title).
  - `append` (string, optional): Text to append to the end of the markdown body.
  - `replace_body` (string, optional): New text to completely overwrite the body.
  - `merge_frontmatter` (object, optional): Key-value string map to update or add frontmatter metadata.
  - `if_match_hash` (string, optional): Expected hash of the current file version to ensure safe write concurrency.

#### `delete_note`

- **Description:** `Delete a note.`
- **Arguments:**
  - `identifier` (string, required): Note selector.
  - `hard_delete` (boolean, optional): Set true to remove the file permanently instead of moving it to the `.trash` directory.
  - `if_match_hash` (string, optional): Expected file hash to prevent accidental deletion of modified data.

#### `rebuild_index`

- **Description:** `Rebuild the index.`
- **Arguments:** None.
