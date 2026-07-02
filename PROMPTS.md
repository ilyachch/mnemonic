# Mnemonic AI Prompts & Tool Specifications

This document provides a reference of all prompts, system instructions, and tool descriptions configured in the `mnemonic` MCP server.

---

## 1. System Instructions (Global Context)

Depending on the operational mode (standard or read-only), `mnemonic` sends specific guiding instructions to the connected AI Client during session initialization.

### Standard Mode Instructions

```text
You MUST use the mnemonic tools as your primary long-term memory.
- ALWAYS search the knowledge base using search_notes or list_notes before starting a task to gather context.
- ALWAYS write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
- Use read_note, list_tags, and list_backlinks when they help clarify the existing knowledge base.
- ALWAYS link related notes using [[Wiki-Links]].
```

### Read-Only Mode Instructions

```text
You MUST use the mnemonic tools as your primary long-term memory.
- ALWAYS search the knowledge base using search_notes or list_notes before starting a task to gather context.
- Use read_note, list_tags, and list_backlinks when they help clarify the existing knowledge base.
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

#### `read_note`

- **Description:** `Read a note by note_id, slug, path, or title.`
- **Arguments:**
  - `identifier` (string, required): The unique ID, slug, title, or relative path of the note.

#### `search_notes`

- **Description:** `Search this knowledge base.`
  _(Note: If a project `description` is provided, it is prepended to this description to give the AI agent precise search context)._
- **Arguments:**
  - `query` (string, required): FTS5 query string.
  - `limit` (integer, optional): Max search results (defaults to 20).
  - `tag` (string, optional): Filter results by tag.

#### `list_tags`

- **Description:** `List tags in this knowledge base.`
- **Arguments:**
  - `limit` (integer, optional): Max tags to return.

#### `list_backlinks`

- **Description:** `List backlinks for a note.`
- **Arguments:**
  - `identifier` (string, required): Note ID, slug, or title to find references to.
  - `limit` (integer, optional): Limit results.

#### `doctor`

- **Description:** `Run index and content health checks.`
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
  - `if_match_hash` (string, optional): Expected hash of the current file version to ensure safe write concurrency (highly recommended for `replace_body`).

#### `delete_note`

- **Description:** `Delete a note.`
- **Arguments:**
  - `identifier` (string, required): Note selector.
  - `hard_delete` (boolean, optional): Set true to remove the file permanently instead of moving it to the `.trash` directory.
  - `if_match_hash` (string, optional): Expected file hash to prevent accidental deletion of modified data.

#### `rebuild_index`

- **Description:** `Rebuild the index.`
- **Arguments:** None.
