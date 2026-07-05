# Изменения в ветке `fixes-and-improvements` относительно `main`

## Список измененных файлов:
- `AGENTS.md`
- `ARCHITECTURE.md`
- `PROMPTS.md`
- `README.md`
- `internal/adapter/cli/notes_edit.go`
- `internal/adapter/cli/notes_search.go`
- `internal/adapter/cli/project_add.go`
- `internal/adapter/cli/project_doctor.go`
- `internal/adapter/cli/project_import.go`
- `internal/adapter/cli/project_init.go`
- `internal/adapter/cli/project_list.go`
- `internal/adapter/cli/project_reindex.go`
- `internal/adapter/cli/project_remove.go`
- `internal/adapter/cli/project_show.go`
- `internal/adapter/cli/project_sync.go`
- `internal/adapter/cli/root.go`
- `internal/adapter/cli/runtime.go`
- `internal/adapter/cli/stdio.go`
- `internal/adapter/cli/tags_list.go`
- `internal/adapter/cli/web_serve.go`
- `internal/adapter/stdio/server.go`
- `internal/adapter/stdio/tools.go`
- `internal/adapter/web/manager.go`
- `internal/app/app.go`
- `internal/app/runtime.go`
- `internal/apperr/errors.go`
- `internal/domain/kb/knowledge_base.go`
- `internal/format/manifest/manifest.go`
- `internal/format/markdown/README.md`
- `internal/format/markdown/errors.go`
- `internal/format/markdown/note.go`
- `internal/format/markdown/render.go`
- `internal/format/markdown/testdata/basic-memory/permalink.md`
- `internal/service/catalogsvc/service.go`
- `internal/service/indexsvc/diagnostics.go`
- `internal/service/indexsvc/service.go`
- `internal/service/maintsvc/service.go`
- `internal/service/notesvc/service.go`
- `internal/service/searchsvc/service.go`
- `internal/store/markdownstore/store.go`
- `internal/store/registry/store.go`
- `internal/store/sqliteindex/rebuild.go`
- `internal/store/sqliteindex/scan.go`
- `internal/store/sqliteindex/schema.go`
- `internal/store/sqliteindex/schema_check.go`
- `internal/store/sqliteindex/store.go`
- `scripts/mcp_smoke.go`

---

## Файл: `AGENTS.md`

```diff
diff --git a/AGENTS.md b/AGENTS.md
index 9bdc840..27e8f63 100644
--- a/AGENTS.md
+++ b/AGENTS.md
@@ -63,7 +63,8 @@ Direct usage of `time.Now()` or external UUID generators within services or doma

 Errors must be wrapped in the `apperr.Error` struct to return the correct exit codes in the CLI.

-- Use the helper constructors: `apperr.CLIUsage()`, `apperr.NotFound()`, `apperr.Unsafe()`, `apperr.Corrupted()`.
+- Use the helper constructors: `apperr.CLIUsage()`, `apperr.NotFound()`, `apperr.Unsafe()`, `apperr.Corrupted()`, `apperr.IO()`, `apperr.Ambiguous()`.
+- **Do not classify errors by message text.** Classification must use type assertions (`errors.As`) or `apperr.Code`. Code that inspects `err.Error()` with `HasPrefix`, `HasSuffix`, or `Contains` to determine the error category is prohibited.
 - Avoid direct calls to `panic()`; errors should be handled at the adapter boundaries.

 ### 3. Safe File Writes
@@ -82,6 +83,12 @@ Errors must be wrapped in the `apperr.Error` struct to return the correct exit c
 - The project uses the `modernc.org/sqlite` driver to avoid CGO dependencies.
 - All SQL queries must remain compatible with the SQLite3 specification.

+### 6. Index Schema Validation
+
+- `ValidateSchema()` in `internal/store/sqliteindex/schema_check.go` performs **structural** checking only (table and column presence via `PRAGMA table_info`).
+- It does **not** use version numbers, `PRAGMA user_version`, or any migration-like mechanisms.
+- The index is a disposable derived artifact — incompatibility is resolved by an explicit `mnemonic project reindex`, never automatically.
+
 ---

 ## Guidelines for Extending Code
```

## Файл: `ARCHITECTURE.md`

```diff
diff --git a/ARCHITECTURE.md b/ARCHITECTURE.md
index 65957d1..999ee84 100644
--- a/ARCHITECTURE.md
+++ b/ARCHITECTURE.md
@@ -74,7 +74,7 @@ The registry is implemented as a flat file layout within the `MemoriesHome` dire

 - **Central Projects:** Represented as subdirectories containing a `mnemonic.toml` manifest file.
 - **Local Projects:** Represented as pointer files (`[slug].toml`) containing the absolute path to the `mnemonic.toml` manifest situated in an external workspace (e.g., a development Git repository).
-- **Resilience to Corruption:** Errors reading individual manifests or the migration of external local project directories do not disrupt registry scanning. Problematic projects are flagged as `[CORRUPTED]` or `[ORPHANED/MISSING]` in list outputs.
+- **Resilience to Corruption:** Errors reading individual manifests or the relocation of external local project directories do not disrupt registry scanning. Problematic projects are flagged as `[CORRUPTED]` or `[ORPHANED/MISSING]` in list outputs.

 ### State Files

@@ -117,16 +117,26 @@ Markdown Files ──► Parser ──► NoteDoc ──► index.new.sqlite ─

 ### Index Rebuild Lifecycle

-The index database schema is treated as immutable. If schema version mismatches or corruption are detected, the database is rebuilt from scratch:
+The index is always rebuilt from scratch. Each rebuild creates a new temporary database, populates it, and atomically replaces the old index file:

 1. All Markdown files in the project's root directory are scanned (excluding system files and the `.trash` directory).
 2. Metadata from the YAML frontmatter, wikilinks, inline tags, observations, and declared relations are extracted from each document.
 3. A temporary database file (`index.new.sqlite`) is created.
-4. The schema tables (`notes`, `note_tags`, `observations`, `links`, `notes_fts`) are applied.
+4. The schema tables (`notes`, `note_tags`, `note_aliases`, `observations`, `links`, `notes_fts`) are applied.
 5. Data is written to the temporary database, and relations between notes are resolved.
 6. Structural integrity is validated via `PRAGMA quick_check`.
 7. The connection is closed, old index files (including `-wal` and `-shm`) are deleted, and the temporary file is atomically renamed to the primary file name.

+### Index Validation
+
+When opening an existing index for read operations, the application validates that all required tables and columns are present (`ValidateSchema` in `internal/store/sqliteindex/schema_check.go`).
+
+- Index validation is **structural only** — it does not use version numbers or `PRAGMA user_version`.
+- An incompatible index is **rejected** with the message `index is invalid; run \`mnemonic project reindex\``.
+- The application **never** performs automatic schema migration or implicit rebuild.
+- The index is a disposable derived artifact; repair is always a manual explicit `mnemonic project reindex`.
+- `index_runs` is not part of the read contract and is only used as internal rebuild bookkeeping.
+
 ---

 ## 5. Scope of Responsibility and Package Import Rules
```

## Файл: `PROMPTS.md`

```diff
diff --git a/PROMPTS.md b/PROMPTS.md
index 0322d40..9934ee2 100644
--- a/PROMPTS.md
+++ b/PROMPTS.md
@@ -6,24 +6,35 @@ This document provides a reference of all prompts, system instructions, and tool

 ## 1. System Instructions (Global Context)

-Depending on the operational mode (standard or read-only), `mnemonic` sends specific guiding instructions to the connected AI Client during session initialization.
+Depending on the operational mode (standard or read-only), `mnemonic` sends specific guiding instructions to the connected AI Client during session initialization. The link format instruction adapts to the project's `format.links_style` setting (`wiki` or `regular`).

 ### Standard Mode Instructions

 ```text
 You MUST use the mnemonic tools as your primary long-term memory.
-- ALWAYS search the knowledge base using search_notes or list_notes before starting a task to gather context.
-- ALWAYS write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
-- Use read_note, list_tags, and list_backlinks when they help clarify the existing knowledge base.
-- ALWAYS link related notes using [[Wiki-Links]].
+- Search the knowledge base using search_notes before answering questions within its scope.
+- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
+- Batch-read all selected notes in one read_notes call.
+- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
+- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
+- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
+- Link related notes using [[target-slug|Display Label]].
+```
+
+With `links_style = "regular"`:
+```text
+- Link related notes using [Display Label](target-slug.md).
 ```

 ### Read-Only Mode Instructions

 ```text
 You MUST use the mnemonic tools as your primary long-term memory.
-- ALWAYS search the knowledge base using search_notes or list_notes before starting a task to gather context.
-- Use read_note, list_tags, and list_backlinks when they help clarify the existing knowledge base.
+- Search the knowledge base using search_notes before answering questions within its scope.
+- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
+- Batch-read all selected notes in one read_notes call.
+- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
+- Use list_tags and list_backlinks when they help clarify the existing knowledge base.
 - This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.
 ```

@@ -56,37 +67,52 @@ Below is the exact list of tools exposed to the Model Context Protocol client, i
   - `limit` (integer, optional): Maximum notes to return per page (defaults to 20).
   - `cursor` (string, optional): Pagination offset.

-#### `read_note`
+#### `read_notes`

-- **Description:** `Read a note by note_id, slug, path, or title.`
+- **Description:** Batch-read one or more notes by note_id, slug, path, or title (max 50 identifiers). Unresolved identifiers are returned in `missing`. Per-selector errors are reported in `issues` with kind: ambiguous, corrupted, io_error, internal.
 - **Arguments:**
-  - `identifier` (string, required): The unique ID, slug, title, or relative path of the note.
+  - `identifiers` ([]string, required): Note IDs, slugs, file paths, or titles to resolve.
+  - `fields` ([]string, optional): Optional fields to include. Valid values: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at. Timestamps are Unix seconds.
+  - `max_body_chars` (int, optional): Truncate body to this many characters (max 100000).

 #### `search_notes`

-- **Description:** `Search this knowledge base.`
-  _(Note: If a project `description` is provided, it is prepended to this description to give the AI agent precise search context)._
+- **Description:** Multi-query full-text search with time filters, tag filters, and graph-aware reranking. Results are combined via Reciprocal Rank Fusion. Each hit includes note_id, slug, title, snippet, summary, tags, and matched_queries (the original user-supplied query strings that matched).
 - **Arguments:**
-  - `query` (string, required): FTS5 query string.
-  - `limit` (integer, optional): Max search results (defaults to 20).
-  - `tag` (string, optional): Filter results by tag.
+  - `queries` ([]string, optional): FTS5 query variants (max 8, 500 Unicode characters each).
+  - `tags` ([]string, optional): Filter results by tags (AND logic).
+  - `created_before` / `created_after` (int64, optional): Unix timestamp filters.
+  - `updated_before` / `updated_after` (int64, optional): Unix timestamp filters.
+  - `created_since` / `updated_since` (string, optional): Relative duration filters (e.g. "24h", "7d").
+  - `limit` (integer, optional): Max results (1–100, defaults to 10).
+  - `include_related` (boolean, optional): Include related notes (links and backlinks).
+  - `debug` (boolean, optional): Include score, path, and content_hash in output.

 #### `list_tags`

 - **Description:** `List tags in this knowledge base.`
 - **Arguments:**
-  - `limit` (integer, optional): Max tags to return.
+  - `limit` (integer, optional): Max tags to return. 0 = no limit, positive = maximum count, negative = validation error.

 #### `list_backlinks`

 - **Description:** `List backlinks for a note.`
 - **Arguments:**
   - `identifier` (string, required): Note ID, slug, or title to find references to.
-  - `limit` (integer, optional): Limit results.
+  - `limit` (integer, optional): Limit results. 0 = no limit, positive = maximum count, negative = validation error.
+
+#### `diagnose_notes`
+
+- **Description:** Scan notes for metadata and content issues. Supported kinds: invalid_frontmatter, missing_required_field, missing_summary, missing_timestamp, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body. Candidate suggestions for broken links are available via `include_suggestions`.
+- **Arguments:**
+  - `kinds` ([]string, optional): Filter by diagnostic kind.
+  - `limit` (integer, optional): Page size (1–200, default 50).
+  - `cursor` (integer, optional): Zero-based pagination offset (must be >= 0).
+  - `include_suggestions` (boolean, optional): Search for link target candidates.

 #### `doctor`

-- **Description:** `Run index and content health checks.`
+- **Description:** Run index and content health checks at project level.
 - **Arguments:** None.

 ---
@@ -108,8 +134,10 @@ Below is the exact list of tools exposed to the Model Context Protocol client, i
   - `identifier` (string, required): The note selector (ID, slug, path, or title).
   - `append` (string, optional): Text to append to the end of the markdown body.
   - `replace_body` (string, optional): New text to completely overwrite the body.
-  - `merge_frontmatter` (object, optional): Key-value string map to update or add frontmatter metadata.
-  - `if_match_hash` (string, optional): Expected hash of the current file version to ensure safe write concurrency (highly recommended for `replace_body`).
+  - `merge_frontmatter` (object, optional): Key-value string map to update or add frontmatter metadata. Must not set `tags` or `aliases` — use the typed fields instead.
+  - `tags` ([]string, optional): Replace the tags list. Field absent = no change, empty array = clear, non-empty = replace.
+  - `aliases` ([]string, optional): Replace the aliases list. Field absent = no change, empty array = clear, non-empty = replace.
+  - `if_match_hash` (string, optional): Expected hash of the current file version to ensure safe write concurrency.

 #### `delete_note`
```

## Файл: `README.md`

```diff
diff --git a/README.md b/README.md
index d67fa8a..2ef1ed6 100644
--- a/README.md
+++ b/README.md
@@ -10,17 +10,17 @@ Notes are indexed into a local SQLite database with FTS5 (Full-Text Search) for

 - **Local-first Architecture**: Plain Markdown files and an SQLite index stored in the user's workspace.
 - **Markdown Frontmatter**: Every note carries YAML frontmatter with `mnemonic_note_id`, `slug`, `title`, `tags`, `summary`, `created_at`, `updated_at`, `type`, and `aliases`.
-- **Unix Integer Timestamps**: `created_at` and `updated_at` are stored as Unix epoch seconds (e.g. `1741737600`), parsed as integers, and returned in RFC 3339 format by tool interfaces.
+- **Unix Integer Timestamps**: `created_at` and `updated_at` are stored and returned as Unix epoch seconds (e.g. `1741737600`).
 - **Slug-based Wiki-Links**: `[[target-slug]]` and `[[target-slug|Display Label]]` syntax for bidirectional linking. Standard markdown links are also supported: `[Label](target-slug.md)`.
 - **Inline Metadata**:
   - Inline hashtags (`#tag`) extracted alongside frontmatter tags.
   - Observations in `- [category] text` bulleted lists under a `## Observations` heading.
   - Explicit relationships declared under a `## Relations` heading (`depends_on [[Target]]`, `relates_to [[Target]]`).
-- **Multi-query Search**: Submit several FTS5 query variants per call; each contributes to the combined BM25 ranking. Filter by tags, creation time, and update time (absolute Unix timestamps or relative durations like `24h`).
-- **Graph-aware Reranking**: Results are reranked using page-rank over the [[Wiki-Link]] graph.
-- **Related Notes**: Opt-in per-query retrieval of backlinks and forward links with their `relation_type`.
-- **Batch Read**: Read multiple notes in a single `read_notes` call by providing an array of identifiers (note_id, slug, path, or title).
-- **Repository Diagnostics**: `diagnose_notes` scans for invalid frontmatter, missing required fields, duplicate slugs/aliases, unresolved or ambiguous wiki-links, and empty bodies. Optionally resolves broken links via search and suggests candidate targets.
+- **Multi-query Search**: Submit several FTS5 query variants per call; results are aggregated via Reciprocal Rank Fusion (RRF). Filter by tags (AND logic), creation time, and update time (absolute Unix timestamps or relative durations like `24h`).
+- **Graph-aware Reranking**: Top results are multiplicatively boosted based on link connections to higher-ranked documents.
+- **Related Notes**: Opt-in per-query retrieval of backlinks and forward links with `relation_type`, `source_kind`, and `direction`.
+- **Batch Read**: Read multiple notes in a single `read_notes` call by providing an array of identifiers (note_id, slug, path, or title). Optional field selection controls payload size.
+- **Repository Diagnostics**: `diagnose_notes` scans for invalid frontmatter, missing required fields, missing/invalid timestamps, duplicate slugs/aliases, unresolved or ambiguous wiki-links, and empty bodies. Optionally resolves broken links via search and suggests candidate targets.
 - **MCP Transport**: stdio and HTTP/SSE transport with optional Bearer token authentication.

 ---
@@ -133,6 +133,9 @@ custom_instructions = "Custom MCP server instructions"
 created_at = 1741737600
 updated_at = 1741824000

+[format]
+links_style = "wiki"
+
 [layout]
 notes_glob = ["**/*.md"]
 ignore = ["mnemonic.toml", ".trash/**"]
@@ -169,7 +172,22 @@ mnemonic notes edit "deployment-guide" --append "\n- [todo] verify rollback proc
 mnemonic notes edit "deployment-guide" --replace-body "New body text" --if-match-hash "abc123..."

 # Merge frontmatter
-mnemonic notes edit "deployment-guide" --set tags="ops,infra"
+mnemonic notes edit "deployment-guide" --set type=decision
+
+# Set or replace tags
+mnemonic notes edit "deployment-guide" --set-tags ops --set-tags reference
+
+# Clear all tags
+mnemonic notes edit "deployment-guide" --clear-tags
+
+# Set or replace aliases
+mnemonic notes edit "deployment-guide" --set-aliases deploy-intro
+
+# Clear all aliases
+mnemonic notes edit "deployment-guide" --clear-aliases
+
+# Combine clear and set in one call (both are tags/aliases mode)
+mnemonic notes edit "deployment-guide" --clear-tags --set-aliases current
 ```

 ### Display a Note
@@ -223,22 +241,24 @@ mnemonic tags list

 ```bash
 # Full scan
-mnemonic notes diagnose
+mnemonic project doctor

 # Filter by kind
-mnemonic notes diagnose --kinds unresolved_link,ambiguous_link
+mnemonic project doctor --kinds unresolved_link,ambiguous_link

 # Paginate
-mnemonic notes diagnose --limit 20
+mnemonic project doctor --limit 20

 # With candidate suggestions for broken links
-mnemonic notes diagnose --include-suggestions
+mnemonic project doctor --include-suggestions
 ```

 ### Rebuild Index

+The SQLite index is a disposable artifact derived from the Markdown notes. Incompatible index schemas are not automatically migrated or rebuilt — the application reports `index is invalid` and asks the user to run `mnemonic project reindex`.
+
 ```bash
-mnemonic index rebuild
+mnemonic project reindex
 ```

 ### Run Health Checks
@@ -294,7 +314,7 @@ Multi-query FTS5 search with time and tag filters, graph-aware reranking, and op

 | Parameter | Type | Description |
 |---|---|---|
-| `queries` | `[]string` | FTS5 query strings; submit phrasing variants |
+| `queries` | `[]string` | FTS5 query strings; max 8, 500 Unicode chars each (submit phrasing variants) |
 | `tags` | `[]string` | Filter by tags (AND logic) |
 | `created_before` | `int64` | Unix timestamp, upper bound for `created_at` |
 | `created_after` | `int64` | Unix timestamp, lower bound for `created_at` |
@@ -302,20 +322,20 @@ Multi-query FTS5 search with time and tag filters, graph-aware reranking, and op
 | `updated_after` | `int64` | Unix timestamp, lower bound for `updated_at` |
 | `created_since` | `string` | Relative duration (e.g. `"24h"`, `"7d"`) |
 | `updated_since` | `string` | Relative duration (e.g. `"24h"`, `"7d"`) |
-| `limit` | `int` | Max results (default 20) |
+| `limit` | `int` | Max results (1–100, default 10) |
 | `include_related` | `bool` | Include `related_notes` array per hit |
 | `debug` | `bool` | Expose `path`, `score`, `content_hash` |

+Results include `note_id`, `slug`, `title`, `snippet`, `summary`, `tags`, `matched_queries` (original query strings that matched), and optionally `related_notes`, `path`, `score`, `content_hash`.
+
 ### `read_notes`
-Batch-read notes by an array of identifiers.
+Batch-read notes by an array of identifiers (max 50). Returns a `notes` array, a `missing` array for unresolvable identifiers, and an `issues` array with per-selector errors (ambiguous, corrupted, io_error, internal).

 | Parameter | Type | Description |
 |---|---|---|
 | `identifiers` | `[]string` | note_ids, slugs, paths, or titles |
-| `fields` | `[]string` | Limit output fields |
-| `max_body_chars` | `int` | Truncate body to N characters |
-
-Returns `notes` array and a `missing` array for unresolvable identifiers.
+| `fields` | `[]string` | Select fields: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at. Timestamps are Unix seconds. |
+| `max_body_chars` | `int` | Truncate body to N characters (0–100000) |

 ### `diagnose_notes`
 Scan for metadata issues, broken links, and content problems.
@@ -323,32 +343,39 @@ Scan for metadata issues, broken links, and content problems.
 | Parameter | Type | Description |
 |---|---|---|
 | `kinds` | `[]string` | Filter by kind (see list below) |
-| `limit` | `int` | Issues per page (default 50) |
-| `cursor` | `int` | Zero-based page offset |
+| `limit` | `int` | Issues per page (1–200, default 50) |
+| `cursor` | `int` | Zero-based page offset (>= 0) |
 | `include_suggestions` | `bool` | Resolve broken links via search |

-Diagnostic kinds: `invalid_frontmatter`, `missing_required_field`, `missing_summary`, `invalid_timestamp`, `duplicate_slug`, `duplicate_alias`, `unresolved_link`, `ambiguous_link`, `empty_body`.
+Diagnostic kinds: `invalid_frontmatter`, `missing_required_field`, `missing_summary`, `missing_timestamp`, `invalid_timestamp`, `duplicate_slug`, `duplicate_alias`, `unresolved_link`, `ambiguous_link`, `empty_body`.

 ### `list_notes`
 List all notes with pagination (`limit`, `cursor`).

 ### `list_tags`
-List all tags with usage counts.
+List all tags with usage counts. `limit`: 0 = no explicit limit, positive = max results, negative = validation error.

 ### `list_backlinks`
-List notes that link to a given note (`identifier`, `limit`).
+List notes that link to a given note (`identifier`, `limit`). `limit`: 0 = no explicit limit, positive = max results, negative = validation error.

 ### `create_note`
 Create a note with `title`, `body`, and `tags`.

 ### `edit_note`
-Edit a note by `identifier` with one of `append`, `replace_body` (requires `if_match_hash`), or `merge_frontmatter`.
+Edit a note by `identifier` with one of `append`, `replace_body` (requires `if_match_hash`), `merge_frontmatter`, typed `tags`, or typed `aliases`.
+
+`tags` and `aliases` are presence-aware:
+- field absent → do not modify
+- empty array `[]` → clear the field
+- non-empty array `["a", "b"]` → replace the field with the given list
+
+`tags` and `aliases` must not be passed through `merge_frontmatter`.

 ### `delete_note`
 Delete or trash a note by `identifier`. `hard_delete` requires `if_match_hash`.

 ### `rebuild_index`
-Rebuild the full-text search index.
+Rebuild the full-text search index from scratch.

 ### `doctor`
 Run index and content health checks.
```

## Файл: `internal/adapter/cli/notes_edit.go`

```go
diff --git a/internal/adapter/cli/notes_edit.go b/internal/adapter/cli/notes_edit.go
index 53fbb19..402fa79 100644
--- a/internal/adapter/cli/notes_edit.go
+++ b/internal/adapter/cli/notes_edit.go
@@ -21,6 +21,10 @@ func newNotesEditCommand() *cobra.Command {
 	cmd.Flags().String("body-file", "", "replace the note body with the contents of a file")
 	cmd.Flags().String("if-match", "", "only update if the current content hash matches")
 	cmd.Flags().StringArray("set", nil, "set a frontmatter field")
+	cmd.Flags().StringArray("set-tags", nil, "replace the tags list")
+	cmd.Flags().StringArray("set-aliases", nil, "replace the aliases list")
+	cmd.Flags().Bool("clear-tags", false, "clear all tags")
+	cmd.Flags().Bool("clear-aliases", false, "clear all aliases")
 	return cmd
 }

@@ -39,6 +43,8 @@ func runNotesEdit(cmd *cobra.Command, args []string) error {
 		Selector: args[0],
 		Set:      parsed.setFields,
 		IfMatch:  parsed.ifMatch,
+		Tags:     parsed.setTags,
+		Aliases:  parsed.setAliases,
 	}
 	if parsed.bodyFile != "" {
 		body, readErr := os.ReadFile(parsed.bodyFile)
@@ -67,6 +73,8 @@ type notesEditFlags struct {
 	bodyFile   string
 	ifMatch    string
 	setFields  map[string]string
+	setTags    *[]string
+	setAliases *[]string
 }

 func parseNotesEditFlags(cmd *cobra.Command) (notesEditFlags, error) {
@@ -82,24 +90,98 @@ func parseNotesEditFlags(cmd *cobra.Command) (notesEditFlags, error) {
 	if err != nil {
 		return notesEditFlags{}, err
 	}
-	var setFields map[string]string
-	if cmd.Flags().Changed("set") {
-		setValues, setErr := cmd.Flags().GetStringArray("set")
-		if setErr != nil {
-			return notesEditFlags{}, setErr
-		}
-		setFields, err = parseEditSetValues(setValues)
-		if err != nil {
-			return notesEditFlags{}, err
-		}
+	setFields, err := parseSetFieldsFlag(cmd)
+	if err != nil {
+		return notesEditFlags{}, err
 	}
-	if appendText == "" && bodyFile == "" && len(setFields) == 0 {
-		return notesEditFlags{}, apperr.CLIUsage("edit requires --append, --body-file, or --set", nil)
+	setTags, err := parseSetTagsFlag(cmd)
+	if err != nil {
+		return notesEditFlags{}, err
+	}
+	setAliases, err := parseSetAliasesFlag(cmd)
+	if err != nil {
+		return notesEditFlags{}, err
+	}
+	return validateEditFlags(appendText, bodyFile, setFields, setTags, setAliases, ifMatch)
+}
+
+func parseSetFieldsFlag(cmd *cobra.Command) (map[string]string, error) {
+	if !cmd.Flags().Changed("set") {
+		return nil, nil
+	}
+	setValues, err := cmd.Flags().GetStringArray("set")
+	if err != nil {
+		return nil, err
+	}
+	return parseEditSetValues(setValues)
+}
+
+func parseSetTagsFlag(cmd *cobra.Command) (*[]string, error) {
+	if !cmd.Flags().Changed("set-tags") && !cmd.Flags().Changed("clear-tags") {
+		return nil, nil
+	}
+	if cmd.Flags().Changed("set-tags") && cmd.Flags().Changed("clear-tags") {
+		return nil, apperr.CLIUsage("--set-tags and --clear-tags cannot be combined", nil)
+	}
+	clearTags, err := cmd.Flags().GetBool("clear-tags")
+	if err != nil {
+		return nil, err
+	}
+	if clearTags {
+		return &[]string{}, nil
+	}
+	tagsVal, err := cmd.Flags().GetStringArray("set-tags")
+	if err != nil {
+		return nil, err
+	}
+	return &tagsVal, nil
+}
+
+func parseSetAliasesFlag(cmd *cobra.Command) (*[]string, error) {
+	if !cmd.Flags().Changed("set-aliases") && !cmd.Flags().Changed("clear-aliases") {
+		return nil, nil
+	}
+	if cmd.Flags().Changed("set-aliases") && cmd.Flags().Changed("clear-aliases") {
+		return nil, apperr.CLIUsage("--set-aliases and --clear-aliases cannot be combined", nil)
+	}
+	clearAliases, err := cmd.Flags().GetBool("clear-aliases")
+	if err != nil {
+		return nil, err
+	}
+	if clearAliases {
+		return &[]string{}, nil
+	}
+	aliasesVal, err := cmd.Flags().GetStringArray("set-aliases")
+	if err != nil {
+		return nil, err
+	}
+	return &aliasesVal, nil
+}
+
+func validateEditFlags(appendText, bodyFile string, setFields map[string]string, setTags, setAliases *[]string, ifMatch string) (notesEditFlags, error) {
+	hasContent := appendText != "" || bodyFile != "" || len(setFields) > 0 || setTags != nil || setAliases != nil
+	if !hasContent {
+		return notesEditFlags{}, apperr.CLIUsage("edit requires --append, --body-file, --set, --set-tags, --set-aliases, --clear-tags, or --clear-aliases", nil)
 	}
 	if appendText != "" && bodyFile != "" {
 		return notesEditFlags{}, apperr.CLIUsage("--append and --body-file cannot be combined", nil)
 	}
-	return notesEditFlags{appendText: appendText, bodyFile: bodyFile, ifMatch: ifMatch, setFields: setFields}, nil
+
+	modeCount := 0
+	if appendText != "" || bodyFile != "" {
+		modeCount++
+	}
+	if len(setFields) > 0 {
+		modeCount++
+	}
+	if setTags != nil || setAliases != nil {
+		modeCount++
+	}
+	if modeCount > 1 {
+		return notesEditFlags{}, apperr.CLIUsage("edit modes append/body-file, --set, and tags/aliases are mutually exclusive", nil)
+	}
+
+	return notesEditFlags{appendText: appendText, bodyFile: bodyFile, ifMatch: ifMatch, setFields: setFields, setTags: setTags, setAliases: setAliases}, nil
 }

 func parseEditSetValues(values []string) (map[string]string, error) {
```

## Файл: `internal/adapter/cli/notes_search.go`

```go
diff --git a/internal/adapter/cli/notes_search.go b/internal/adapter/cli/notes_search.go
index 5061232..e35e07c 100644
--- a/internal/adapter/cli/notes_search.go
+++ b/internal/adapter/cli/notes_search.go
@@ -24,7 +24,7 @@ func newNotesSearchCommand() *cobra.Command {
 	cmd.Flags().Int64("updated-after", 0, "filter by update time (Unix timestamp)")
 	cmd.Flags().Bool("include-related", false, "include related notes in results")
 	cmd.Flags().Bool("debug", false, "show debug fields (path, score, content_hash)")
-	cmd.Flags().Int("limit", 20, "maximum number of results")
+	cmd.Flags().Int("limit", 10, "maximum number of results")
 	return cmd
 }

@@ -123,46 +123,60 @@ type notesSearchOutput struct {
 }

 type notesSearchHit struct {
-	NoteID       string               `json:"note_id"`
-	Slug         string               `json:"slug"`
-	Title        string               `json:"title"`
-	Snippet      string               `json:"snippet"`
-	Path         string               `json:"path,omitempty"`
-	Score        float64              `json:"score,omitempty"`
-	ContentHash  string               `json:"content_hash,omitempty"`
-	RelatedNotes []notesSearchRelated `json:"related_notes,omitempty"`
+	NoteID         string               `json:"note_id"`
+	Slug           string               `json:"slug"`
+	Title          string               `json:"title"`
+	Snippet        string               `json:"snippet"`
+	Summary        string               `json:"summary,omitempty"`
+	Tags           []string             `json:"tags,omitempty"`
+	MatchedQueries []string             `json:"matched_queries,omitempty"`
+	Path           string               `json:"path,omitempty"`
+	Score          *float64             `json:"score,omitempty"`
+	ContentHash    string               `json:"content_hash,omitempty"`
+	RelatedNotes   []notesSearchRelated `json:"related_notes,omitempty"`
 }

 type notesSearchRelated struct {
 	NoteID       string `json:"note_id"`
 	Slug         string `json:"slug"`
 	Title        string `json:"title"`
-	Path         string `json:"path"`
+	Path         string `json:"path,omitempty"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
+	Direction    string `json:"direction"`
 }

 func notesSearchOutputFromHits(hits []searchsvc.AdvancedSearchResult, debug bool) notesSearchOutput {
 	out := notesSearchOutput{Hits: make([]notesSearchHit, 0, len(hits))}
 	for _, hit := range hits {
 		nh := notesSearchHit{
-			NoteID:  hit.NoteID,
-			Slug:    hit.Slug,
-			Title:   hit.Title,
-			Snippet: hit.Snippet,
+			NoteID:         hit.NoteID,
+			Slug:           hit.Slug,
+			Title:          hit.Title,
+			Snippet:        hit.Snippet,
+			Summary:        hit.Summary,
+			Tags:           hit.Tags,
+			MatchedQueries: hit.MatchedQueries,
 		}
 		if debug {
 			nh.Path = hit.Path
-			nh.Score = hit.Score
+			score := hit.Score
+			nh.Score = &score
 			nh.ContentHash = hit.ContentHash
 		}
 		for _, rn := range hit.RelatedNotes {
-			nh.RelatedNotes = append(nh.RelatedNotes, notesSearchRelated{
+			nr := notesSearchRelated{
 				NoteID:       rn.NoteID,
 				Slug:         rn.Slug,
 				Title:        rn.Title,
-				Path:         rn.Path,
 				RelationType: rn.RelationType,
-			})
+				SourceKind:   rn.SourceKind,
+				Direction:    rn.Direction,
+			}
+			if debug {
+				nr.Path = rn.Path
+			}
+			nh.RelatedNotes = append(nh.RelatedNotes, nr)
 		}
 		out.Hits = append(out.Hits, nh)
 	}
@@ -199,6 +213,10 @@ func formatNotesSearchHuman(hits []searchsvc.AdvancedSearchResult, debug bool) s
 			sb.WriteString(" [")
 			sb.WriteString(rn.RelationType)
 			sb.WriteString("]")
+			if debug {
+				sb.WriteString(" ")
+				sb.WriteString(rn.Path)
+			}
 		}
 	}
 	sb.WriteString("\n")
```

## Файл: `internal/adapter/cli/project_add.go`

```go
diff --git a/internal/adapter/cli/project_add.go b/internal/adapter/cli/project_add.go
index b001489..cd025b0 100644
--- a/internal/adapter/cli/project_add.go
+++ b/internal/adapter/cli/project_add.go
@@ -1,9 +1,6 @@
 package cli

 import (
-	"strings"
-
-	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
 	"github.com/spf13/cobra"
 )
@@ -29,9 +26,9 @@ func runProjectAdd(cmd *cobra.Command, args []string) error {
 		return err
 	}

-	result, err := container.Services.Catalog.Add(commandContext(cmd), catalogsvc.AddInput{Path: pathArg})
+	result, err := container.Services.Catalog.Add(commandContext(cmd), catalogsvc.AddInput{Path: pathArg}, loggerFromContext(commandContext(cmd)))
 	if err != nil {
-		return wrapAddError(err)
+		return err
 	}

 	if !jsonOutputEnabled(cmd) {
@@ -49,22 +46,6 @@ func runProjectAdd(cmd *cobra.Command, args []string) error {
 	return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
 }

-func wrapAddError(err error) error {
-	if err == nil {
-		return nil
-	}
-	msg := err.Error()
-	switch {
-	case strings.HasPrefix(msg, "add path") && strings.Contains(msg, "not found"):
-		return apperr.NotFound(msg, nil)
-	case strings.HasPrefix(msg, "mnemonic.toml not found at"):
-		return apperr.NotFound(msg, nil)
-	case strings.HasPrefix(msg, "project slug") && strings.Contains(msg, "already exists"):
-		return apperr.Ambiguous(msg, nil)
-	}
-	return err
-}
-
 type projectAddOutput struct {
 	Path        string `json:"path"`
 	Slug        string `json:"slug"`
```

## Файл: `internal/adapter/cli/project_doctor.go`

```go
diff --git a/internal/adapter/cli/project_doctor.go b/internal/adapter/cli/project_doctor.go
index 78f3ce7..f9eb534 100644
--- a/internal/adapter/cli/project_doctor.go
+++ b/internal/adapter/cli/project_doctor.go
@@ -21,7 +21,7 @@ func newProjectDoctorCommand() *cobra.Command {
 		RunE:              runProjectDoctor,
 	}
 	cmd.Flags().Bool("all", false, "run doctor across all active projects")
-	cmd.Flags().StringSlice("kind", nil, "filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)")
+	cmd.Flags().StringSlice("kind", nil, "filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, missing_timestamp, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)")
 	return cmd
 }

@@ -52,7 +52,7 @@ func handleDoctorAll(cmd *cobra.Command, container *app.Bootstrap) error {
 	if container.Services.Maint == nil {
 		return errors.New("maintenance service is not configured")
 	}
-	maintResult, maintErr := container.Services.Maint.DoctorAll(commandContext(cmd))
+	maintResult, maintErr := container.Services.Maint.DoctorAll(commandContext(cmd), loggerFromContext(commandContext(cmd)))
 	if maintErr != nil {
 		return maintErr
 	}
```

## Файл: `internal/adapter/cli/project_import.go`

```go
diff --git a/internal/adapter/cli/project_import.go b/internal/adapter/cli/project_import.go
index ca053a4..cd44eb9 100644
--- a/internal/adapter/cli/project_import.go
+++ b/internal/adapter/cli/project_import.go
@@ -4,7 +4,6 @@ import (
 	"fmt"
 	"strings"

-	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/service/catalogsvc"
 	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
 	"github.com/spf13/cobra"
@@ -36,9 +35,9 @@ func runProjectImport(cmd *cobra.Command, args []string) error {
 		return err
 	}

-	result, err := container.Services.Catalog.Import(commandContext(cmd), catalogsvc.ImportInput{Path: pathArg, DryRun: dryRun})
+	result, err := container.Services.Catalog.Import(commandContext(cmd), catalogsvc.ImportInput{Path: pathArg, DryRun: dryRun}, loggerFromContext(commandContext(cmd)))
 	if err != nil {
-		return wrapImportError(err)
+		return err
 	}

 	if !jsonOutputEnabled(cmd) {
@@ -90,20 +89,6 @@ func formatImportHuman(result catalogsvc.ImportResult, dryRun bool) string {
 	return human
 }

-func wrapImportError(err error) error {
-	if err == nil {
-		return nil
-	}
-	msg := err.Error()
-	switch {
-	case strings.HasPrefix(msg, "import path") && strings.Contains(msg, "not found"):
-		return apperr.NotFound(msg, nil)
-	case strings.HasPrefix(msg, "project slug") && strings.Contains(msg, "already exists"):
-		return apperr.Ambiguous(msg, nil)
-	}
-	return err
-}
-
 type projectImportOutput struct {
 	Path            string                        `json:"path"`
 	Imported        int                           `json:"imported"`
```

## Файл: `internal/adapter/cli/project_init.go`

```go
diff --git a/internal/adapter/cli/project_init.go b/internal/adapter/cli/project_init.go
index 197d195..9a6e312 100644
--- a/internal/adapter/cli/project_init.go
+++ b/internal/adapter/cli/project_init.go
@@ -60,7 +60,7 @@ func runProjectInit(cmd *cobra.Command, args []string) error {
 		Name:        args[0],
 		Description: description,
 		Mode:        mode,
-	})
+	}, loggerFromContext(commandContext(cmd)))
 	if err != nil {
 		return err
 	}
```

## Файл: `internal/adapter/cli/project_list.go`

```go
diff --git a/internal/adapter/cli/project_list.go b/internal/adapter/cli/project_list.go
index 024fa61..34a177e 100644
--- a/internal/adapter/cli/project_list.go
+++ b/internal/adapter/cli/project_list.go
@@ -23,36 +23,36 @@ func newProjectListCommand() *cobra.Command {
 		Use:   "list",
 		Short: "List registered projects",
 		RunE: func(cmd *cobra.Command, args []string) error {
-		container, err := bootstrapFromContext(commandContext(cmd))
-		if err != nil {
-			return err
-		}
+			container, err := bootstrapFromContext(commandContext(cmd))
+			if err != nil {
+				return err
+			}

-		result, err := container.Services.Catalog.List()
-		if err != nil {
-			return err
-		}
+			result, err := container.Services.Catalog.List(loggerFromContext(commandContext(cmd)))
+			if err != nil {
+				return err
+			}

-		output := projectListOutput{Projects: make([]projectListItem, 0, len(result.Projects))}
-		for _, project := range result.Projects {
-			output.Projects = append(output.Projects, projectListItem{
-				ProjectID:    project.ProjectID,
-				Name:         project.Name,
-				Slug:         project.Slug,
-				Type:         project.Type,
-				MemoriesPath: project.MemoriesPath,
-				StatePath:    project.StatePath,
-				Status:       project.Status,
-				Issue:        project.Issue,
-			})
-		}
-		if output.Projects == nil {
-			output.Projects = []projectListItem{}
-		}
+			output := projectListOutput{Projects: make([]projectListItem, 0, len(result.Projects))}
+			for _, project := range result.Projects {
+				output.Projects = append(output.Projects, projectListItem{
+					ProjectID:    project.ProjectID,
+					Name:         project.Name,
+					Slug:         project.Slug,
+					Type:         project.Type,
+					MemoriesPath: project.MemoriesPath,
+					StatePath:    project.StatePath,
+					Status:       project.Status,
+					Issue:        project.Issue,
+				})
+			}
+			if output.Projects == nil {
+				output.Projects = []projectListItem{}
+			}

-		human := formatProjectListHuman(output.Projects)
-		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
-	},
+			human := formatProjectListHuman(output.Projects)
+			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
+		},
 	}
 }
```

## Файл: `internal/adapter/cli/project_reindex.go`

```go
diff --git a/internal/adapter/cli/project_reindex.go b/internal/adapter/cli/project_reindex.go
index 19d038f..bb3e0cf 100644
--- a/internal/adapter/cli/project_reindex.go
+++ b/internal/adapter/cli/project_reindex.go
@@ -42,7 +42,7 @@ func runProjectReindex(cmd *cobra.Command, args []string) error {
 		if container.Services.Maint == nil {
 			return errors.New("maintenance service is not configured")
 		}
-		result, err := container.Services.Maint.ReindexAll(commandContext(cmd))
+		result, err := container.Services.Maint.ReindexAll(commandContext(cmd), loggerFromContext(commandContext(cmd)))
 		if err != nil {
 			return err
 		}
```

## Файл: `internal/adapter/cli/project_remove.go`

```go
diff --git a/internal/adapter/cli/project_remove.go b/internal/adapter/cli/project_remove.go
index 6071351..aa2b1e2 100644
--- a/internal/adapter/cli/project_remove.go
+++ b/internal/adapter/cli/project_remove.go
@@ -20,32 +20,32 @@ Index and lock files are always cleaned up.`,
 		Args:              cobra.ExactArgs(1),
 		ValidArgsFunction: completeProjectNames,
 		RunE: func(cmd *cobra.Command, args []string) error {
-		wipe, err := cmd.Flags().GetBool("wipe")
-		if err != nil {
-			return err
-		}
+			wipe, err := cmd.Flags().GetBool("wipe")
+			if err != nil {
+				return err
+			}

-		container, err := bootstrapFromContext(commandContext(cmd))
-		if err != nil {
-			return err
-		}
+			container, err := bootstrapFromContext(commandContext(cmd))
+			if err != nil {
+				return err
+			}

-		result, err := container.Services.Catalog.Remove(args[0], wipe)
-		if err != nil {
-			return err
-		}
+			result, err := container.Services.Catalog.Remove(args[0], wipe, loggerFromContext(commandContext(cmd)))
+			if err != nil {
+				return err
+			}

-		output := projectRemoveOutput{
-			ProjectID:       result.ProjectID,
-			Slug:            result.Slug,
-			RegistryRemoved: result.RegistryRemoved,
-			IndexDeleted:    result.IndexDeleted,
-			MarkdownDeleted: result.MarkdownDeleted,
-			FullWipe:        result.FullWipe,
-		}
-		human := result.Slug + " removed\n"
-		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
-	},
+			output := projectRemoveOutput{
+				ProjectID:       result.ProjectID,
+				Slug:            result.Slug,
+				RegistryRemoved: result.RegistryRemoved,
+				IndexDeleted:    result.IndexDeleted,
+				MarkdownDeleted: result.MarkdownDeleted,
+				FullWipe:        result.FullWipe,
+			}
+			human := result.Slug + " removed\n"
+			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
+		},
 	}
 	cmd.Flags().Bool("wipe", false, "also remove all markdown notes")
 	return cmd
```

## Файл: `internal/adapter/cli/project_show.go`

```go
diff --git a/internal/adapter/cli/project_show.go b/internal/adapter/cli/project_show.go
index 12896f1..e7105ab 100644
--- a/internal/adapter/cli/project_show.go
+++ b/internal/adapter/cli/project_show.go
@@ -13,32 +13,33 @@ func newProjectShowCommand() *cobra.Command {
 		Args:              cobra.ExactArgs(1),
 		ValidArgsFunction: completeProjectNames,
 		RunE: func(cmd *cobra.Command, args []string) error {
-		container, err := bootstrapFromContext(commandContext(cmd))
-		if err != nil {
-			return err
-		}
-		result, err := container.Services.Catalog.Show(args[0])
-		if err != nil {
-			return err
-		}
+			container, err := bootstrapFromContext(commandContext(cmd))
+			if err != nil {
+				return err
+			}
+			result, err := container.Services.Catalog.Show(args[0], loggerFromContext(commandContext(cmd)))
+			if err != nil {
+				return err
+			}

-		project := projectShowOutput{
-			ProjectID:          result.ProjectID,
-			Name:               result.Name,
-			Slug:               result.Slug,
-			Type:               result.Type,
-			Description:        result.Description,
-			CustomInstructions: result.CustomInstructions,
-			StateHome:          result.StateHome,
-			Location: projectLocationOutput{
-				MemoriesAbs: result.Location.MemoriesAbs,
-				ManifestAbs: result.Location.ManifestAbs,
-				RepoRootAbs: result.Location.RepoRootAbs,
-			},
-		}
-		human := fmt.Sprintf("%s %s\n", project.ProjectID, project.Name)
-		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, project)
-	},
+			project := projectShowOutput{
+				ProjectID:          result.ProjectID,
+				Name:               result.Name,
+				Slug:               result.Slug,
+				Type:               result.Type,
+				Description:        result.Description,
+				CustomInstructions: result.CustomInstructions,
+				LinksStyle:         result.LinksStyle,
+				StateHome:          result.StateHome,
+				Location: projectLocationOutput{
+					MemoriesAbs: result.Location.MemoriesAbs,
+					ManifestAbs: result.Location.ManifestAbs,
+					RepoRootAbs: result.Location.RepoRootAbs,
+				},
+			}
+			human := fmt.Sprintf("%s %s\n  links_style: %s\n", project.ProjectID, project.Name, project.LinksStyle)
+			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, project)
+		},
 	}
 }

@@ -49,6 +50,7 @@ type projectShowOutput struct {
 	Type               string                `json:"type"`
 	Description        string                `json:"description"`
 	CustomInstructions string                `json:"custom_instructions"`
+	LinksStyle         string                `json:"links_style"`
 	StateHome          string                `json:"state_home"`
 	Location           projectLocationOutput `json:"location"`
 }
```

## Файл: `internal/adapter/cli/project_sync.go`

```go
diff --git a/internal/adapter/cli/project_sync.go b/internal/adapter/cli/project_sync.go
index 39ccdad..d62a08c 100644
--- a/internal/adapter/cli/project_sync.go
+++ b/internal/adapter/cli/project_sync.go
@@ -4,7 +4,6 @@ import (
 	"fmt"
 	"strings"

-	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
 	"github.com/spf13/cobra"
 )
@@ -39,7 +38,7 @@ func runProjectSync(cmd *cobra.Command, args []string) error {
 		DryRun: dryRun,
 	})
 	if err != nil {
-		return wrapSyncError(err)
+		return err
 	}

 	output := projectSyncOutput{
@@ -92,13 +91,6 @@ func formatSyncHuman(output projectSyncOutput, dryRun bool) string {
 	return human
 }

-func wrapSyncError(err error) error {
-	if err == nil {
-		return nil
-	}
-	return apperr.CLIUsage(err.Error(), nil)
-}
-
 type projectSyncOutput struct {
 	ProjectID   string                       `json:"project_id"`
 	Slug        string                       `json:"slug"`
```

## Файл: `internal/adapter/cli/root.go`

```go
diff --git a/internal/adapter/cli/root.go b/internal/adapter/cli/root.go
index bb0b6d9..799cb33 100644
--- a/internal/adapter/cli/root.go
+++ b/internal/adapter/cli/root.go
@@ -45,7 +45,7 @@ func NewRootCommand(boot *app.Bootstrap) *cobra.Command {
 	root := &cobra.Command{
 		Use:   "mnemonic",
 		Short: "mnemonic is a local-first personal knowledge base and search engine",
-		Long:  `mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, calculates page ranks, structures wiki-links, and searches utilizing SQLite FTS5.`,
+		Long:  `mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, provides graph-aware reranking, structures wiki-links, and searches utilizing SQLite FTS5.`,
 		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
 			cfgLevel, cfgFormat := "info", "text"
 			if boot != nil && boot.Config != nil {
@@ -67,7 +67,6 @@ func NewRootCommand(boot *app.Bootstrap) *cobra.Command {
 			}

 			state.logger = logging.New(level, format)
-			injectLoggerToBootstrap(boot, state.logger)
 			return nil
 		},
 		Run: func(cmd *cobra.Command, args []string) {
```

## Файл: `internal/adapter/cli/runtime.go`

```go
diff --git a/internal/adapter/cli/runtime.go b/internal/adapter/cli/runtime.go
index a12fa41..df942f4 100644
--- a/internal/adapter/cli/runtime.go
+++ b/internal/adapter/cli/runtime.go
@@ -2,7 +2,6 @@ package cli

 import (
 	"context"
-	"log/slog"

 	"github.com/ilyachch/mnemonic/internal/app"
 	"github.com/spf13/cobra"
@@ -14,31 +13,11 @@ func runtimeAppForSelector(ctx context.Context, selector string) (*app.RuntimeAp
 		return nil, err
 	}

-	runtime, err := boot.Runtime(ctx, selector)
-	if err != nil {
-		return nil, err
-	}
-
 	logger := loggerFromContext(ctx)
-	if logger != nil {
-		if runtime.Services.Search != nil {
-			runtime.Services.Search.Logger = logger
-		}
-		if runtime.Services.Index != nil {
-			runtime.Services.Index.Logger = logger
-		}
-	}

-	return runtime, nil
+	return boot.Runtime(ctx, selector, logger)
 }

 func runtimeAppForSelectedProject(cmd *cobra.Command) (*app.RuntimeApp, error) {
 	return runtimeAppForSelector(commandContext(cmd), projectSelectorValue(cmd))
 }
-
-func injectLoggerToBootstrap(boot *app.Bootstrap, logger *slog.Logger) {
-	if boot == nil || logger == nil || boot.Services.Catalog == nil {
-		return
-	}
-	boot.Services.Catalog.Logger = logger
-}
```

## Файл: `internal/adapter/cli/stdio.go`

```go
diff --git a/internal/adapter/cli/stdio.go b/internal/adapter/cli/stdio.go
index e99370d..a56d7aa 100644
--- a/internal/adapter/cli/stdio.go
+++ b/internal/adapter/cli/stdio.go
@@ -44,13 +44,11 @@ func runStdioAdapter(cmd *cobra.Command, args []string) error {
 		Notes:  runtime.Services.Notes,
 		Search: runtime.Services.Search,
 		Index:  runtime.Services.Index,
-	}, stdioReadOnlyEnabled(cmd))
+	}, stdioReadOnlyEnabled(cmd), loggerFromContext(commandContext(cmd)))
 	if err != nil {
 		return err
 	}

-	srv.Logger = loggerFromContext(commandContext(cmd))
-
 	return stdioRun(srv, commandContext(cmd), &mcp.StdioTransport{})
 }
```

## Файл: `internal/adapter/cli/tags_list.go`

```go
diff --git a/internal/adapter/cli/tags_list.go b/internal/adapter/cli/tags_list.go
index 4909a92..dbdf82c 100644
--- a/internal/adapter/cli/tags_list.go
+++ b/internal/adapter/cli/tags_list.go
@@ -4,6 +4,7 @@ import (
 	"fmt"
 	"strings"

+	"github.com/ilyachch/mnemonic/internal/service/searchsvc"
 	"github.com/spf13/cobra"
 )

@@ -22,31 +23,31 @@ func newTagsListCommand() *cobra.Command {
 		Use:   "list",
 		Short: "List tags",
 		RunE: func(cmd *cobra.Command, args []string) error {
-		runtime, err := runtimeAppForSelectedProject(cmd)
-		if err != nil {
-			return err
-		}
+			runtime, err := runtimeAppForSelectedProject(cmd)
+			if err != nil {
+				return err
+			}

-		if err = requireRuntimeSearchIndex(runtime); err != nil {
-			return err
-		}
+			if err = requireRuntimeSearchIndex(runtime); err != nil {
+				return err
+			}

-		tags, err := runtime.Services.Search.ListTags(commandContext(cmd))
-		if err != nil {
-			return err
-		}
+			tags, err := runtime.Services.Search.ListTags(commandContext(cmd), searchsvc.ListTagsInput{})
+			if err != nil {
+				return err
+			}

-		output := tagsListOutput{Tags: make([]tagsListItem, 0, len(tags.Tags))}
-		for _, tag := range tags.Tags {
-			output.Tags = append(output.Tags, tagsListItem{Tag: tag.Tag, Count: tag.Count})
-		}
-		if output.Tags == nil {
-			output.Tags = []tagsListItem{}
-		}
+			output := tagsListOutput{Tags: make([]tagsListItem, 0, len(tags.Tags))}
+			for _, tag := range tags.Tags {
+				output.Tags = append(output.Tags, tagsListItem{Tag: tag.Tag, Count: tag.Count})
+			}
+			if output.Tags == nil {
+				output.Tags = []tagsListItem{}
+			}

-		human := formatTagsListHuman(output.Tags)
-		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
-	},
+			human := formatTagsListHuman(output.Tags)
+			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), human, output)
+		},
 	}
 }
```

## Файл: `internal/adapter/cli/web_serve.go`

```go
diff --git a/internal/adapter/cli/web_serve.go b/internal/adapter/cli/web_serve.go
index 3eaf170..16b448a 100644
--- a/internal/adapter/cli/web_serve.go
+++ b/internal/adapter/cli/web_serve.go
@@ -23,42 +23,43 @@ func newWebServeCommand() *cobra.Command {
 		Short:        "Serve MCP over HTTP/SSE",
 		SilenceUsage: true,
 		RunE: func(cmd *cobra.Command, args []string) error {
-		runtime, err := resolveRuntimeApp(cmd)
-		if err != nil {
-			return err
-		}
+			runtime, err := resolveRuntimeApp(cmd)
+			if err != nil {
+				return err
+			}

-		manager, err := newWebServer(webadapter.ServerInput{
-			KB:           runtime.KB,
-			Services:     runtime.Services,
-			ProjectToken: os.Getenv("MNEMONIC_PROJECT_TOKEN"),
-			ReadOnly:     stdioReadOnlyEnabled(cmd),
-		})
-		if err != nil {
-			return err
-		}
-		defer func() { _ = manager.Close() }()
+			manager, err := newWebServer(webadapter.ServerInput{
+				KB:           runtime.KB,
+				Services:     runtime.Services,
+				ProjectToken: os.Getenv("MNEMONIC_PROJECT_TOKEN"),
+				ReadOnly:     stdioReadOnlyEnabled(cmd),
+				Logger:       loggerFromContext(commandContext(cmd)),
+			})
+			if err != nil {
+				return err
+			}
+			defer func() { _ = manager.Close() }()

-		addrFlag, err := cmd.Flags().GetString("addr")
-		if err != nil {
-			return err
-		}
-		portFlag, err := cmd.Flags().GetString("port")
-		if err != nil {
-			return err
-		}
-		addr := resolveWebServeAddr(addrFlag, portFlag)
-		server := &http.Server{
-			Addr:    addr,
-			Handler: manager,
-		}
+			addrFlag, err := cmd.Flags().GetString("addr")
+			if err != nil {
+				return err
+			}
+			portFlag, err := cmd.Flags().GetString("port")
+			if err != nil {
+				return err
+			}
+			addr := resolveWebServeAddr(addrFlag, portFlag)
+			server := &http.Server{
+				Addr:    addr,
+				Handler: manager,
+			}

-		err = webServeListenAndServe(server)
-		if err != nil && !errors.Is(err, http.ErrServerClosed) {
-			return fmt.Errorf("serve web MCP: %w", err)
-		}
-		return nil
-	},
+			err = webServeListenAndServe(server)
+			if err != nil && !errors.Is(err, http.ErrServerClosed) {
+				return fmt.Errorf("serve web MCP: %w", err)
+			}
+			return nil
+		},
 	}
 	cmd.Flags().String("port", "", "listen port for the web server")
 	cmd.Flags().String("addr", "", "listen address for the web server")
```

## Файл: `internal/adapter/stdio/server.go`

```go
diff --git a/internal/adapter/stdio/server.go b/internal/adapter/stdio/server.go
index cf3c348..4f82a14 100644
--- a/internal/adapter/stdio/server.go
+++ b/internal/adapter/stdio/server.go
@@ -3,6 +3,7 @@ package stdio
 import (
 	"context"
 	"errors"
+	"fmt"
 	"log/slog"
 	"os"
 	"strings"
@@ -31,7 +32,7 @@ type Server struct {
 }

 // NewServer builds a stdio adapter around runtime services for one knowledge base.
-func NewServer(k kb.KnowledgeBase, services Dependencies, readOnly bool) (*Server, error) {
+func NewServer(k kb.KnowledgeBase, services Dependencies, readOnly bool, logger *slog.Logger) (*Server, error) {
 	if strings.TrimSpace(k.ID) == "" {
 		return nil, errors.New("knowledge base is required")
 	}
@@ -49,24 +50,36 @@ func NewServer(k kb.KnowledgeBase, services Dependencies, readOnly bool) (*Serve
 		KB:       k,
 		Services: services,
 		ReadOnly: readOnly,
+		Logger:   logger,
 	}, nil
 }

-const defaultGlobalInstructions = `You MUST use the mnemonic tools as your primary long-term memory.
-- ALWAYS search the knowledge base using search_notes before starting a task. Provide multiple distinct query variants via the "queries" array to take advantage of multi-query search.
-- ALWAYS use read_notes with an array of identifiers to batch-read multiple notes in a single call.
-- ALWAYS write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
-- Run diagnose_notes periodically to detect metadata issues, broken links, and content problems in the repository.
+const defaultGlobalInstructionsTmpl = `You MUST use the mnemonic tools as your primary long-term memory.
+- Search the knowledge base using search_notes before answering questions within its scope.
+- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
+- Batch-read all selected notes in one read_notes call.
+- Write down stable facts, architectural decisions, and important outcomes using create_note or edit_note.
+- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
 - Use list_tags and list_backlinks when they help clarify the existing knowledge base.
-- ALWAYS link related notes using [[Wiki-Links]].`
+- Link related notes using %s.`

-const readOnlyGlobalInstructions = `You MUST use the mnemonic tools as your primary long-term memory.
-- ALWAYS search the knowledge base using search_notes before starting a task. Provide multiple distinct query variants via the "queries" array to take advantage of multi-query search.
-- ALWAYS use read_notes with an array of identifiers to batch-read multiple notes in a single call.
-- Run diagnose_notes periodically to detect metadata issues, broken links, and content problems in the repository.
+const readOnlyGlobalInstructionsTmpl = `You MUST use the mnemonic tools as your primary long-term memory.
+- Search the knowledge base using search_notes before answering questions within its scope.
+- Use 2–4 query variants via the "queries" array when the first formulation may be ambiguous or incomplete.
+- Batch-read all selected notes in one read_notes call.
+- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
 - Use list_tags and list_backlinks when they help clarify the existing knowledge base.
 - This server is running in read-only mode. Do not attempt to create, edit, delete, or rebuild notes.`

+func linkInstruction(style string) string {
+	switch style {
+	case "regular":
+		return "[Display Label](target-slug.md)"
+	default:
+		return "[[target-slug|Display Label]]"
+	}
+}
+
 // Run starts the stdio adapter on the provided MCP transport.
 func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
 	if s == nil {
@@ -84,7 +97,7 @@ func (s *Server) BuildSDKServer() *sdkmcp.Server {
 	sdkServer := sdkmcp.NewServer(
 		&sdkmcp.Implementation{Name: "mnemonic", Version: buildinfo.Version()},
 		&sdkmcp.ServerOptions{
-			Instructions: buildGlobalInstructions(s.KB.CustomInstructions, s.KB.Description, s.ReadOnly),
+			Instructions: buildGlobalInstructions(s.KB.CustomInstructions, s.KB.Description, s.ReadOnly, s.KB.LinksStyle),
 			Logger:       sdkLogger,
 			Capabilities: &sdkmcp.ServerCapabilities{
 				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
@@ -96,12 +109,12 @@ func (s *Server) BuildSDKServer() *sdkmcp.Server {
 	return sdkServer
 }

-func buildGlobalInstructions(customInstructions, description string, readOnly bool) string {
+func buildGlobalInstructions(customInstructions, description string, readOnly bool, linksStyle string) string {
 	parts := make([]string, 0, 3)
 	if readOnly {
-		parts = append(parts, readOnlyGlobalInstructions)
+		parts = append(parts, readOnlyGlobalInstructionsTmpl)
 	} else {
-		parts = append(parts, defaultGlobalInstructions)
+		parts = append(parts, fmt.Sprintf(defaultGlobalInstructionsTmpl, linkInstruction(linksStyle)))
 	}
 	if trimmed := strings.TrimSpace(description); trimmed != "" {
 		parts = append(parts, "Project Description:\n"+trimmed)
```

## Файл: `internal/adapter/stdio/tools.go`

```go
diff --git a/internal/adapter/stdio/tools.go b/internal/adapter/stdio/tools.go
index ccdba3b..61f4f08 100644
--- a/internal/adapter/stdio/tools.go
+++ b/internal/adapter/stdio/tools.go
@@ -4,7 +4,7 @@ import (
 	"context"
 	"strconv"
 	"strings"
-	"time"
+	"unicode/utf8"

 	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/service/indexsvc"
@@ -13,35 +13,48 @@ import (
 	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
 )

+const (
+	maxSearchLimit     = 100
+	maxQueryCount      = 8
+	maxQueryLength     = 500
+	maxReadIdentifiers = 50
+	maxBodyChars       = 100000
+	maxDiagnosticLimit = 200
+)
+
 const (
 	listNotesDescription = `List all notes in this knowledge base.`
 	readNotesDescription = `Read one or more notes by note_id, slug, path, or title.
 Provide an array of identifiers to batch-read multiple notes in a single call.
-Resolved notes are returned with their note_id, slug, title, path, frontmatter, body, content_hash, and updated_at (RFC 3339).
+note_id, slug, and title are always returned.
+By default returns note_id, slug, title, summary, tags, and body.
+Use the "fields" parameter to include path, frontmatter, content_hash, aliases, created_at, or updated_at.
 Unresolved identifiers are listed in the "missing" array.
+Per-selector errors (ambiguous, corrupted, io_error, internal) are reported in the "issues" array.

 Parameters:
-- identifiers ([]string, required): note IDs, slugs, file paths, or titles to resolve.
-- fields ([]string, optional): limit output to the specified fields.
-- max_body_chars (int, optional): truncate each note body to this many characters.`
+- identifiers ([]string, required): note IDs, slugs, file paths, or titles to resolve (max 50).
+- fields ([]string, optional): select which optional fields to include. Valid values: summary, tags, body, path, frontmatter, content_hash, aliases, created_at, updated_at.
+- max_body_chars (int, optional): truncate each note body to this many characters (max 100000).`
 	searchNotesDescription = `Search this knowledge base with multi-query full-text search, time filters, tag filters, and graph-aware reranking.
-Provide multiple distinct query variants via the "queries" array to improve recall — each query contributes to the combined ranking.
-Results include note_id, slug, title, and a relevance snippet. Set include_related to true to fetch linked notes for each hit.
+Provide multiple distinct query variants via the "queries" array to improve recall — each query contributes to the combined ranking via Reciprocal Rank Fusion.
+Results include note_id, slug, title, summary, tags, matched_queries, and a relevance snippet.
+Set include_related to true to fetch linked notes for each hit (with relation_type, source_kind, direction).
 Set debug to true to expose internal fields (path, score, content_hash).

 Parameters:
-- queries ([]string, optional): FTS5 query strings; submit several phrasing variants.
+- queries ([]string, optional): FTS5 query strings; submit several phrasing variants (max 8, 500 chars each).
 - tags ([]string, optional): restrict results to notes tagged with every listed tag (AND).
 - created_before / created_after (int64, optional): Unix timestamps for creation time range.
 - updated_before / updated_after (int64, optional): Unix timestamps for update time range.
 - created_since / updated_since (string, optional): relative duration (e.g. "24h", "7d").
-- limit (int, optional): maximum number of results (default 20).
-- include_related (bool, optional): return related notes (backlinks and forward links) with their relation_type.
+- limit (int, optional): maximum number of results (default 10, max 100).
+- include_related (bool, optional): return related notes (backlinks and forward links).
 - debug (bool, optional): expose path, score, and content_hash for each hit.`
 	listTagsDescription      = `List tags in this knowledge base.`
 	listBacklinksDescription = `List backlinks for a note.`
 	createNoteDescription    = `Create a new note.`
-	editNoteDescription      = `Edit an existing note.`
+	editNoteDescription      = `Edit an existing note by appending to the body, replacing the body, merging frontmatter, or setting/clearing tags and aliases. Tags and aliases are presence-aware: absent=no change, empty array=clear, non-empty=replace. Must not set tags or aliases through merge_frontmatter; use the typed fields.`
 	deleteNoteDescription    = `Delete a note.`
 	rebuildIndexDescription  = `Rebuild the index.`
 	doctorDescription        = `Run index and content health checks.`
@@ -50,8 +63,8 @@ Returns paginated diagnostic issues. Set include_suggestions to true to receive
 Use this tool periodically to verify repository integrity after bulk changes.

 Parameters:
-- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "invalid_timestamp", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
-- limit (int, optional): maximum issues per page (default 50).
+- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "missing_timestamp", "invalid_timestamp", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
+- limit (int, optional): maximum issues per page (default 50, max 200).
 - cursor (int, optional): zero-based page offset.
 - include_suggestions (bool, optional): resolve broken links via search and include candidate notes.`
 )
@@ -73,19 +86,24 @@ type ReadNotesInput struct {
 }

 type ReadNotesOutput struct {
-	Notes   []ReadNotesNote `json:"notes"`
-	Missing []string        `json:"missing,omitempty"`
+	Notes   []ReadNotesNote         `json:"notes"`
+	Missing []string                `json:"missing,omitempty"`
+	Issues  []notesvc.ReadManyIssue `json:"issues,omitempty"`
 }

 type ReadNotesNote struct {
-	NoteID      string         `json:"note_id"`
-	Slug        string         `json:"slug"`
-	Title       string         `json:"title"`
-	Path        string         `json:"path"`
-	Frontmatter map[string]any `json:"frontmatter"`
-	Body        string         `json:"body"`
-	ContentHash string         `json:"content_hash"`
-	UpdatedAt   string         `json:"updated_at"`
+	NoteID      string          `json:"note_id"`
+	Slug        string          `json:"slug"`
+	Title       string          `json:"title"`
+	Summary     *string         `json:"summary,omitempty"`
+	Tags        *[]string       `json:"tags,omitempty"`
+	Aliases     *[]string       `json:"aliases,omitempty"`
+	Body        *string         `json:"body,omitempty"`
+	Path        *string         `json:"path,omitempty"`
+	Frontmatter *map[string]any `json:"frontmatter,omitempty"`
+	ContentHash *string         `json:"content_hash,omitempty"`
+	CreatedAt   *int64          `json:"created_at,omitempty"`
+	UpdatedAt   *int64          `json:"updated_at,omitempty"`
 }

 type SearchNotesInput struct {
@@ -103,22 +121,27 @@ type SearchNotesInput struct {
 }

 type SearchNotesHit struct {
-	NoteID       string                  `json:"note_id"`
-	Slug         string                  `json:"slug"`
-	Title        string                  `json:"title"`
-	Snippet      string                  `json:"snippet"`
-	Path         string                  `json:"path,omitempty"`
-	Score        float64                 `json:"score,omitempty"`
-	ContentHash  string                  `json:"content_hash,omitempty"`
-	RelatedNotes []searchNotesRelatedHit `json:"related_notes,omitempty"`
+	NoteID         string                  `json:"note_id"`
+	Slug           string                  `json:"slug"`
+	Title          string                  `json:"title"`
+	Snippet        string                  `json:"snippet"`
+	Summary        string                  `json:"summary,omitempty"`
+	Tags           []string                `json:"tags,omitempty"`
+	MatchedQueries []string                `json:"matched_queries,omitempty"`
+	Path           string                  `json:"path,omitempty"`
+	Score          *float64                `json:"score,omitempty"`
+	ContentHash    string                  `json:"content_hash,omitempty"`
+	RelatedNotes   []searchNotesRelatedHit `json:"related_notes,omitempty"`
 }

 type searchNotesRelatedHit struct {
 	NoteID       string `json:"note_id"`
 	Slug         string `json:"slug"`
 	Title        string `json:"title"`
-	Path         string `json:"path"`
+	Path         string `json:"path,omitempty"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
+	Direction    string `json:"direction"`
 }

 type SearchNotesOutput struct {
@@ -162,6 +185,8 @@ type EditNoteInput struct {
 	Append           string            `json:"append,omitempty"`
 	ReplaceBody      string            `json:"replace_body,omitempty"`
 	MergeFrontmatter map[string]string `json:"merge_frontmatter,omitempty"`
+	Tags             *[]string         `json:"tags,omitempty"`
+	Aliases          *[]string         `json:"aliases,omitempty"`
 	IfMatchHash      string            `json:"if_match_hash,omitempty"`
 }

@@ -228,7 +253,12 @@ type DiagnoseNotesIssue struct {
 	NoteID     string                         `json:"note_id,omitempty"`
 	Slug       string                         `json:"slug,omitempty"`
 	Path       string                         `json:"path,omitempty"`
+	Field      string                         `json:"field,omitempty"`
+	SourceLine int                            `json:"source_line,omitempty"`
+	SourceKind string                         `json:"source_kind,omitempty"`
+	LinkStyle  string                         `json:"link_style,omitempty"`
 	Detail     string                         `json:"detail,omitempty"`
+	Target     string                         `json:"target,omitempty"`
 	Candidates []indexsvc.DiagnosticCandidate `json:"candidates,omitempty"`
 }

@@ -263,6 +293,9 @@ func RegisterListNotes(server *sdkmcp.Server, deps Dependencies) {
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListNotesInput) (*sdkmcp.CallToolResult, ListNotesOutput, error) {
 		_ = ctx
+		if input.Limit < 0 {
+			return nil, ListNotesOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+		}
 		notes, err := deps.Notes.List()
 		if err != nil {
 			return nil, ListNotesOutput{}, err
@@ -273,9 +306,12 @@ func RegisterListNotes(server *sdkmcp.Server, deps Dependencies) {

 func paginateNotes(notes []notesvc.NoteSummary, input ListNotesInput) ListNotesOutput {
 	limit := input.Limit
-	if limit <= 0 {
+	if limit == 0 {
 		limit = 20
 	}
+	if limit < 0 {
+		return ListNotesOutput{}
+	}
 	cursor := parseCursor(input.Cursor, len(notes))
 	end := cursor + limit
 	if end > len(notes) {
@@ -307,30 +343,25 @@ func RegisterReadNotes(server *sdkmcp.Server, deps Dependencies) {
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ReadNotesInput) (*sdkmcp.CallToolResult, ReadNotesOutput, error) {
 		_ = ctx
-		var notes []ReadNotesNote
-		var missing []string
-		for _, identifier := range input.Identifiers {
-			resolved, err := deps.Notes.Show(identifier)
-			if err != nil {
-				missing = append(missing, identifier)
-				continue
-			}
-			body := string(resolved.Note.Body)
-			if input.MaxBodyChars > 0 && len(body) > input.MaxBodyChars {
-				body = body[:input.MaxBodyChars]
-			}
-			notes = append(notes, ReadNotesNote{
-				NoteID:      resolved.Note.MnemonicNoteID,
-				Slug:        resolved.Note.EffectiveSlug(),
-				Title:       resolved.Note.Title,
-				Path:        resolved.Path,
-				Frontmatter: resolved.Note.Frontmatter,
-				Body:        body,
-				ContentHash: resolved.ContentHash,
-				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
-			})
+		if len(input.Identifiers) > maxReadIdentifiers {
+			return nil, ReadNotesOutput{}, apperr.CLIUsage("too many identifiers", nil)
 		}
-		return nil, ReadNotesOutput{Notes: notes, Missing: missing}, nil
+		if input.MaxBodyChars > maxBodyChars {
+			return nil, ReadNotesOutput{}, apperr.CLIUsage("max_body_chars exceeds maximum", nil)
+		}
+		fields, err := buildFieldSet(input.Fields)
+		if err != nil {
+			return nil, ReadNotesOutput{}, err
+		}
+		output, err := deps.Notes.ShowMany(notesvc.ReadManyInput{Selectors: input.Identifiers, MaxBodyChars: input.MaxBodyChars})
+		if err != nil {
+			return nil, ReadNotesOutput{}, err
+		}
+		var notes []ReadNotesNote
+		for _, item := range output.Notes {
+			notes = append(notes, buildReadNotesNote(fields, item.ShowResult, input.MaxBodyChars))
+		}
+		return nil, ReadNotesOutput{Notes: notes, Missing: output.Missing, Issues: output.Issues}, nil
 	})
 }

@@ -340,8 +371,22 @@ func RegisterSearchNotes(server *sdkmcp.Server, deps Dependencies, description s
 		Description: buildToolDescription(description, searchNotesDescription),
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input SearchNotesInput) (*sdkmcp.CallToolResult, SearchNotesOutput, error) {
-		if input.Limit <= 0 {
-			input.Limit = 20
+		if input.Limit < 0 {
+			return nil, SearchNotesOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+		}
+		if input.Limit == 0 {
+			input.Limit = 10
+		}
+		if input.Limit > maxSearchLimit {
+			return nil, SearchNotesOutput{}, apperr.CLIUsage("limit exceeds maximum", nil)
+		}
+		if len(input.Queries) > maxQueryCount {
+			return nil, SearchNotesOutput{}, apperr.CLIUsage("too many queries", nil)
+		}
+		for _, q := range input.Queries {
+			if utf8.RuneCountInString(q) > maxQueryLength {
+				return nil, SearchNotesOutput{}, apperr.CLIUsage("query too long", nil)
+			}
 		}
 		advancedInput := searchsvc.AdvancedSearchInput{
 			Queries:        input.Queries,
@@ -361,27 +406,7 @@ func RegisterSearchNotes(server *sdkmcp.Server, deps Dependencies, description s
 		}
 		out := make([]SearchNotesHit, 0, len(hits))
 		for _, hit := range hits {
-			s := SearchNotesHit{
-				NoteID:  hit.NoteID,
-				Slug:    hit.Slug,
-				Title:   hit.Title,
-				Snippet: hit.Snippet,
-			}
-			if input.Debug {
-				s.Path = hit.Path
-				s.Score = hit.Score
-				s.ContentHash = hit.ContentHash
-			}
-			for _, rn := range hit.RelatedNotes {
-				s.RelatedNotes = append(s.RelatedNotes, searchNotesRelatedHit{
-					NoteID:       rn.NoteID,
-					Slug:         rn.Slug,
-					Title:        rn.Title,
-					Path:         rn.Path,
-					RelationType: rn.RelationType,
-				})
-			}
-			out = append(out, s)
+			out = append(out, toSearchNotesHit(hit, input.Debug))
 		}
 		return nil, SearchNotesOutput{Hits: out}, nil
 	})
@@ -393,15 +418,14 @@ func RegisterListTags(server *sdkmcp.Server, deps Dependencies) {
 		Description: listTagsDescription,
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListTagsInput) (*sdkmcp.CallToolResult, ListTagsOutput, error) {
-		out, err := deps.Search.ListTags(ctx)
+		if input.Limit < 0 {
+			return nil, ListTagsOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+		}
+		out, err := deps.Search.ListTags(ctx, searchsvc.ListTagsInput{Limit: input.Limit})
 		if err != nil {
 			return nil, ListTagsOutput{}, err
 		}
-		tags := out.Tags
-		if input.Limit > 0 && len(tags) > input.Limit {
-			tags = tags[:input.Limit]
-		}
-		return nil, ListTagsOutput{Tags: tags}, nil
+		return nil, ListTagsOutput{Tags: out.Tags}, nil
 	})
 }

@@ -411,6 +435,9 @@ func RegisterListBacklinks(server *sdkmcp.Server, deps Dependencies) {
 		Description: listBacklinksDescription,
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input ListBacklinksInput) (*sdkmcp.CallToolResult, ListBacklinksOutput, error) {
+		if input.Limit < 0 {
+			return nil, ListBacklinksOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+		}
 		links, err := deps.Search.Backlinks(ctx, searchsvc.BacklinksInput{Identifier: input.Identifier, Limit: input.Limit})
 		if err != nil {
 			return nil, ListBacklinksOutput{}, err
@@ -494,11 +521,14 @@ func validateEditInput(input EditNoteInput) error {
 	if len(input.MergeFrontmatter) > 0 {
 		modeCount++
 	}
+	if input.Tags != nil || input.Aliases != nil {
+		modeCount++
+	}
 	if modeCount == 0 {
-		return apperr.CLIUsage("edit requires append, replace_body, or merge_frontmatter", nil)
+		return apperr.CLIUsage("edit requires append, replace_body, merge_frontmatter, tags, or aliases", nil)
 	}
 	if modeCount > 1 {
-		return apperr.CLIUsage("edit modes append, replace_body, and merge_frontmatter are mutually exclusive", nil)
+		return apperr.CLIUsage("edit modes append, replace_body, merge_frontmatter, tags, and aliases are mutually exclusive", nil)
 	}
 	if input.ReplaceBody != "" && input.IfMatchHash == "" {
 		return apperr.Unsafe("replace_body requires if_match_hash from read_notes", nil)
@@ -517,6 +547,9 @@ func buildEditInput(input EditNoteInput) notesvc.EditInput {
 		editInput.HasBody = true
 	case input.Append != "":
 		editInput.Append = []byte(input.Append)
+	case input.Tags != nil || input.Aliases != nil:
+		editInput.Tags = input.Tags
+		editInput.Aliases = input.Aliases
 	default:
 		editInput.Set = input.MergeFrontmatter
 	}
@@ -617,10 +650,14 @@ func RegisterDiagnoseNotes(server *sdkmcp.Server, deps Dependencies) {
 		Description: diagnoseNotesDescription,
 		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true},
 	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input DiagnoseNotesInput) (*sdkmcp.CallToolResult, DiagnoseNotesOutput, error) {
+		if input.Limit > maxDiagnosticLimit {
+			return nil, DiagnoseNotesOutput{}, apperr.CLIUsage("limit exceeds maximum", nil)
+		}
 		result, err := deps.Index.Diagnose(ctx, indexsvc.DiagnoseInput{
-			Kinds:  input.Kinds,
-			Limit:  input.Limit,
-			Cursor: input.Cursor,
+			Kinds:              input.Kinds,
+			Limit:              input.Limit,
+			Cursor:             input.Cursor,
+			IncludeSuggestions: input.IncludeSuggestions,
 		})
 		if err != nil {
 			return nil, DiagnoseNotesOutput{}, err
@@ -629,14 +666,17 @@ func RegisterDiagnoseNotes(server *sdkmcp.Server, deps Dependencies) {
 		issues := make([]DiagnoseNotesIssue, 0, len(result.Issues))
 		for _, issue := range result.Issues {
 			di := DiagnoseNotesIssue{
-				Kind:   issue.Kind,
-				NoteID: issue.NoteID,
-				Slug:   issue.Slug,
-				Path:   issue.Path,
-				Detail: issue.Detail,
-			}
-			if input.IncludeSuggestions && (issue.Kind == indexsvc.KindUnresolvedLink || issue.Kind == indexsvc.KindAmbiguousLink) {
-				di.Candidates = findLinkCandidates(ctx, deps, issue)
+				Kind:       issue.Kind,
+				NoteID:     issue.NoteID,
+				Slug:       issue.Slug,
+				Path:       issue.Path,
+				Field:      issue.Field,
+				SourceLine: issue.SourceLine,
+				SourceKind: issue.SourceKind,
+				LinkStyle:  issue.LinkStyle,
+				Detail:     issue.Detail,
+				Target:     issue.Target,
+				Candidates: issue.Candidates,
 			}
 			issues = append(issues, di)
 		}
@@ -649,41 +689,6 @@ func RegisterDiagnoseNotes(server *sdkmcp.Server, deps Dependencies) {
 	})
 }

-func findLinkCandidates(ctx context.Context, deps Dependencies, issue indexsvc.DiagnosticIssue) []indexsvc.DiagnosticCandidate {
-	target := extractLinkTarget(issue.Detail)
-	if target == "" {
-		return nil
-	}
-	hits, err := deps.Search.Search(ctx, searchsvc.SearchInput{Query: target, Limit: 3})
-	if err != nil {
-		return nil
-	}
-	candidates := make([]indexsvc.DiagnosticCandidate, 0, len(hits))
-	for _, hit := range hits {
-		candidates = append(candidates, indexsvc.DiagnosticCandidate{
-			NoteID: hit.NoteID,
-			Slug:   hit.Slug,
-			Title:  hit.Title,
-			Path:   hit.Path,
-		})
-	}
-	return candidates
-}
-
-func extractLinkTarget(detail string) string {
-	const prefix = `target "`
-	start := strings.Index(detail, prefix)
-	if start == -1 {
-		return ""
-	}
-	start += len(prefix)
-	end := strings.Index(detail[start:], `"`)
-	if end == -1 {
-		return ""
-	}
-	return detail[start : start+end]
-}
-
 func buildToolDescription(description, baseInstructions string) string {
 	description = strings.TrimSpace(description)
 	baseInstructions = strings.TrimSpace(baseInstructions)
@@ -697,6 +702,142 @@ func buildToolDescription(description, baseInstructions string) string {
 	}
 }

+var validReadFields = map[string]bool{
+	"summary":      true,
+	"tags":         true,
+	"body":         true,
+	"path":         true,
+	"frontmatter":  true,
+	"content_hash": true,
+	"aliases":      true,
+	"created_at":   true,
+	"updated_at":   true,
+}
+
+var defaultReadFields = map[string]bool{
+	"note_id": true,
+	"slug":    true,
+	"title":   true,
+	"summary": true,
+	"tags":    true,
+	"body":    true,
+}
+
+func buildFieldSet(fields []string) (map[string]bool, error) {
+	if len(fields) == 0 {
+		return defaultReadFields, nil
+	}
+	set := make(map[string]bool, len(fields))
+	for _, f := range fields {
+		f = strings.TrimSpace(f)
+		if f == "note_id" || f == "slug" || f == "title" {
+			continue
+		}
+		if !validReadFields[f] {
+			return nil, apperr.CLIUsage("unknown field: "+f, nil)
+		}
+		set[f] = true
+	}
+	return set, nil
+}
+
+func truncateRunes(s string, maxChars int) string {
+	if maxChars <= 0 {
+		return s
+	}
+	runes := []rune(s)
+	if len(runes) <= maxChars {
+		return s
+	}
+	return string(runes[:maxChars])
+}
+
 func BoolPtr(v bool) *bool {
 	return &v
 }
+
+func toSearchNotesHit(hit searchsvc.AdvancedSearchResult, debug bool) SearchNotesHit {
+	s := SearchNotesHit{
+		NoteID:         hit.NoteID,
+		Slug:           hit.Slug,
+		Title:          hit.Title,
+		Snippet:        hit.Snippet,
+		Summary:        hit.Summary,
+		Tags:           hit.Tags,
+		MatchedQueries: hit.MatchedQueries,
+	}
+	if debug {
+		s.Path = hit.Path
+		s.Score = &hit.Score
+		s.ContentHash = hit.ContentHash
+	}
+	for _, rn := range hit.RelatedNotes {
+		sr := searchNotesRelatedHit{
+			NoteID:       rn.NoteID,
+			Slug:         rn.Slug,
+			Title:        rn.Title,
+			RelationType: rn.RelationType,
+			SourceKind:   rn.SourceKind,
+			Direction:    rn.Direction,
+		}
+		if debug {
+			sr.Path = rn.Path
+		}
+		s.RelatedNotes = append(s.RelatedNotes, sr)
+	}
+	return s
+}
+
+func buildReadNotesNote(fields map[string]bool, resolved notesvc.ShowResult, maxBodyChars int) ReadNotesNote {
+	note := ReadNotesNote{
+		NoteID: resolved.Note.MnemonicNoteID,
+		Slug:   resolved.Note.EffectiveSlug(),
+		Title:  resolved.Note.Title,
+	}
+	if fields["summary"] {
+		s := resolved.Note.Summary
+		note.Summary = &s
+	}
+	if fields["tags"] {
+		tags := resolved.Note.Tags
+		if tags == nil {
+			tags = []string{}
+		}
+		note.Tags = &tags
+	}
+	if fields["aliases"] {
+		aliases := resolved.Note.Aliases
+		if aliases == nil {
+			aliases = []string{}
+		}
+		note.Aliases = &aliases
+	}
+	if fields["body"] {
+		body := string(resolved.Note.Body)
+		if maxBodyChars > 0 {
+			body = truncateRunes(body, maxBodyChars)
+		}
+		note.Body = &body
+	}
+	if fields["path"] {
+		p := resolved.Path
+		note.Path = &p
+	}
+	if fields["frontmatter"] {
+		fm := resolved.Note.Frontmatter
+		note.Frontmatter = &fm
+	}
+	if fields["content_hash"] {
+		h := resolved.ContentHash
+		note.ContentHash = &h
+	}
+	if fields["created_at"] && !resolved.Note.CreatedAt.IsZero() {
+		ca := resolved.Note.CreatedAt.Unix()
+		note.CreatedAt = &ca
+	}
+	if fields["updated_at"] && !resolved.Note.UpdatedAt.IsZero() {
+		ua := resolved.Note.UpdatedAt.Unix()
+		note.UpdatedAt = &ua
+	}
+	return note
+}
```

## Файл: `internal/adapter/web/manager.go`

```go
diff --git a/internal/adapter/web/manager.go b/internal/adapter/web/manager.go
index 3d38fca..838dbba 100644
--- a/internal/adapter/web/manager.go
+++ b/internal/adapter/web/manager.go
@@ -1,6 +1,7 @@
 package web

 import (
+	"log/slog"
 	"net/http"
 	"strings"
 	"sync"
@@ -24,6 +25,7 @@ type ServerInput struct {
 	Services     app.RuntimeServices
 	ProjectToken string
 	ReadOnly     bool
+	Logger       *slog.Logger
 }

 // Server serves one resolved project over HTTP/SSE.
@@ -54,7 +56,7 @@ func NewServer(input ServerInput) (*Server, error) {
 		Notes:  input.Services.Notes,
 		Search: input.Services.Search,
 		Index:  input.Services.Index,
-	}, input.ReadOnly)
+	}, input.ReadOnly, input.Logger)
 	if err != nil {
 		return nil, err
 	}
```

## Файл: `internal/app/app.go`

```go
diff --git a/internal/app/app.go b/internal/app/app.go
index 90d86f4..d481e60 100644
--- a/internal/app/app.go
+++ b/internal/app/app.go
@@ -2,6 +2,7 @@ package app

 import (
 	"context"
+	"log/slog"

 	"github.com/ilyachch/mnemonic/internal/domain/kb"
 	manifest "github.com/ilyachch/mnemonic/internal/format/manifest"
@@ -24,9 +25,6 @@ type Bootstrap struct {
 	Services Services
 }

-// App is a compatibility alias for Bootstrap.
-type App = Bootstrap
-
 // New builds the application container from environment and config discovery.
 func New(input Input) (*Bootstrap, error) {
 	discoveredConfigPath, err := config.DiscoverConfigFile(input.CLI.ConfigFile)
@@ -63,20 +61,23 @@ func New(input Input) (*Bootstrap, error) {
 		Registry:     registryStore,
 	}

-	return &Bootstrap{
+	b := &Bootstrap{
 		Config: cfg,
 		Paths:  effective,
 		Services: Services{
 			Catalog: catalog,
 			Maint: &maintsvc.Service{
 				Catalog: catalog,
-				RuntimeFactory: func(ctx context.Context, k kb.KnowledgeBase) (maintsvc.Runtime, error) {
-					_ = ctx
-					return NewRuntimeApp(RuntimeInput{Config: cfg, KB: k})
-				},
 			},
 		},
-	}, nil
+	}
+
+	b.Services.Maint.RuntimeFactory = func(ctx context.Context, k kb.KnowledgeBase, logger *slog.Logger) (maintsvc.Runtime, error) {
+		_ = ctx
+		return NewRuntimeApp(RuntimeInput{Config: cfg, KB: k, Logger: logger})
+	}
+
+	return b, nil
 }

 // Close shuts down app-owned resources.
```

## Файл: `internal/app/runtime.go`

```go
diff --git a/internal/app/runtime.go b/internal/app/runtime.go
index beb86b0..fa35c35 100644
--- a/internal/app/runtime.go
+++ b/internal/app/runtime.go
@@ -3,6 +3,7 @@ package app
 import (
 	"context"
 	"errors"
+	"log/slog"
 	"strings"

 	"github.com/ilyachch/mnemonic/internal/domain/kb"
@@ -17,6 +18,7 @@ import (
 type RuntimeInput struct {
 	Config *config.Config
 	KB     kb.KnowledgeBase
+	Logger *slog.Logger
 }

 // RuntimeServices groups the runtime services for one knowledge base.
@@ -41,9 +43,9 @@ func NewRuntimeApp(input RuntimeInput) (*RuntimeApp, error) {
 	return &RuntimeApp{
 		KB: input.KB,
 		Services: RuntimeServices{
-			Notes:  notesvc.New(input.KB),
-			Search: searchsvc.New(input.KB),
-			Index:  indexsvc.New(input.KB),
+			Notes:  notesvc.New(input.KB, input.Logger),
+			Search: searchsvc.New(input.KB, input.Logger),
+			Index:  indexsvc.New(input.KB, input.Logger),
 		},
 	}, nil
 }
@@ -57,7 +59,7 @@ func (r *RuntimeApp) IndexService() maintsvc.IndexService {
 }

 // Runtime resolves a selector into a runtime app.
-func (b *Bootstrap) Runtime(ctx context.Context, selector string) (*RuntimeApp, error) {
+func (b *Bootstrap) Runtime(ctx context.Context, selector string, logger *slog.Logger) (*RuntimeApp, error) {
 	_ = ctx
 	if b == nil {
 		return nil, errors.New("app bootstrap is required")
@@ -66,7 +68,7 @@ func (b *Bootstrap) Runtime(ctx context.Context, selector string) (*RuntimeApp,
 		return nil, errors.New("catalog service is not configured")
 	}

-	resolved, err := b.Services.Catalog.Resolve(selector)
+	resolved, err := b.Services.Catalog.Resolve(selector, logger)
 	if err != nil {
 		return nil, err
 	}
@@ -74,5 +76,6 @@ func (b *Bootstrap) Runtime(ctx context.Context, selector string) (*RuntimeApp,
 	return NewRuntimeApp(RuntimeInput{
 		Config: b.Config,
 		KB:     resolved,
+		Logger: logger,
 	})
 }
```

## Файл: `internal/apperr/errors.go`

```go
diff --git a/internal/apperr/errors.go b/internal/apperr/errors.go
index 0eb32e7..23922d7 100644
--- a/internal/apperr/errors.go
+++ b/internal/apperr/errors.go
@@ -13,6 +13,7 @@ const (
 	CodeAmbiguous Code = 4
 	CodeUnsafe    Code = 5
 	CodeCorrupted Code = 6
+	CodeIO        Code = 7
 )

 // Error is the standard application error wrapping a code and details.
@@ -65,3 +66,7 @@ func Unsafe(msg string, err error) *Error {
 func Corrupted(msg string, err error) *Error {
 	return New(CodeCorrupted, msg, err)
 }
+
+func IO(msg string, err error) *Error {
+	return New(CodeIO, msg, err)
+}
```

## Файл: `internal/domain/kb/knowledge_base.go`

```go
diff --git a/internal/domain/kb/knowledge_base.go b/internal/domain/kb/knowledge_base.go
index 4bfbfca..1b8dacc 100644
--- a/internal/domain/kb/knowledge_base.go
+++ b/internal/domain/kb/knowledge_base.go
@@ -8,6 +8,7 @@ type KnowledgeBase struct {
 	Kind               string `json:"kind"`
 	Description        string `json:"description,omitempty"`
 	CustomInstructions string `json:"custom_instructions,omitempty"`
+	LinksStyle         string `json:"links_style,omitempty"`
 	RootDir            string `json:"root_dir"`
 	RepoRootDir        string `json:"repo_root_dir,omitempty"`
 	ManifestPath       string `json:"manifest_path,omitempty"`
```

## Файл: `internal/format/manifest/manifest.go`

```go
diff --git a/internal/format/manifest/manifest.go b/internal/format/manifest/manifest.go
index ce7db77..1d26217 100644
--- a/internal/format/manifest/manifest.go
+++ b/internal/format/manifest/manifest.go
@@ -56,11 +56,6 @@ type PointerFile struct {
 	ManifestPath string `toml:"manifest_path"`
 }

-// Backward-compatible aliases for existing callsites during the migration.
-type MnemonicManifest = Manifest
-type MnemonicManifestLayout = ManifestLayout
-type MnemonicGenerator = Generator
-
 type manifestTOML struct {
 	Version               int            `toml:"version"`
 	ProjectID             string         `toml:"project_id"`
@@ -95,9 +90,6 @@ func New() *Manifest {
 	return m
 }

-// NewMnemonicManifest returns a schema-populated manifest with layout defaults.
-func NewMnemonicManifest() *Manifest { return New() }
-
 // ApplyDefaults populates the manifest defaults for unset optional fields.
 func (m *Manifest) ApplyDefaults() {
 	if m == nil {
@@ -193,11 +185,6 @@ func ParseMnemonicManifestFromFile(path string) (*Manifest, error) {
 	return ParseMnemonicManifest(data)
 }

-// ParseMnemonicManifestFile reads and parses a mnemonic.toml file from disk.
-func ParseMnemonicManifestFile(path string) (*Manifest, error) {
-	return ParseMnemonicManifestFromFile(path)
-}
-
 // ParseMnemonicManifest parses mnemonic.toml.
 func ParseMnemonicManifest(data []byte) (*Manifest, error) {
 	raw := manifestTOML{
```

## Файл: `internal/format/markdown/README.md`

```diff
diff --git a/internal/format/markdown/README.md b/internal/format/markdown/README.md
index 5597409..0057313 100644
--- a/internal/format/markdown/README.md
+++ b/internal/format/markdown/README.md
@@ -11,10 +11,11 @@ This package does not own file I/O, note locking, project resolution, index stor
 ## Important invariants

 - Frontmatter is optional, but when present it is YAML delimited by `---` lines.
-- Canonical fields include `mnemonic_note_id`, `title`, `slug`, `tags`, `created_at`, `updated_at`, and `type`.
-- `permalink` is only a compatibility fallback for reads when `slug` is missing.
+- Canonical fields include `mnemonic_note_id`, `title`, `slug`, `tags`, `summary`, `aliases`, `created_at`, `updated_at`, and `type`. The `slug` field is the canonical identifier.
+- `tags` and `aliases` must be YAML lists of strings. Scalar values for these fields are rejected.
 - Body parsing must preserve note text while extracting tags, observations, and relations.
 - Rendering writes canonical fields and preserves extra frontmatter fields where possible.
+- Legacy fields (such as `permalink`) are silently stripped during render and rejected by edit operations.

 ## Tests to update when changing this package
```

## Файл: `internal/format/markdown/errors.go`

```go
diff --git a/internal/format/markdown/errors.go b/internal/format/markdown/errors.go
new file mode 100644
index 0000000..1d1c551
--- /dev/null
+++ b/internal/format/markdown/errors.go
@@ -0,0 +1,39 @@
+package markdown
+
+import "fmt"
+
+type FrontmatterFieldError struct {
+	Field string
+	Kind  string
+	Err   error
+}
+
+func (e *FrontmatterFieldError) Error() string {
+	if e.Err != nil {
+		return fmt.Sprintf("frontmatter %q: %s: %v", e.Field, e.Kind, e.Err)
+	}
+	return fmt.Sprintf("frontmatter %q: %s", e.Field, e.Kind)
+}
+
+func (e *FrontmatterFieldError) Unwrap() error {
+	return e.Err
+}
+
+const (
+	FieldErrKindInvalidType       = "invalid_type"
+	FieldErrKindInvalidString     = "invalid_string"
+	FieldErrKindInvalidStringList = "invalid_string_list"
+	FieldErrKindInvalidTimestamp  = "invalid_timestamp"
+)
+
+func newStringFieldError(key string, err error) error {
+	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidString, Err: err}
+}
+
+func newStringSliceFieldError(key string, err error) error {
+	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidStringList, Err: err}
+}
+
+func newTimeFieldError(key string, err error) error {
+	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidTimestamp, Err: err}
+}
```

## Файл: `internal/format/markdown/note.go`

```go
diff --git a/internal/format/markdown/note.go b/internal/format/markdown/note.go
index ab8c04d..3e0c93f 100644
--- a/internal/format/markdown/note.go
+++ b/internal/format/markdown/note.go
@@ -19,19 +19,12 @@ type Note struct {
 	CreatedAt      time.Time
 	UpdatedAt      time.Time
 	Type           string
-	Permalink      string
 	Body           []byte
 }

-// EffectiveSlug returns the canonical slug to write back to frontmatter.
-//
-// Permalink remains a compatibility fallback for older notes that have not yet
-// been rewritten to the canonical `slug` field.
+// EffectiveSlug returns the canonical slug.
 func (n Note) EffectiveSlug() string {
-	if n.Slug != "" {
-		return n.Slug
-	}
-	return n.Permalink
+	return n.Slug
 }

 // ParseNote parses a markdown note into canonical metadata, raw frontmatter, and body.
@@ -76,12 +69,6 @@ func (n *Note) populateFromRaw(raw map[string]any) error {
 	if n.Slug, errField = noteStringField(raw, "slug"); errField != nil {
 		return errField
 	}
-	if n.Permalink, errField = noteStringField(raw, "permalink"); errField != nil {
-		return errField
-	}
-	if n.Slug == "" {
-		n.Slug = n.Permalink
-	}
 	if n.Tags, errField = noteStringSliceField(raw, "tags"); errField != nil {
 		return errField
 	}
@@ -112,7 +99,7 @@ func noteStringField(raw map[string]any, key string) (string, error) {

 	s, ok := value.(string)
 	if !ok {
-		return "", fmt.Errorf("frontmatter %q must be a string", key)
+		return "", newStringFieldError(key, fmt.Errorf("must be a string, got %T", value))
 	}

 	return s, nil
@@ -132,15 +119,13 @@ func noteStringSliceField(raw map[string]any, key string) ([]string, error) {
 		for _, item := range typed {
 			s, ok := item.(string)
 			if !ok {
-				return nil, fmt.Errorf("frontmatter %q items must be strings", key)
+				return nil, newStringSliceFieldError(key, fmt.Errorf("items must be strings, got %T", item))
 			}
 			out = append(out, s)
 		}
 		return out, nil
-	case string:
-		return []string{typed}, nil
 	default:
-		return nil, fmt.Errorf("frontmatter %q must be a string or list of strings", key)
+		return nil, newStringSliceFieldError(key, fmt.Errorf("must be a list of strings, got %T", value))
 	}
 }

@@ -156,6 +141,6 @@ func noteTimeField(raw map[string]any, key string) (time.Time, error) {
 	case int64:
 		return time.Unix(typed, 0).UTC(), nil
 	default:
-		return time.Time{}, fmt.Errorf("frontmatter %q must be a Unix timestamp as an integer, got %T", key, value)
+		return time.Time{}, newTimeFieldError(key, fmt.Errorf("must be a Unix timestamp as an integer, got %T", value))
 	}
 }
```

## Файл: `internal/format/markdown/render.go`

```go
diff --git a/internal/format/markdown/render.go b/internal/format/markdown/render.go
index aa63a19..7a6aea4 100644
--- a/internal/format/markdown/render.go
+++ b/internal/format/markdown/render.go
@@ -19,7 +19,10 @@ var canonicalFrontmatterKeys = map[string]struct{}{
 	"created_at":       {},
 	"updated_at":       {},
 	"type":             {},
-	"permalink":        {},
+}
+
+var removedFrontmatterKeys = map[string]struct{}{
+	"permalink": {},
 }

 // RenderNote serializes a note into canonical YAML frontmatter plus body.
@@ -114,6 +117,9 @@ func renderExtraFrontmatter(buf *bytes.Buffer, note Note) error {
 		if _, ok := canonicalFrontmatterKeys[key]; ok {
 			continue
 		}
+		if _, ok := removedFrontmatterKeys[key]; ok {
+			continue
+		}
 		extraKeys = append(extraKeys, key)
 	}
 	sort.Strings(extraKeys)
```

## Файл: `internal/format/markdown/testdata/basic-memory/permalink.md`

```diff
diff --git a/internal/format/markdown/testdata/basic-memory/permalink.md b/internal/format/markdown/testdata/basic-memory/permalink.md
index 99f5aa7..5b17655 100644
--- a/internal/format/markdown/testdata/basic-memory/permalink.md
+++ b/internal/format/markdown/testdata/basic-memory/permalink.md
@@ -1,5 +1,6 @@
 ---
 title: Session storage redesign
+slug: session-storage-redesign
 permalink: session-storage-redesign
 tags:
   - backend
@@ -9,4 +10,4 @@ type: note

 # Session storage redesign

-This note uses legacy Basic Memory permalink frontmatter.
+This note has both slug and permalink frontmatter fields.
```

## Файл: `internal/service/catalogsvc/service.go`

```go
diff --git a/internal/service/catalogsvc/service.go b/internal/service/catalogsvc/service.go
index 1a3c6de..16d00f7 100644
--- a/internal/service/catalogsvc/service.go
+++ b/internal/service/catalogsvc/service.go
@@ -37,7 +37,6 @@ type Service struct {
 	MemoriesHome string
 	StateHome    string
 	Registry     registry.Store
-	Logger       *slog.Logger
 }

 // ListResult mirrors the project list payload.
@@ -65,6 +64,7 @@ type ShowResult struct {
 	Type               string             `json:"type"`
 	Description        string             `json:"description"`
 	CustomInstructions string             `json:"custom_instructions"`
+	LinksStyle         string             `json:"links_style"`
 	StateHome          string             `json:"state_home"`
 	Location           ShowLocationResult `json:"location"`
 }
@@ -166,7 +166,7 @@ type InitResult struct {
 }

 // Resolve returns the selected knowledge base for a project selector.
-func (s Service) Resolve(selector string) (kb.KnowledgeBase, error) {
+func (s Service) Resolve(selector string, logger *slog.Logger) (kb.KnowledgeBase, error) {
 	selector = strings.TrimSpace(selector)
 	if selector == "" {
 		return kb.KnowledgeBase{}, apperr.CLIUsage("no project selected; specify --project or set MNEMONIC_PROJECT", nil)
@@ -182,8 +182,8 @@ func (s Service) Resolve(selector string) (kb.KnowledgeBase, error) {
 		return kb.KnowledgeBase{}, err
 	}

-	if s.Logger != nil {
-		s.Logger.Info("project selected",
+	if logger != nil {
+		logger.Info("project selected",
 			"slug", resolved.Slug,
 			"kind", resolved.Kind,
 			"root_dir", resolved.RootDir,
@@ -194,7 +194,7 @@ func (s Service) Resolve(selector string) (kb.KnowledgeBase, error) {
 }

 // List returns the registered projects with state paths and issue status.
-func (s Service) List() (ListResult, error) {
+func (s Service) List(logger *slog.Logger) (ListResult, error) {
 	entries, issues, err := s.registryStore().Scan()
 	if err != nil {
 		return ListResult{}, fmt.Errorf("scan registry: %w", err)
@@ -226,8 +226,8 @@ func (s Service) List() (ListResult, error) {
 			item.Issue = issue.Error
 			projects = append(projects, item)

-			if s.Logger != nil {
-				s.Logger.Warn("project issue detected",
+			if logger != nil {
+				logger.Warn("project issue detected",
 					"slug", entry.Slug,
 					"status", item.Status,
 					"error", issue.Error,
@@ -253,8 +253,8 @@ func (s Service) List() (ListResult, error) {
 }

 // Show returns the registered project details for a selector.
-func (s Service) Show(selector string) (ShowResult, error) {
-	resolved, err := s.Resolve(selector)
+func (s Service) Show(selector string, logger *slog.Logger) (ShowResult, error) {
+	resolved, err := s.Resolve(selector, logger)
 	if err != nil {
 		return ShowResult{}, err
 	}
@@ -266,6 +266,7 @@ func (s Service) Show(selector string) (ShowResult, error) {
 		Type:               resolved.Kind,
 		Description:        resolved.Description,
 		CustomInstructions: resolved.CustomInstructions,
+		LinksStyle:         resolved.LinksStyle,
 		StateHome:          resolved.StateDir,
 		Location: ShowLocationResult{
 			MemoriesAbs: resolved.RootDir,
@@ -279,7 +280,7 @@ func (s Service) Show(selector string) (ShowResult, error) {
 // ecosystem. When mnemonic.toml is missing it is generated in-place; all
 // notes lacking canonical frontmatter are hydrated; the project is
 // registered via a pointer file and the search index is rebuilt.
-func (s Service) Import(ctx context.Context, input ImportInput) (ImportResult, error) {
+func (s Service) Import(ctx context.Context, input ImportInput, logger *slog.Logger) (ImportResult, error) {
 	resolvedPath, err := resolveImportPath(input)
 	if err != nil {
 		return ImportResult{}, err
@@ -322,8 +323,8 @@ func (s Service) Import(ctx context.Context, input ImportInput) (ImportResult, e
 	result.Hydrated = hydrateResult.Hydrated
 	result.Skipped = hydrateResult.Skipped

-	if s.Logger != nil {
-		s.Logger.Info("import hydration complete",
+	if logger != nil {
+		logger.Info("import hydration complete",
 			"slug", candidate.Slug,
 			"hydrated", len(hydrateResult.Hydrated),
 			"skipped", len(hydrateResult.Skipped),
@@ -339,12 +340,12 @@ func (s Service) Import(ctx context.Context, input ImportInput) (ImportResult, e
 		return ImportResult{}, err
 	}

-	return s.finalizeImportIndexStatus(ctx, result), nil
+	return s.finalizeImportIndexStatus(ctx, result, logger), nil
 }

 // Add registers an existing structured project (one that already contains a
 // valid mnemonic.toml) and rebuilds its search index.
-func (s Service) Add(ctx context.Context, input AddInput) (AddResult, error) {
+func (s Service) Add(ctx context.Context, input AddInput, logger *slog.Logger) (AddResult, error) {
 	resolvedPath, err := resolveAddPath(input)
 	if err != nil {
 		return AddResult{}, err
@@ -353,12 +354,12 @@ func (s Service) Add(ctx context.Context, input AddInput) (AddResult, error) {
 	manifestPath := filepath.Join(resolvedPath, "mnemonic.toml")
 	if _, statErr := os.Stat(manifestPath); statErr != nil {
 		if os.IsNotExist(statErr) {
-			return AddResult{}, fmt.Errorf("mnemonic.toml not found at %s", resolvedPath)
+			return AddResult{}, apperr.NotFound("mnemonic.toml not found at "+resolvedPath, statErr)
 		}
 		return AddResult{}, fmt.Errorf("stat mnemonic.toml: %w", statErr)
 	}

-	manifest, err := manifestfmt.ParseMnemonicManifestFile(manifestPath)
+	manifest, err := manifestfmt.ParseMnemonicManifestFromFile(manifestPath)
 	if err != nil {
 		return AddResult{}, err
 	}
@@ -374,12 +375,12 @@ func (s Service) Add(ctx context.Context, input AddInput) (AddResult, error) {
 		IndexStatus: "skipped",
 	}

-	resolved, err := s.Resolve(manifest.Slug)
+	resolved, err := s.Resolve(manifest.Slug, logger)
 	if err != nil {
 		result.IndexError = err.Error()
 		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
 	}
-	if _, err := indexsvc.New(resolved).Rebuild(ctx); err != nil {
+	if _, err := indexsvc.New(resolved, logger).Rebuild(ctx); err != nil {
 		result.IndexStatus = "stale"
 		result.IndexError = err.Error()
 		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
@@ -395,7 +396,7 @@ func (s Service) Add(ctx context.Context, input AddInput) (AddResult, error) {
 // not written to disk — so callers can preview the would-be project metadata.
 func ensureManifest(manifestPath, resolvedPath string, dryRun bool) (*manifestfmt.Manifest, bool, error) {
 	if _, statErr := os.Stat(manifestPath); statErr == nil {
-		manifest, parseErr := manifestfmt.ParseMnemonicManifestFile(manifestPath)
+		manifest, parseErr := manifestfmt.ParseMnemonicManifestFromFile(manifestPath)
 		if parseErr != nil {
 			return nil, false, parseErr
 		}
@@ -423,7 +424,8 @@ func ensureManifest(manifestPath, resolvedPath string, dryRun bool) (*manifestfm
 // exists, returning an Ambiguous-style error on conflict.
 func (s Service) registerPointer(pointerPath, manifestPath string) error {
 	if _, err := os.Stat(pointerPath); err == nil {
-		return fmt.Errorf("project slug %q already exists", strings.TrimSuffix(filepath.Base(pointerPath), ".toml"))
+		slug := strings.TrimSuffix(filepath.Base(pointerPath), ".toml")
+		return apperr.Ambiguous(fmt.Sprintf("project slug %q already exists", slug), nil)
 	} else if !os.IsNotExist(err) {
 		return fmt.Errorf("stat pointer file: %w", err)
 	}
@@ -449,14 +451,14 @@ func resolveAddPath(input AddInput) (string, error) {
 	}
 	if _, err := os.Stat(absPath); err != nil {
 		if os.IsNotExist(err) {
-			return "", fmt.Errorf("add path %q not found", path)
+			return "", apperr.NotFound(fmt.Sprintf("add path %q not found", path), err)
 		}
 		return "", fmt.Errorf("stat add path %s: %w", absPath, err)
 	}
 	return absPath, nil
 }

-func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportResult) ImportResult {
+func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportResult, logger *slog.Logger) ImportResult {
 	if result.IndexErrors == nil {
 		result.IndexErrors = []ImportIndexError{}
 	}
@@ -466,7 +468,7 @@ func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportRes
 	}

 	for _, candidate := range result.Candidates {
-		resolved, err := s.Resolve(candidate.Slug)
+		resolved, err := s.Resolve(candidate.Slug, logger)
 		if err != nil {
 			result.IndexErrors = append(result.IndexErrors, ImportIndexError{
 				ProjectID: candidate.ProjectID,
@@ -475,7 +477,7 @@ func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportRes
 			})
 			continue
 		}
-		if _, err := indexsvc.New(resolved).Rebuild(ctx); err != nil {
+		if _, err := indexsvc.New(resolved, logger).Rebuild(ctx); err != nil {
 			result.IndexErrors = append(result.IndexErrors, ImportIndexError{
 				ProjectID: resolved.ID,
 				Slug:      resolved.Slug,
@@ -494,7 +496,7 @@ func (s Service) finalizeImportIndexStatus(ctx context.Context, result ImportRes
 }

 // Init creates a project, resolves it, and builds the initial index.
-func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error) {
+func (s Service) Init(ctx context.Context, input InitInput, logger *slog.Logger) (InitResult, error) {
 	slugValue, err := slug.Slugify(input.Name)
 	if err != nil {
 		return InitResult{}, err
@@ -507,10 +509,10 @@ func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error)
 		Description:  input.Description,
 		Mode:         input.Mode,
 	}); err != nil {
-		return InitResult{}, wrapInitError(err)
+		return InitResult{}, err
 	}

-	resolved, err := s.Resolve(slugValue)
+	resolved, err := s.Resolve(slugValue, logger)
 	if err != nil {
 		return InitResult{}, err
 	}
@@ -528,7 +530,7 @@ func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error)
 		IndexPath:    resolved.IndexPath,
 		IndexStatus:  "stale",
 	}
-	_, err = indexsvc.New(resolved).Rebuild(ctx)
+	_, err = indexsvc.New(resolved, logger).Rebuild(ctx)
 	if err != nil {
 		result.IndexError = err.Error()
 		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
@@ -538,8 +540,8 @@ func (s Service) Init(ctx context.Context, input InitInput) (InitResult, error)
 }

 // Remove removes a project from the registry and state directories.
-func (s Service) Remove(selector string, wipe bool) (RemoveResult, error) {
-	resolved, err := s.Resolve(selector)
+func (s Service) Remove(selector string, wipe bool, logger *slog.Logger) (RemoveResult, error) {
+	resolved, err := s.Resolve(selector, logger)
 	if err != nil {
 		return RemoveResult{}, err
 	}
@@ -630,6 +632,7 @@ func (s Service) knowledgeBaseFromEntry(entry registry.Entry) (kb.KnowledgeBase,
 		Kind:               resolved.Type,
 		Description:        resolved.Description,
 		CustomInstructions: resolved.CustomInstructions,
+		LinksStyle:         resolved.LinksStyle,
 		RootDir:            resolved.MemoriesAbs,
 		RepoRootDir:        resolved.RepoRootAbs,
 		ManifestPath:       resolved.ManifestPath,
@@ -725,19 +728,27 @@ func (s Service) fillEntryFromManifest(resolved *registry.Entry) {
 	if strings.TrimSpace(resolved.CustomInstructions) == "" {
 		resolved.CustomInstructions = manifest.CustomInstructions
 	}
+	resolved.LinksStyle = manifest.Format.LinksStyle
 }

 func (s Service) fillMetadataOnly(resolved *registry.Entry) {
-	if strings.TrimSpace(resolved.Description) != "" && strings.TrimSpace(resolved.CustomInstructions) != "" {
+	if strings.TrimSpace(resolved.Description) != "" && strings.TrimSpace(resolved.CustomInstructions) != "" && resolved.LinksStyle != "" {
 		return
 	}
 	manifest, err := manifestfmt.ParseMnemonicManifestFromFile(resolved.ManifestPath)
-	if err == nil {
-		if strings.TrimSpace(resolved.Description) == "" {
-			resolved.Description = manifest.Description
-		}
-		if strings.TrimSpace(resolved.CustomInstructions) == "" {
-			resolved.CustomInstructions = manifest.CustomInstructions
+	if err != nil {
+		return
+	}
+	if strings.TrimSpace(resolved.Description) == "" {
+		resolved.Description = manifest.Description
+	}
+	if strings.TrimSpace(resolved.CustomInstructions) == "" {
+		resolved.CustomInstructions = manifest.CustomInstructions
+	}
+	if resolved.LinksStyle == "" {
+		resolved.LinksStyle = manifest.Format.LinksStyle
+		if resolved.LinksStyle == "" {
+			resolved.LinksStyle = "wiki"
 		}
 	}
 }
@@ -778,18 +789,6 @@ func (s Service) registryStore() registry.Store {
 	return store
 }

-func wrapInitError(err error) error {
-	if err == nil {
-		return nil
-	}
-
-	if strings.Contains(err.Error(), "project slug") && strings.Contains(err.Error(), "already exists") {
-		return apperr.Ambiguous(err.Error(), nil)
-	}
-
-	return err
-}
-
 // InitProjectInput configures the direct project init helper.
 type InitProjectInput struct {
 	CWD          string
@@ -825,7 +824,7 @@ func initCentralProject(memoriesHome, name, slugValue, description, projectID st
 		return err
 	}
 	if exists {
-		return fmt.Errorf("project slug %q already exists", slugValue)
+		return apperr.Ambiguous(fmt.Sprintf("project slug %q already exists", slugValue), nil)
 	}
 	if err := os.MkdirAll(filepath.Join(memoriesHome, slugValue), 0o755); err != nil {
 		return fmt.Errorf("create central memories directory: %w", err)
@@ -845,7 +844,7 @@ func initLocalProject(memoriesHome, cwd, name, slugValue, description, projectID
 		return err
 	}
 	if exists {
-		return fmt.Errorf("project slug %q already exists", slugValue)
+		return apperr.Ambiguous(fmt.Sprintf("project slug %q already exists", slugValue), nil)
 	}
 	memoriesPath := filepath.Join(cwd, ".mnemonic-memories", slugValue)
 	if err := os.MkdirAll(memoriesPath, 0o755); err != nil {
@@ -861,7 +860,7 @@ func initLocalProject(memoriesHome, cwd, name, slugValue, description, projectID
 }

 func buildInitManifest(projectID, name, slugValue, kind, description string, now time.Time) *manifestfmt.Manifest {
-	m := manifestfmt.NewMnemonicManifest()
+	m := manifestfmt.New()
 	m.ProjectID = projectID
 	m.Name = name
 	m.Slug = slugValue
@@ -889,7 +888,7 @@ func resolveImportPath(input ImportInput) (string, error) {
 	}
 	if _, err := os.Stat(absPath); err != nil {
 		if os.IsNotExist(err) {
-			return "", fmt.Errorf("import path %q not found", path)
+			return "", apperr.NotFound(fmt.Sprintf("import path %q not found", path), err)
 		}
 		return "", fmt.Errorf("stat import path %s: %w", absPath, err)
 	}
```

## Файл: `internal/service/indexsvc/diagnostics.go`

```go
diff --git a/internal/service/indexsvc/diagnostics.go b/internal/service/indexsvc/diagnostics.go
index f688227..cbec34b 100644
--- a/internal/service/indexsvc/diagnostics.go
+++ b/internal/service/indexsvc/diagnostics.go
@@ -2,13 +2,17 @@ package indexsvc

 import (
 	"context"
+	"database/sql"
+	"errors"
 	"os"
 	"path/filepath"
 	"strconv"
 	"strings"

+	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/format/markdown"
 	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
+	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
 )

 // DiagnosticKind enumerates known diagnostic categories.
@@ -18,6 +22,7 @@ const (
 	KindInvalidFrontmatter   DiagnosticKind = "invalid_frontmatter"
 	KindMissingRequiredField DiagnosticKind = "missing_required_field"
 	KindMissingSummary       DiagnosticKind = "missing_summary"
+	KindMissingTimestamp     DiagnosticKind = "missing_timestamp"
 	KindInvalidTimestamp     DiagnosticKind = "invalid_timestamp"
 	KindDuplicateSlug        DiagnosticKind = "duplicate_slug"
 	KindDuplicateAlias       DiagnosticKind = "duplicate_alias"
@@ -47,7 +52,12 @@ type DiagnosticIssue struct {
 	NoteID     string                `json:"note_id,omitempty"`
 	Slug       string                `json:"slug,omitempty"`
 	Path       string                `json:"path,omitempty"`
+	Field      string                `json:"field,omitempty"`
+	SourceLine int                   `json:"source_line,omitempty"`
+	SourceKind string                `json:"source_kind,omitempty"`
+	LinkStyle  string                `json:"link_style,omitempty"`
 	Detail     string                `json:"detail,omitempty"`
+	Target     string                `json:"target,omitempty"`
 	Candidates []DiagnosticCandidate `json:"candidates,omitempty"`
 }

@@ -63,6 +73,16 @@ type DiagnosticCandidate struct {
 func (s Service) Diagnose(ctx context.Context, input DiagnoseInput) (DiagnoseOutput, error) {
 	_ = ctx

+	if input.Limit < 0 {
+		return DiagnoseOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+	}
+	if input.Limit == 0 {
+		input.Limit = 50
+	}
+	if err := validateDiagnoseInput(input); err != nil {
+		return DiagnoseOutput{}, err
+	}
+
 	kindFilter := buildKindFilter(input.Kinds)
 	issues, err := s.collectIssues(kindFilter)
 	if err != nil {
@@ -71,11 +91,11 @@ func (s Service) Diagnose(ctx context.Context, input DiagnoseInput) (DiagnoseOut

 	totalCount := len(issues)
 	cursor := input.Cursor
-	if cursor < 0 {
-		cursor = 0
+	if cursor >= len(issues) {
+		return DiagnoseOutput{TotalCount: totalCount}, nil
 	}
 	limit := input.Limit
-	if limit <= 0 {
+	if limit == 0 {
 		limit = 50
 	}

@@ -84,8 +104,13 @@ func (s Service) Diagnose(ctx context.Context, input DiagnoseInput) (DiagnoseOut
 		end = len(issues)
 	}

+	paged := issues[cursor:end]
+	if input.IncludeSuggestions {
+		paged = s.populateSuggestions(paged)
+	}
+
 	out := DiagnoseOutput{
-		Issues:     issues[cursor:end],
+		Issues:     paged,
 		TotalCount: totalCount,
 	}
 	if end < len(issues) {
@@ -105,6 +130,37 @@ func buildKindFilter(kinds []DiagnosticKind) map[DiagnosticKind]bool {
 	return filter
 }

+var validDiagnosticKinds = map[DiagnosticKind]bool{
+	KindInvalidFrontmatter:   true,
+	KindMissingRequiredField: true,
+	KindMissingSummary:       true,
+	KindMissingTimestamp:     true,
+	KindInvalidTimestamp:     true,
+	KindDuplicateSlug:        true,
+	KindDuplicateAlias:       true,
+	KindUnresolvedLink:       true,
+	KindAmbiguousLink:        true,
+	KindEmptyBody:            true,
+}
+
+func validateDiagnoseInput(input DiagnoseInput) error {
+	if input.Limit < 1 || input.Limit > 200 {
+		return apperr.CLIUsage("limit must be between 1 and 200", nil)
+	}
+	if input.Cursor < 0 {
+		return apperr.CLIUsage("cursor must be >= 0", nil)
+	}
+	if len(input.Kinds) > len(validDiagnosticKinds) {
+		return apperr.CLIUsage("too many diagnostic kinds", nil)
+	}
+	for _, k := range input.Kinds {
+		if !validDiagnosticKinds[k] {
+			return apperr.CLIUsage("unknown diagnostic kind: "+string(k), nil)
+		}
+	}
+	return nil
+}
+
 type kindFilter map[DiagnosticKind]bool

 func (f kindFilter) include(kind DiagnosticKind) bool {
@@ -165,11 +221,7 @@ func (s Service) checkOneNote(root, rel string, filter kindFilter) (issues []Dia

 	note, parseErr := markdown.ParseNote(data)
 	if parseErr != nil {
-		if filter.include(KindInvalidFrontmatter) {
-			issues = append(issues, DiagnosticIssue{
-				Kind: KindInvalidFrontmatter, Path: rel, Detail: parseErr.Error(),
-			})
-		}
+		issues = s.classifyParseError(parseErr, rel, filter)
 		return
 	}

@@ -185,6 +237,33 @@ func (s Service) checkOneNote(root, rel string, filter kindFilter) (issues []Dia
 	return
 }

+func (s Service) classifyParseError(parseErr error, rel string, filter kindFilter) []DiagnosticIssue {
+	var issues []DiagnosticIssue
+	var fieldErr *markdown.FrontmatterFieldError
+	if errors.As(parseErr, &fieldErr) {
+		if fieldErr.Kind == markdown.FieldErrKindInvalidTimestamp {
+			if filter.include(KindInvalidTimestamp) {
+				issues = append(issues, DiagnosticIssue{
+					Kind: KindInvalidTimestamp, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
+				})
+			}
+		} else {
+			if filter.include(KindInvalidFrontmatter) {
+				issues = append(issues, DiagnosticIssue{
+					Kind: KindInvalidFrontmatter, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
+				})
+			}
+		}
+	} else {
+		if filter.include(KindInvalidFrontmatter) {
+			issues = append(issues, DiagnosticIssue{
+				Kind: KindInvalidFrontmatter, Path: rel, Detail: parseErr.Error(),
+			})
+		}
+	}
+	return issues
+}
+
 func (s Service) checkNoteFields(note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
 	issues := make([]DiagnosticIssue, 0, 7)
 	noteID := note.MnemonicNoteID
@@ -219,24 +298,13 @@ func (s Service) checkRequiredFields(noteID, slug, title, path string, filter ki
 }

 func (s Service) checkContentFields(noteID, slug string, note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
-	issues := make([]DiagnosticIssue, 0, 4)
+	issues := make([]DiagnosticIssue, 0, 6)
 	if note.Summary == "" && filter.include(KindMissingSummary) {
 		issues = append(issues, DiagnosticIssue{
 			Kind: KindMissingSummary, NoteID: noteID, Slug: slug, Path: path,
 		})
 	}
-	if !note.CreatedAt.IsZero() && note.CreatedAt.Unix() <= 0 && filter.include(KindInvalidTimestamp) {
-		issues = append(issues, DiagnosticIssue{
-			Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
-			Detail: "created_at <= 0",
-		})
-	}
-	if !note.UpdatedAt.IsZero() && note.UpdatedAt.Unix() <= 0 && filter.include(KindInvalidTimestamp) {
-		issues = append(issues, DiagnosticIssue{
-			Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
-			Detail: "updated_at <= 0",
-		})
-	}
+	issues = append(issues, s.checkTimestamps(noteID, slug, note, path, filter)...)
 	if len(strings.TrimSpace(string(note.Body))) == 0 && filter.include(KindEmptyBody) {
 		issues = append(issues, DiagnosticIssue{
 			Kind: KindEmptyBody, NoteID: noteID, Slug: slug, Path: path,
@@ -245,6 +313,42 @@ func (s Service) checkContentFields(noteID, slug string, note markdown.Note, pat
 	return issues
 }

+func (s Service) checkTimestamps(noteID, slug string, note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
+	var issues []DiagnosticIssue
+	addMissing := func(field string) {
+		if filter.include(KindMissingTimestamp) {
+			issues = append(issues, DiagnosticIssue{
+				Kind: KindMissingTimestamp, NoteID: noteID, Slug: slug, Path: path,
+				Field:  field,
+				Detail: "missing " + field,
+			})
+		}
+	}
+	addInvalid := func(detail, field string) {
+		if filter.include(KindInvalidTimestamp) {
+			issues = append(issues, DiagnosticIssue{
+				Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
+				Field:  field,
+				Detail: detail,
+			})
+		}
+	}
+	if note.CreatedAt.IsZero() {
+		addMissing("created_at")
+	} else if note.CreatedAt.Unix() <= 0 {
+		addInvalid("created_at <= 0", "created_at")
+	}
+	if note.UpdatedAt.IsZero() {
+		addMissing("updated_at")
+	} else if note.UpdatedAt.Unix() <= 0 {
+		addInvalid("updated_at <= 0", "updated_at")
+	}
+	if !note.CreatedAt.IsZero() && !note.UpdatedAt.IsZero() && note.CreatedAt.After(note.UpdatedAt) {
+		addInvalid("created_at > updated_at", "created_at")
+	}
+	return issues
+}
+
 func (s Service) collectDuplicateSlugIssues(
 	seenSlugs map[string][]string, filter kindFilter,
 ) []DiagnosticIssue {
@@ -309,16 +413,85 @@ func (s Service) collectLinkIssues(filter kindFilter) ([]DiagnosticIssue, error)
 			issues = append(issues, DiagnosticIssue{
 				Kind:   KindAmbiguousLink,
 				NoteID: li.NoteID, Slug: li.Slug, Path: li.Path,
-				Detail: "ambiguous link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
+				Target:     li.Target,
+				SourceLine: li.SourceLine,
+				SourceKind: li.SourceKind,
+				LinkStyle:  li.LinkStyle,
+				Detail:     "ambiguous link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
 			})
 		}
 		if !li.IsAmbiguous && filter.include(KindUnresolvedLink) {
 			issues = append(issues, DiagnosticIssue{
 				Kind:   KindUnresolvedLink,
 				NoteID: li.NoteID, Slug: li.Slug, Path: li.Path,
-				Detail: "unresolved link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
+				Target:     li.Target,
+				SourceLine: li.SourceLine,
+				SourceKind: li.SourceKind,
+				LinkStyle:  li.LinkStyle,
+				Detail:     "unresolved link target \"" + li.Target + "\" at line " + strconv.Itoa(li.SourceLine),
 			})
 		}
 	}
 	return issues, nil
 }
+
+func (s Service) populateSuggestions(issues []DiagnosticIssue) []DiagnosticIssue {
+	targets := collectLinkIssueTargets(issues)
+	if len(targets) == 0 {
+		return issues
+	}
+
+	db, err := s.Index.OpenReadonly()
+	if err != nil {
+		return issues
+	}
+	defer func() { _ = db.Close() }()
+
+	candidatesByTarget := searchCandidatesByTarget(s.Index, db, targets)
+
+	for i := range issues {
+		if issues[i].Target != "" && (issues[i].Kind == KindUnresolvedLink || issues[i].Kind == KindAmbiguousLink) {
+			issues[i].Candidates = candidatesByTarget[issues[i].Target]
+		}
+	}
+	return issues
+}
+
+func collectLinkIssueTargets(issues []DiagnosticIssue) map[string]bool {
+	targets := make(map[string]bool)
+	for _, issue := range issues {
+		if issue.Target != "" && (issue.Kind == KindUnresolvedLink || issue.Kind == KindAmbiguousLink) {
+			targets[issue.Target] = true
+		}
+	}
+	return targets
+}
+
+func searchCandidatesByTarget(store sqliteindex.Store, db *sql.DB, targets map[string]bool) map[string][]DiagnosticCandidate {
+	candidatesByTarget := make(map[string][]DiagnosticCandidate, len(targets))
+
+	targetList := make([]string, 0, len(targets))
+	for target := range targets {
+		targetList = append(targetList, target)
+	}
+
+	hitsByTarget, err := store.SearchCandidatesByTargets(db, targetList, 3)
+	if err != nil {
+		return candidatesByTarget
+	}
+
+	for target, hits := range hitsByTarget {
+		candidates := make([]DiagnosticCandidate, 0, len(hits))
+		for _, hit := range hits {
+			candidates = append(candidates, DiagnosticCandidate{
+				NoteID: hit.NoteID,
+				Slug:   hit.Slug,
+				Title:  hit.Title,
+				Path:   hit.Path,
+			})
+		}
+		candidatesByTarget[target] = candidates
+	}
+
+	return candidatesByTarget
+}
```

## Файл: `internal/service/indexsvc/service.go`

```go
diff --git a/internal/service/indexsvc/service.go b/internal/service/indexsvc/service.go
index 6ebb3e2..df9589d 100644
--- a/internal/service/indexsvc/service.go
+++ b/internal/service/indexsvc/service.go
@@ -45,7 +45,7 @@ type DoctorCheck struct {
 }

 // New constructs the runtime index service for one knowledge base.
-func New(k kb.KnowledgeBase) *Service {
+func New(k kb.KnowledgeBase, logger *slog.Logger) *Service {
 	return &Service{
 		KB: k,
 		Notes: markdownstore.Store{
@@ -58,6 +58,7 @@ func New(k kb.KnowledgeBase) *Service {
 			StateDir:  k.StateDir,
 			KBID:      k.ID,
 		},
+		Logger: logger,
 	}
 }

@@ -113,17 +114,6 @@ func (s Service) Doctor(ctx context.Context) (DoctorOutput, error) {
 	}
 	result.addCheck(DoctorCheck{Name: "index quick_check", Status: "ok"})

-	schemaStatus, err := s.Index.SchemaStatus()
-	if err != nil {
-		return DoctorOutput{}, err
-	}
-	if schemaStatus != sqliteindex.SchemaStatusOK {
-		result.Status = "needs_reindex"
-		result.addCheck(DoctorCheck{Name: "index schema", Status: "needs_reindex"})
-		return result, nil
-	}
-	result.addCheck(DoctorCheck{Name: "index schema", Status: "ok"})
-
 	dupUUIDs, dupSlugs, unresolved, trashIgnored, err := doctorNoteChecks(root, s.Index)
 	if err != nil {
 		return DoctorOutput{}, err
```

## Файл: `internal/service/maintsvc/service.go`

```go
diff --git a/internal/service/maintsvc/service.go b/internal/service/maintsvc/service.go
index 4c3ab25..3b1d7e2 100644
--- a/internal/service/maintsvc/service.go
+++ b/internal/service/maintsvc/service.go
@@ -3,6 +3,7 @@ package maintsvc
 import (
 	"context"
 	"errors"
+	"log/slog"
 	"strings"

 	"github.com/ilyachch/mnemonic/internal/domain/kb"
@@ -22,7 +23,7 @@ type Runtime interface {
 }

 // RuntimeFactory builds a runtime for one selected knowledge base.
-type RuntimeFactory func(context.Context, kb.KnowledgeBase) (Runtime, error)
+type RuntimeFactory func(context.Context, kb.KnowledgeBase, *slog.Logger) (Runtime, error)

 // Service owns maintenance operations across many knowledge bases.
 type Service struct {
@@ -65,8 +66,8 @@ func New(catalog *catalogsvc.Service, runtimeFactory RuntimeFactory) *Service {
 }

 // ReindexAll rebuilds the indexes for every catalog project.
-func (s Service) ReindexAll(ctx context.Context) (ReindexAllResult, error) {
-	entries, err := s.catalogEntries()
+func (s Service) ReindexAll(ctx context.Context, logger *slog.Logger) (ReindexAllResult, error) {
+	entries, err := s.catalogEntries(logger)
 	if err != nil {
 		return ReindexAllResult{}, err
 	}
@@ -78,7 +79,7 @@ func (s Service) ReindexAll(ctx context.Context) (ReindexAllResult, error) {
 			Slug:      item.Slug,
 		}

-		resolved, err := s.resolveKnowledgeBase(item.Slug)
+		resolved, err := s.resolveKnowledgeBase(item.Slug, logger)
 		if err != nil {
 			projectResult.Status = "error"
 			projectResult.Error = err.Error()
@@ -87,7 +88,7 @@ func (s Service) ReindexAll(ctx context.Context) (ReindexAllResult, error) {
 			continue
 		}

-		runtime, err := s.runtimeFor(ctx, resolved)
+		runtime, err := s.runtimeFor(ctx, resolved, logger)
 		if err != nil {
 			projectResult.ProjectID = resolved.ID
 			projectResult.Status = "error"
@@ -130,26 +131,26 @@ func (s Service) ReindexAll(ctx context.Context) (ReindexAllResult, error) {
 }

 // DoctorAll runs the runtime doctor checks for every catalog project.
-func (s Service) DoctorAll(ctx context.Context) (DoctorAllResult, error) {
-	entries, err := s.catalogEntries()
+func (s Service) DoctorAll(ctx context.Context, logger *slog.Logger) (DoctorAllResult, error) {
+	entries, err := s.catalogEntries(logger)
 	if err != nil {
 		return DoctorAllResult{}, err
 	}
 	result := DoctorAllResult{Total: len(entries)}
 	for _, item := range entries {
-		s.doctorOne(ctx, item, &result)
+		s.doctorOne(ctx, item, logger, &result)
 	}
 	return result, nil
 }

-func (s Service) doctorOne(ctx context.Context, item catalogsvc.ListItem, result *DoctorAllResult) {
+func (s Service) doctorOne(ctx context.Context, item catalogsvc.ListItem, logger *slog.Logger, result *DoctorAllResult) {
 	projectResult := ProjectResult{ProjectID: item.ProjectID, Slug: item.Slug}
-	resolved, err := s.resolveKnowledgeBase(item.Slug)
+	resolved, err := s.resolveKnowledgeBase(item.Slug, logger)
 	if err != nil {
 		doctorFail(result, &projectResult, err.Error())
 		return
 	}
-	runtime, err := s.runtimeFor(ctx, resolved)
+	runtime, err := s.runtimeFor(ctx, resolved, logger)
 	if err != nil {
 		projectResult.ProjectID = resolved.ID
 		doctorFail(result, &projectResult, err.Error())
@@ -194,28 +195,28 @@ func (s Service) accumulateDoctorStatus(result *DoctorAllResult, status string)
 	}
 }

-func (s Service) catalogEntries() ([]catalogsvc.ListItem, error) {
+func (s Service) catalogEntries(logger *slog.Logger) ([]catalogsvc.ListItem, error) {
 	if s.Catalog == nil {
 		return nil, errors.New("catalog service is not configured")
 	}
-	projects, err := s.Catalog.List()
+	projects, err := s.Catalog.List(logger)
 	if err != nil {
 		return nil, err
 	}
 	return projects.Projects, nil
 }

-func (s Service) resolveKnowledgeBase(selector string) (kb.KnowledgeBase, error) {
+func (s Service) resolveKnowledgeBase(selector string, logger *slog.Logger) (kb.KnowledgeBase, error) {
 	selector = strings.TrimSpace(selector)
 	if selector == "" {
 		return kb.KnowledgeBase{}, errors.New("project selector is required")
 	}
-	return s.Catalog.Resolve(selector)
+	return s.Catalog.Resolve(selector, logger)
 }

-func (s Service) runtimeFor(ctx context.Context, resolved kb.KnowledgeBase) (Runtime, error) {
+func (s Service) runtimeFor(ctx context.Context, resolved kb.KnowledgeBase, logger *slog.Logger) (Runtime, error) {
 	if s.RuntimeFactory == nil {
 		return nil, errors.New("runtime factory is not configured")
 	}
-	return s.RuntimeFactory(ctx, resolved)
+	return s.RuntimeFactory(ctx, resolved, logger)
 }
```

## Файл: `internal/service/notesvc/service.go`

```go
diff --git a/internal/service/notesvc/service.go b/internal/service/notesvc/service.go
index c3762b9..99e2db7 100644
--- a/internal/service/notesvc/service.go
+++ b/internal/service/notesvc/service.go
@@ -1,17 +1,22 @@
 package notesvc

 import (
+	"errors"
+	"log/slog"
 	"time"

+	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/domain/kb"
+	"github.com/ilyachch/mnemonic/internal/platform/clock"
 	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
 	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
 )

 // Service owns runtime note operations for one selected knowledge base.
 type Service struct {
-	Notes markdownstore.Store
-	Index sqliteindex.Store
+	Notes  markdownstore.Store
+	Index  sqliteindex.Store
+	Logger *slog.Logger
 }

 // CreateInput configures note creation.
@@ -40,6 +45,8 @@ type EditInput struct {
 	Body     []byte
 	HasBody  bool
 	Set      map[string]string
+	Tags     *[]string
+	Aliases  *[]string
 	IfMatch  string
 	Now      func() time.Time
 }
@@ -75,14 +82,40 @@ type DeleteResult struct {
 	IndexError  string `json:"index_error,omitempty"`
 }

-// NoteSummary mirrors the legacy note listing payload.
+// NoteSummary aliases the markdownstore note summary type.
 type NoteSummary = markdownstore.NoteSummary

-// ShowResult mirrors the legacy note show payload.
+// ShowResult aliases the markdownstore show result type.
 type ShowResult = markdownstore.ShowResult

+// ReadManyItem holds one resolved note from a batch read.
+type ReadManyItem struct {
+	ShowResult
+	ID string
+}
+
+// ReadManyInput configures a batch note read operation.
+type ReadManyInput struct {
+	Selectors    []string
+	MaxBodyChars int
+}
+
+// ReadManyIssue describes an error encountered when resolving a single selector.
+type ReadManyIssue struct {
+	Selector string `json:"selector"`
+	Kind     string `json:"kind"`
+	Message  string `json:"message"`
+}
+
+// ReadManyOutput is the result of a batch read including missing selectors and per-selector issues.
+type ReadManyOutput struct {
+	Notes   []ReadManyItem
+	Missing []string
+	Issues  []ReadManyIssue
+}
+
 // New constructs the runtime notes service for one knowledge base.
-func New(k kb.KnowledgeBase) *Service {
+func New(k kb.KnowledgeBase, logger *slog.Logger) *Service {
 	return &Service{
 		Notes: markdownstore.Store{RootDir: k.RootDir, StateDir: k.StateDir, IndexPath: k.IndexPath},
 		Index: sqliteindex.Store{
@@ -91,6 +124,7 @@ func New(k kb.KnowledgeBase) *Service {
 			StateDir:  k.StateDir,
 			KBID:      k.ID,
 		},
+		Logger: logger,
 	}
 }

@@ -117,9 +151,15 @@ func (s Service) Create(input CreateInput) (CreateResult, error) {
 	if err := s.rebuildIndex(); err != nil {
 		result.IndexStatus = "stale"
 		result.IndexError = err.Error()
-		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
+		if s.Logger != nil {
+			s.Logger.Warn("note created but index rebuild failed", "note_id", created.NoteID, "path", created.Path, "error", err)
+		}
+		return result, nil
 	}
 	result.IndexStatus = "ok"
+	if s.Logger != nil {
+		s.Logger.Info("note created", "note_id", created.NoteID, "slug", created.Slug, "path", created.Path)
+	}
 	return result, nil
 }

@@ -131,6 +171,8 @@ func (s Service) Edit(input EditInput) (EditResult, error) {
 		Body:     input.Body,
 		HasBody:  input.HasBody,
 		Set:      input.Set,
+		Tags:     input.Tags,
+		Aliases:  input.Aliases,
 		IfMatch:  input.IfMatch,
 		Now:      input.Now,
 	})
@@ -150,9 +192,15 @@ func (s Service) Edit(input EditInput) (EditResult, error) {
 	if err := s.rebuildIndex(); err != nil {
 		result.IndexStatus = "stale"
 		result.IndexError = err.Error()
-		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
+		if s.Logger != nil {
+			s.Logger.Warn("note edited but index rebuild failed", "note_id", edited.NoteID, "error", err)
+		}
+		return result, nil
 	}
 	result.IndexStatus = "ok"
+	if s.Logger != nil {
+		s.Logger.Info("note edited", "note_id", edited.NoteID, "slug", edited.Slug, "path", edited.Path)
+	}
 	return result, nil
 }

@@ -182,9 +230,15 @@ func (s Service) Delete(input DeleteInput) (DeleteResult, error) {
 	if err := s.rebuildIndex(); err != nil {
 		result.IndexStatus = "stale"
 		result.IndexError = err.Error()
-		return result, nil //nolint:nilerr // partial success: IndexError communicates the failure
+		if s.Logger != nil {
+			s.Logger.Warn("note deleted but index rebuild failed", "path", deleted.Path, "error", err)
+		}
+		return result, nil
 	}
 	result.IndexStatus = "ok"
+	if s.Logger != nil {
+		s.Logger.Info("note deleted", "path", deleted.Path, "mode", deleted.Mode)
+	}
 	return result, nil
 }

@@ -195,10 +249,80 @@ func (s Service) List() ([]NoteSummary, error) {

 // Show returns a parsed note from the bound store.
 func (s Service) Show(selector string) (ShowResult, error) {
-	return s.Notes.Show(selector)
+	result, err := s.Notes.Show(selector)
+	if s.Logger != nil {
+		if err != nil {
+			s.Logger.Debug("note read failed", "selector", selector, "error", err)
+		} else {
+			s.Logger.Debug("note read", "note_id", result.Note.MnemonicNoteID, "selector", selector)
+		}
+	}
+	return result, err
 }

-// HydrateResult mirrors the markdownstore hydration payload.
+// ShowMany resolves multiple selectors and returns found notes, missing identifiers,
+// and per-selector issues. Only apperr.NotFound errors go to Missing; other errors
+// (ambiguous, corrupted, io_error, internal) are reported in Issues. Successful notes
+// are returned even when other selectors fail.
+func (s Service) ShowMany(input ReadManyInput) (ReadManyOutput, error) {
+	if len(input.Selectors) < 1 || len(input.Selectors) > 50 {
+		return ReadManyOutput{}, apperr.CLIUsage("identifiers count must be between 1 and 50", nil)
+	}
+	if input.MaxBodyChars < 0 || input.MaxBodyChars > 100000 {
+		return ReadManyOutput{}, apperr.CLIUsage("max_body_chars must be between 0 and 100000", nil)
+	}
+
+	start := clock.NowUTC()
+
+	var notes []ReadManyItem
+	var missing []string
+	var issues []ReadManyIssue
+	for _, identifier := range input.Selectors {
+		resolved, err := s.Show(identifier)
+		if err != nil {
+			missing, issues = classifyShowError(identifier, err, missing, issues)
+			continue
+		}
+		notes = append(notes, ReadManyItem{
+			ShowResult: resolved,
+			ID:         identifier,
+		})
+	}
+
+	if s.Logger != nil {
+		s.Logger.Info("batch read completed",
+			"requested_count", len(input.Selectors),
+			"found_count", len(notes),
+			"missing_count", len(missing),
+			"issue_count", len(issues),
+			"duration", clock.NowUTC().Sub(start),
+		)
+	}
+
+	return ReadManyOutput{Notes: notes, Missing: missing, Issues: issues}, nil
+}
+
+func classifyShowError(selector string, err error, missing []string, issues []ReadManyIssue) ([]string, []ReadManyIssue) {
+	var appErr *apperr.Error
+	if errors.As(err, &appErr) {
+		switch appErr.Code {
+		case apperr.CodeNotFound:
+			return append(missing, selector), issues
+		case apperr.CodeAmbiguous:
+			return missing, append(issues, ReadManyIssue{Selector: selector, Kind: "ambiguous", Message: err.Error()})
+		case apperr.CodeCorrupted:
+			return missing, append(issues, ReadManyIssue{Selector: selector, Kind: "corrupted", Message: err.Error()})
+		case apperr.CodeIO:
+			return missing, append(issues, ReadManyIssue{Selector: selector, Kind: "io_error", Message: err.Error()})
+		default:
+			return missing, append(issues, ReadManyIssue{Selector: selector, Kind: "internal", Message: err.Error()})
+		}
+	}
+
+	return missing, append(issues, ReadManyIssue{Selector: selector, Kind: "internal", Message: err.Error()})
+}
+
+// HydrateResult aliases the markdownstore hydration payload.
 type HydrateResult = markdownstore.HydrateResult

 // Hydrate fills missing canonical frontmatter for raw notes via the bound
```

## Файл: `internal/service/searchsvc/service.go`

```go
diff --git a/internal/service/searchsvc/service.go b/internal/service/searchsvc/service.go
index 4e3dee8..0694bda 100644
--- a/internal/service/searchsvc/service.go
+++ b/internal/service/searchsvc/service.go
@@ -3,7 +3,7 @@ package searchsvc
 import (
 	"context"
 	"log/slog"
-	"strings"
+	"unicode/utf8"

 	"github.com/ilyachch/mnemonic/internal/apperr"
 	"github.com/ilyachch/mnemonic/internal/domain/kb"
@@ -17,25 +17,12 @@ type Service struct {
 	Logger *slog.Logger
 }

-// SearchInput configures a runtime search query.
-type SearchInput struct {
-	Query string
+// ListTagsInput configures tag listing.
+type ListTagsInput struct {
 	Limit int
-	Tag   string
 }

-// SearchResult mirrors the legacy search payload.
-type SearchResult struct {
-	NoteID      string  `json:"note_id"`
-	Slug        string  `json:"slug"`
-	Title       string  `json:"title"`
-	Path        string  `json:"path"`
-	Score       float64 `json:"score"`
-	Snippet     string  `json:"snippet"`
-	ContentHash string  `json:"content_hash"`
-}
-
-// ListTagsOutput mirrors the legacy tag listing payload.
+// ListTagsOutput wraps a tag listing result.
 type ListTagsOutput struct {
 	Tags []ListTagsItem `json:"tags"`
 }
@@ -52,7 +39,7 @@ type BacklinksInput struct {
 	Limit      int
 }

-// Backlink mirrors the legacy backlink payload.
+// Backlink wraps a backlink reference.
 type Backlink struct {
 	LinkID       string `json:"link_id"`
 	NoteID       string `json:"note_id"`
@@ -60,6 +47,7 @@ type Backlink struct {
 	Title        string `json:"title"`
 	Path         string `json:"path"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
 	SourceLine   int    `json:"source_line"`
 }

@@ -81,14 +69,17 @@ type AdvancedSearchInput struct {
 // AdvancedSearchResult mirrors an advanced search hit with optional related
 // notes.
 type AdvancedSearchResult struct {
-	NoteID       string            `json:"note_id"`
-	Slug         string            `json:"slug"`
-	Title        string            `json:"title"`
-	Path         string            `json:"path"`
-	Score        float64           `json:"score"`
-	Snippet      string            `json:"snippet"`
-	ContentHash  string            `json:"content_hash"`
-	RelatedNotes []RelatedNoteItem `json:"related_notes,omitempty"`
+	NoteID         string            `json:"note_id"`
+	Slug           string            `json:"slug"`
+	Title          string            `json:"title"`
+	Path           string            `json:"path"`
+	Score          float64           `json:"score"`
+	Snippet        string            `json:"snippet"`
+	ContentHash    string            `json:"content_hash"`
+	Summary        string            `json:"summary"`
+	Tags           []string          `json:"tags,omitempty"`
+	MatchedQueries []string          `json:"matched_queries,omitempty"`
+	RelatedNotes   []RelatedNoteItem `json:"related_notes,omitempty"`
 }

 // RelatedNoteItem is a short linked-note reference for the service layer.
@@ -98,10 +89,12 @@ type RelatedNoteItem struct {
 	Title        string `json:"title"`
 	Path         string `json:"path"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
+	Direction    string `json:"direction"`
 }

 // New constructs the runtime search service for one knowledge base.
-func New(k kb.KnowledgeBase) *Service {
+func New(k kb.KnowledgeBase, logger *slog.Logger) *Service {
 	return &Service{
 		Index: sqliteindex.Store{
 			IndexPath: k.IndexPath,
@@ -109,52 +102,16 @@ func New(k kb.KnowledgeBase) *Service {
 			StateDir:  k.StateDir,
 			KBID:      k.ID,
 		},
+		Logger: logger,
 	}
 }

-// Search runs a full-text query against the bound index.
-func (s Service) Search(ctx context.Context, input SearchInput) ([]SearchResult, error) {
-	_ = ctx
-	db, err := s.Index.OpenReadonly()
-	if err != nil {
-		return nil, err
-	}
-	defer func() { _ = db.Close() }()
-
-	hits, err := s.Index.Search(db, input.Query, input.Limit, input.Tag)
-	if err != nil {
-		return nil, err
-	}
-
-	if s.Logger != nil {
-		s.Logger.Debug("search executed",
-			"terms", input.Query,
-			"tag", input.Tag,
-			"limit", input.Limit,
-		)
-		s.Logger.Info("search completed",
-			"count", len(hits),
-		)
-	}
-
-	out := make([]SearchResult, 0, len(hits))
-	for _, hit := range hits {
-		out = append(out, SearchResult{
-			NoteID:      hit.NoteID,
-			Slug:        hit.Slug,
-			Title:       hit.Title,
-			Path:        hit.Path,
-			Score:       hit.Score,
-			Snippet:     hit.Snippet,
-			ContentHash: hit.ContentHash,
-		})
-	}
-	return out, nil
-}
-
 // ListTags returns tag counts from the bound index.
-func (s Service) ListTags(ctx context.Context) (ListTagsOutput, error) {
+func (s Service) ListTags(ctx context.Context, input ListTagsInput) (ListTagsOutput, error) {
 	_ = ctx
+	if input.Limit < 0 {
+		return ListTagsOutput{}, apperr.CLIUsage("limit must be >= 0", nil)
+	}
 	db, err := s.Index.OpenReadonly()
 	if err != nil {
 		return ListTagsOutput{}, err
@@ -170,12 +127,18 @@ func (s Service) ListTags(ctx context.Context) (ListTagsOutput, error) {
 	for _, tag := range tags {
 		out.Tags = append(out.Tags, ListTagsItem{Tag: tag.Tag, Count: tag.Count})
 	}
+	if input.Limit > 0 && len(out.Tags) > input.Limit {
+		out.Tags = out.Tags[:input.Limit]
+	}
 	return out, nil
 }

 // Backlinks resolves a note identifier and returns inbound links.
 func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]Backlink, error) {
 	_ = ctx
+	if input.Limit < 0 {
+		return nil, apperr.CLIUsage("limit must be >= 0", nil)
+	}
 	db, err := s.Index.OpenReadonly()
 	if err != nil {
 		return nil, err
@@ -184,9 +147,6 @@ func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]Backlin

 	target, err := s.Index.LookupNoteByIdentifier(db, input.Identifier)
 	if err != nil {
-		if strings.HasPrefix(err.Error(), "note ") && strings.HasSuffix(err.Error(), " not found") {
-			return nil, apperr.NotFound(err.Error(), nil)
-		}
 		return nil, err
 	}

@@ -204,6 +164,7 @@ func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]Backlin
 			Title:        link.Title,
 			Path:         link.Path,
 			RelationType: link.RelationType,
+			SourceKind:   link.SourceKind,
 			SourceLine:   link.SourceLine,
 		})
 	}
@@ -214,6 +175,17 @@ func (s Service) Backlinks(ctx context.Context, input BacklinksInput) ([]Backlin
 // filters, graph-aware reranking, and optional related notes.
 func (s Service) AdvancedSearch(ctx context.Context, input AdvancedSearchInput) ([]AdvancedSearchResult, error) {
 	_ = ctx
+
+	if input.Limit < 0 {
+		return nil, apperr.CLIUsage("limit must be >= 0", nil)
+	}
+	if input.Limit == 0 {
+		input.Limit = 10
+	}
+	if err := validateAdvancedSearchInput(input); err != nil {
+		return nil, err
+	}
+
 	db, err := s.Index.OpenReadonly()
 	if err != nil {
 		return nil, err
@@ -264,18 +236,38 @@ func (s Service) AdvancedSearch(ctx context.Context, input AdvancedSearchInput)
 				Title:        rn.Title,
 				Path:         rn.Path,
 				RelationType: rn.RelationType,
+				SourceKind:   rn.SourceKind,
+				Direction:    rn.Direction,
 			})
 		}
 		out = append(out, AdvancedSearchResult{
-			NoteID:       hit.NoteID,
-			Slug:         hit.Slug,
-			Title:        hit.Title,
-			Path:         hit.Path,
-			Score:        hit.Score,
-			Snippet:      hit.Snippet,
-			ContentHash:  hit.ContentHash,
-			RelatedNotes: related,
+			NoteID:         hit.NoteID,
+			Slug:           hit.Slug,
+			Title:          hit.Title,
+			Path:           hit.Path,
+			Score:          hit.Score,
+			Snippet:        hit.Snippet,
+			ContentHash:    hit.ContentHash,
+			Summary:        hit.Summary,
+			Tags:           hit.Tags,
+			MatchedQueries: hit.MatchedQueries,
+			RelatedNotes:   related,
 		})
 	}
 	return out, nil
 }
+
+func validateAdvancedSearchInput(input AdvancedSearchInput) error {
+	if input.Limit < 1 || input.Limit > 100 {
+		return apperr.CLIUsage("limit must be between 1 and 100", nil)
+	}
+	if len(input.Queries) > 8 {
+		return apperr.CLIUsage("too many queries", nil)
+	}
+	for _, q := range input.Queries {
+		if utf8.RuneCountInString(q) > 500 {
+			return apperr.CLIUsage("query too long", nil)
+		}
+	}
+	return nil
+}
```

## Файл: `internal/store/markdownstore/store.go`

```go
diff --git a/internal/store/markdownstore/store.go b/internal/store/markdownstore/store.go
index 8c77884..6e0c2dd 100644
--- a/internal/store/markdownstore/store.go
+++ b/internal/store/markdownstore/store.go
@@ -60,6 +60,8 @@ type EditInput struct {
 	Body     []byte
 	HasBody  bool
 	Set      map[string]string
+	Tags     *[]string
+	Aliases  *[]string
 	IfMatch  string
 	Now      func() time.Time
 }
@@ -222,6 +224,13 @@ func (s Store) Edit(input EditInput) (EditResult, error) {
 		return EditResult{}, err
 	}

+	if input.Tags != nil {
+		edited.Tags = *input.Tags
+	}
+	if input.Aliases != nil {
+		edited.Aliases = *input.Aliases
+	}
+
 	now := input.Now
 	if now == nil {
 		now = clock.NowUTC
@@ -379,12 +388,12 @@ func (s Store) Show(selector string) (ShowResult, error) {

 	data, err := os.ReadFile(filepath.Join(s.rootDir(), filepath.FromSlash(resolved.Path)))
 	if err != nil {
-		return ShowResult{}, fmt.Errorf("read note %q: %w", resolved.Path, err)
+		return ShowResult{}, apperr.IO(fmt.Sprintf("read note %q", resolved.Path), err)
 	}

 	note, err := markdown.ParseNote(data)
 	if err != nil {
-		return ShowResult{}, fmt.Errorf("parse note %q: %w", resolved.Path, err)
+		return ShowResult{}, apperr.Corrupted(fmt.Sprintf("parse note %q", resolved.Path), err)
 	}

 	return ShowResult{
@@ -880,8 +889,12 @@ func applyEditSet(note *markdown.Note, set map[string]string) error {

 	for key, value := range set {
 		switch key {
-		case "mnemonic_note_id", "created_at":
+		case "mnemonic_note_id", "created_at", "updated_at":
 			return apperr.Unsafe(fmt.Sprintf("frontmatter %q is protected", key), nil)
+		case "tags", "aliases":
+			return apperr.CLIUsage(fmt.Sprintf("frontmatter %q must be updated as a list, not a string", key), nil)
+		case "permalink":
+			return apperr.CLIUsage(fmt.Sprintf("frontmatter %q is removed", key), nil)
 		case "title":
 			note.Title = value
 			note.Frontmatter[key] = value
@@ -891,9 +904,6 @@ func applyEditSet(note *markdown.Note, set map[string]string) error {
 		case "type":
 			note.Type = value
 			note.Frontmatter[key] = value
-		case "permalink":
-			note.Permalink = value
-			note.Frontmatter[key] = value
 		default:
 			note.Frontmatter[key] = value
 		}
```

## Файл: `internal/store/registry/store.go`

```go
diff --git a/internal/store/registry/store.go b/internal/store/registry/store.go
index f94be26..7e0b2cd 100644
--- a/internal/store/registry/store.go
+++ b/internal/store/registry/store.go
@@ -18,6 +18,7 @@ type Entry struct {
 	Type               string // "central" or "local"
 	Description        string
 	CustomInstructions string
+	LinksStyle         string
 	ManifestPath       string
 	MemoriesAbs        string
 	RepoRootAbs        string
```

## Файл: `internal/store/sqliteindex/rebuild.go`

```go
diff --git a/internal/store/sqliteindex/rebuild.go b/internal/store/sqliteindex/rebuild.go
index ec12969..39239a1 100644
--- a/internal/store/sqliteindex/rebuild.go
+++ b/internal/store/sqliteindex/rebuild.go
@@ -131,12 +131,6 @@ func validateDocs(docs []NoteDoc) (seenNoteIDs, seenSlugs map[string]struct{}, s
 func insertNoteDoc(db *sql.DB, doc NoteDoc, kbid string) error {
 	createdAt := doc.CreatedAt
 	updatedAt := doc.UpdatedAt
-	if createdAt == 0 {
-		createdAt = doc.FileMTimeNS / 1e9
-	}
-	if updatedAt == 0 {
-		updatedAt = createdAt
-	}
 	if _, err := db.Exec(`INSERT INTO notes(note_id, project_id, slug, rel_path, title, content_hash, summary, created_at, updated_at)
 		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
 		doc.NoteID, kbid, doc.Slug, doc.RelPath, doc.Title, doc.ContentHash, doc.Summary, createdAt, updatedAt); err != nil {
@@ -207,9 +201,9 @@ func insertDocLinks(db *sql.DB, doc NoteDoc, docs []NoteDoc, seenNorm map[string
 		} else if ambiguous {
 			isAmbiguous = 1
 		}
-		_, _ = db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, is_resolved, is_ambiguous, source_line) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
-			markdownstore.HashBytes([]byte(doc.NoteID+link.RawTarget+link.SourceKind+strconv.Itoa(link.Line))),
-			doc.NoteID, toID, link.RawTarget, link.Label, link.LinkStyle, link.SourceKind, isResolved, isAmbiguous, link.Line)
+		_, _ = db.Exec(`INSERT INTO links(link_id, note_id, to_note_id, target, label, link_style, source_kind, relation_type, is_resolved, is_ambiguous, source_line) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
+			markdownstore.HashBytes([]byte(doc.NoteID+link.RawTarget+link.SourceKind+link.RelationType+strconv.Itoa(link.Line))),
+			doc.NoteID, toID, link.RawTarget, link.Label, link.LinkStyle, link.SourceKind, link.RelationType, isResolved, isAmbiguous, link.Line)
 	}
 }
```

## Файл: `internal/store/sqliteindex/scan.go`

```go
diff --git a/internal/store/sqliteindex/scan.go b/internal/store/sqliteindex/scan.go
index 7e93c21..a12afc0 100644
--- a/internal/store/sqliteindex/scan.go
+++ b/internal/store/sqliteindex/scan.go
@@ -48,14 +48,15 @@ type observationRow struct {
 }

 type linkRow struct {
-	Label       string
-	LinkStyle   string
-	SourceKind  string
-	RawTarget   string
-	ToNoteID    sql.NullString
-	IsResolved  int
-	IsAmbiguous int
-	Line        int
+	Label        string
+	LinkStyle    string
+	SourceKind   string
+	RelationType string
+	RawTarget    string
+	ToNoteID     sql.NullString
+	IsResolved   int
+	IsAmbiguous  int
+	Line         int
 }

 // ScanNotes collects markdown notes from a project root.
@@ -150,11 +151,12 @@ func populateNoteDocData(info *NoteDoc, note markdown.Note, data []byte) {
 	for _, rel := range markdown.ParseRelations(note.Body) {
 		target := strings.TrimSpace(rel.Target.Target)
 		info.Links = append(info.Links, linkRow{
-			RawTarget:  target,
-			Label:      rel.Target.Alias,
-			LinkStyle:  rel.LinkStyle,
-			SourceKind: string(rel.Source),
-			Line:       rel.Line,
+			RawTarget:    target,
+			Label:        rel.Target.Alias,
+			LinkStyle:    rel.LinkStyle,
+			SourceKind:   string(rel.Source),
+			RelationType: rel.RelationType,
+			Line:         rel.Line,
 		})
 	}
 }
```

## Файл: `internal/store/sqliteindex/schema.go`

```go
diff --git a/internal/store/sqliteindex/schema.go b/internal/store/sqliteindex/schema.go
index 4a160ae..d8578e3 100644
--- a/internal/store/sqliteindex/schema.go
+++ b/internal/store/sqliteindex/schema.go
@@ -2,15 +2,10 @@ package sqliteindex

 import "database/sql"

-// ApplySchema creates the index schema for version 2.
+// ApplySchema creates the current index schema in an empty database.
 func ApplySchema(db *sql.DB) error {
 	stmts := []string{
 		`PRAGMA application_id = 1095521358`,
-		`PRAGMA user_version = 2`,
-		`CREATE TABLE IF NOT EXISTS meta (
-			key TEXT PRIMARY KEY,
-			value TEXT NOT NULL
-		)`,
 		`CREATE TABLE IF NOT EXISTS notes (
 			note_id TEXT PRIMARY KEY,
 			project_id TEXT NOT NULL,
@@ -49,6 +44,7 @@ func ApplySchema(db *sql.DB) error {
 			label TEXT DEFAULT '',
 			link_style TEXT DEFAULT '',
 			source_kind TEXT DEFAULT '',
+			relation_type TEXT DEFAULT '',
 			is_resolved INTEGER NOT NULL DEFAULT 0,
 			is_ambiguous INTEGER NOT NULL DEFAULT 0,
 			source_line INTEGER NOT NULL
@@ -68,7 +64,6 @@ func ApplySchema(db *sql.DB) error {
 			body,
 			tokenize = 'unicode61'
 		)`,
-		`INSERT OR IGNORE INTO meta(key, value) VALUES ('schema_version', '2')`,
 	}

 	for _, stmt := range stmts {
@@ -79,15 +74,3 @@ func ApplySchema(db *sql.DB) error {

 	return nil
 }
-
-// CheckSchemaStatus reports whether the current DB schema is compatible.
-func CheckSchemaStatus(db *sql.DB) (SchemaStatus, error) {
-	var version int
-	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
-		return "", err
-	}
-	if version != 2 {
-		return SchemaStatusNeedsRebuild, nil
-	}
-	return SchemaStatusOK, nil
-}
```

## Файл: `internal/store/sqliteindex/schema_check.go`

```go
diff --git a/internal/store/sqliteindex/schema_check.go b/internal/store/sqliteindex/schema_check.go
new file mode 100644
index 0000000..405722d
--- /dev/null
+++ b/internal/store/sqliteindex/schema_check.go
@@ -0,0 +1,67 @@
+package sqliteindex
+
+import (
+	"database/sql"
+	"fmt"
+	"sort"
+)
+
+var requiredTables = map[string][]string{
+	"notes":        {"note_id", "project_id", "slug", "rel_path", "title", "content_hash", "summary", "created_at", "updated_at"},
+	"note_tags":    {"note_id", "tag"},
+	"note_aliases": {"note_id", "alias"},
+	"observations": {"observation_id", "note_id", "kind", "value"},
+	"links":        {"link_id", "note_id", "to_note_id", "target", "label", "link_style", "source_kind", "relation_type", "is_resolved", "is_ambiguous", "source_line"},
+	"notes_fts":    {"note_id", "title", "summary", "tags", "aliases", "body"},
+}
+
+// ValidateSchema checks that the required tables and columns exist in the
+// current index schema. It does not compare versions or attempt migrations.
+func ValidateSchema(db *sql.DB) error {
+	tables := make([]string, 0, len(requiredTables))
+	for table := range requiredTables {
+		tables = append(tables, table)
+	}
+	sort.Strings(tables)
+
+	for _, table := range tables {
+		existing, err := tableColumns(db, table)
+		if err != nil {
+			return fmt.Errorf("inspect table %q: %w", table, err)
+		}
+		columns := requiredTables[table]
+		for _, col := range columns {
+			if !existing[col] {
+				return fmt.Errorf("column %q.%q is missing", table, col)
+			}
+		}
+	}
+	return nil
+}
+
+func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
+	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
+	if err != nil {
+		return nil, err
+	}
+	defer func() { _ = rows.Close() }()
+
+	columns := make(map[string]bool)
+	for rows.Next() {
+		var cid int
+		var name, colType string
+		var notNull, pk int
+		var dflt sql.NullString
+		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
+			return nil, fmt.Errorf("scan table_info %q: %w", table, err)
+		}
+		columns[name] = true
+	}
+	if len(columns) == 0 {
+		return nil, fmt.Errorf("table %q does not exist", table)
+	}
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("iterate table_info %q: %w", table, err)
+	}
+	return columns, nil
+}
```

## Файл: `internal/store/sqliteindex/store.go`

```go
diff --git a/internal/store/sqliteindex/store.go b/internal/store/sqliteindex/store.go
index 8a110c4..b9a0263 100644
--- a/internal/store/sqliteindex/store.go
+++ b/internal/store/sqliteindex/store.go
@@ -6,6 +6,7 @@ import (
 	"fmt"
 	"os"
 	"path/filepath"
+	"sort"
 	"strconv"
 	"strings"
 	"time"
@@ -26,27 +27,6 @@ type Store struct {
 	KBID      string
 }

-// SchemaStatus describes whether an existing index can be reused.
-type SchemaStatus string
-
-const (
-	// SchemaStatusOK indicates that the database schema is compatible.
-	SchemaStatusOK SchemaStatus = "ok"
-	// SchemaStatusNeedsRebuild indicates that the index must be rebuilt.
-	SchemaStatusNeedsRebuild SchemaStatus = "needs_rebuild"
-)
-
-// SearchHit is a single FTS search result from the index database.
-type SearchHit struct {
-	NoteID      string  `json:"note_id"`
-	Slug        string  `json:"slug"`
-	Title       string  `json:"title"`
-	Path        string  `json:"path"`
-	Score       float64 `json:"score"`
-	Snippet     string  `json:"snippet"`
-	ContentHash string  `json:"content_hash"`
-}
-
 // TagCount is a grouped tag row from the index database.
 type TagCount struct {
 	Tag   string `json:"tag"`
@@ -69,6 +49,7 @@ type Backlink struct {
 	Title        string `json:"title"`
 	Path         string `json:"path"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
 	SourceLine   int    `json:"source_line"`
 }

@@ -89,15 +70,18 @@ type SearchOptions struct {

 // SearchResult is an extended search hit that may carry related notes.
 type SearchResult struct {
-	NoteID       string        `json:"note_id"`
-	Slug         string        `json:"slug"`
-	Title        string        `json:"title"`
-	Path         string        `json:"path"`
-	Score        float64       `json:"score"`
-	Snippet      string        `json:"snippet"`
-	ContentHash  string        `json:"content_hash"`
-	RelatedNotes []RelatedNote `json:"related_notes,omitempty"`
-	MatchCount   int           `json:"-"`
+	NoteID         string        `json:"note_id"`
+	Slug           string        `json:"slug"`
+	Title          string        `json:"title"`
+	Path           string        `json:"path"`
+	Score          float64       `json:"score"`
+	Snippet        string        `json:"snippet"`
+	ContentHash    string        `json:"content_hash"`
+	Summary        string        `json:"summary"`
+	Tags           []string      `json:"tags,omitempty"`
+	RelatedNotes   []RelatedNote `json:"related_notes,omitempty"`
+	MatchCount     int           `json:"-"`
+	MatchedQueries []string      `json:"matched_queries,omitempty"`
 }

 // RelatedNote is a short linked-note reference.
@@ -107,6 +91,8 @@ type RelatedNote struct {
 	Title        string `json:"title"`
 	Path         string `json:"path"`
 	RelationType string `json:"relation_type"`
+	SourceKind   string `json:"source_kind"`
+	Direction    string `json:"direction"`
 }

 // Open opens the index database, creating parent directories and applying the
@@ -134,6 +120,10 @@ func (s Store) OpenReadonly() (*sql.DB, error) {
 		_ = db.Close()
 		return nil, fmt.Errorf("ping index database: %w", err)
 	}
+	if err := ValidateSchema(db); err != nil {
+		_ = db.Close()
+		return nil, apperr.Corrupted("index is invalid; run `mnemonic project reindex`", err)
+	}
 	return db, nil
 }

@@ -185,78 +175,6 @@ func (s Store) QuickCheck() error {
 	return nil
 }

-// CheckSchemaStatus reports whether the current DB schema is compatible.
-func (s Store) CheckSchemaStatus(db *sql.DB) (SchemaStatus, error) {
-	return CheckSchemaStatus(db)
-}
-
-// SchemaStatus opens the index read-only and reports whether the schema is compatible.
-func (s Store) SchemaStatus() (SchemaStatus, error) {
-	db, err := s.OpenReadonly()
-	if err != nil {
-		return "", err
-	}
-	defer func() { _ = db.Close() }()
-
-	return CheckSchemaStatus(db)
-}
-
-// Search runs an FTS query against the index database.
-func (s Store) Search(db *sql.DB, query string, limit int, tag string) ([]SearchHit, error) {
-	if db == nil {
-		return nil, errors.New("db is required")
-	}
-	query = sanitizeFTSQuery(query)
-	if strings.TrimSpace(query) == "" {
-		return nil, errors.New("query is required")
-	}
-	if limit <= 0 {
-		limit = 20
-	}
-	tag = strings.TrimSpace(tag)
-
-	sqlQuery := `
-		SELECT n.note_id, n.slug, n.title, n.rel_path, bm25(notes_fts, 10.0, 5.0, 5.0, 2.0, 1.0) AS score,
-		       snippet(notes_fts, 5, '[', ']', '...', 12) AS snippet,
-		       n.content_hash
-		FROM notes_fts
-		JOIN notes n ON n.note_id = notes_fts.note_id
-	`
-	args := []any{query}
-	where := `WHERE notes_fts MATCH ?`
-	if tag != "" {
-		where += ` AND EXISTS (
-			SELECT 1
-			FROM note_tags nt
-			WHERE nt.note_id = n.note_id
-			  AND nt.tag LIKE ?
-		)`
-		args = append(args, "%:"+tag)
-	}
-	sqlQuery += "\n" + where + "\nORDER BY score ASC\nLIMIT ?"
-	args = append(args, limit)
-
-	rows, err := db.Query(sqlQuery, args...)
-	if err != nil {
-		return nil, fmt.Errorf("search query: %w", err)
-	}
-	defer func() { _ = rows.Close() }()
-
-	results := make([]SearchHit, 0)
-	for rows.Next() {
-		var r SearchHit
-		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash); err != nil {
-			return nil, fmt.Errorf("scan search result: %w", err)
-		}
-		results = append(results, r)
-	}
-	if err := rows.Err(); err != nil {
-		return nil, fmt.Errorf("iterate search results: %w", err)
-	}
-
-	return results, nil
-}
-
 // SearchAdvanced runs an advanced search with multi-query, time filters, tag
 // filters, graph-aware reranking, and optional related notes.
 func (s Store) SearchAdvanced(db *sql.DB, opts SearchOptions, now time.Time) ([]SearchResult, error) {
@@ -293,6 +211,11 @@ func (s Store) SearchAdvanced(db *sql.DB, opts SearchOptions, now time.Time) ([]
 		results = results[:opts.Limit]
 	}

+	if err := s.populateSearchTags(db, results); err != nil {
+		return nil, err
+	}
+	s.applySnippetFallback(results)
+
 	if opts.IncludeRelated {
 		if err := s.populateRelatedNotes(db, results); err != nil {
 			return nil, err
@@ -325,91 +248,123 @@ func searchParamsValid(hasQuery, hasTimeFilter, hasTagFilter bool) bool {
 	return hasQuery || hasTimeFilter || hasTagFilter
 }

+const rrfK = 60.0
+
+type normalizedQuery struct {
+	Original string
+	FTS      string
+}
+
 func (s Store) searchMultiQuery(db *sql.DB, opts SearchOptions, ca, cb, ua, ub *int64) ([]SearchResult, error) {
 	timeClause, timeArgs := buildTimeFilterClause(ca, cb, ua, ub)
 	tagClause, tagArgs := buildTagFilterClause(opts.Tags)

-	seen := make(map[string]*SearchResult)
-	for _, q := range opts.Queries {
-		clean := sanitizeFTSQuery(q)
-		if strings.TrimSpace(clean) == "" {
-			continue
-		}
-		hits, err := runFTSSearch(db, clean, opts.Limit, timeClause, timeArgs, tagClause, tagArgs)
+	candidateLimit := opts.Limit * 3
+	if candidateLimit < 30 {
+		candidateLimit = 30
+	}
+
+	queries := dedupQueries(opts.Queries)
+
+	type docEntry struct {
+		result         SearchResult
+		matchCount     int
+		matchedQueries map[string]bool
+		bestRank       int
+	}
+	docs := make(map[string]*docEntry)
+
+	for _, nq := range queries {
+		hits, err := runFTSSearch(db, nq.FTS, candidateLimit, timeClause, timeArgs, tagClause, tagArgs)
 		if err != nil {
 			return nil, err
 		}
-		mergeSearchHits(seen, hits)
+		for rank, hit := range hits {
+			entry, ok := docs[hit.NoteID]
+			if !ok {
+				r := hit
+				r.Score = 0.0
+				entry = &docEntry{result: r, matchedQueries: make(map[string]bool), bestRank: rank}
+				docs[hit.NoteID] = entry
+			}
+			entry.result.Score += 1.0 / (rrfK + float64(rank+1))
+			entry.matchCount++
+			entry.matchedQueries[nq.Original] = true
+			if rank < entry.bestRank {
+				entry.bestRank = rank
+				entry.result.Snippet = hit.Snippet
+			}
+		}
+	}
+
+	results := make([]SearchResult, 0, len(docs))
+	for _, e := range docs {
+		e.result.MatchCount = e.matchCount
+		queries := make([]string, 0, len(e.matchedQueries))
+		for q := range e.matchedQueries {
+			queries = append(queries, q)
+		}
+		sort.Strings(queries)
+		e.result.MatchedQueries = queries
+		results = append(results, e.result)
 	}

-	results := applyMultiQueryBoost(seen)
 	sortSearchResults(results)
 	return results, nil
 }

-func mergeSearchHits(seen map[string]*SearchResult, hits []SearchResult) {
-	for _, hit := range hits {
-		if existing, ok := seen[hit.NoteID]; ok {
-			if hit.Score < existing.Score {
-				existing.Score = hit.Score
-			}
-			existing.MatchCount++
-		} else {
-			r := hit
-			r.MatchCount = 1
-			seen[hit.NoteID] = &r
+func dedupQueries(queries []string) []normalizedQuery {
+	seen := make(map[string]bool, len(queries))
+	result := make([]normalizedQuery, 0, len(queries))
+	for _, q := range queries {
+		clean := sanitizeFTSQuery(q)
+		if strings.TrimSpace(clean) == "" {
+			continue
 		}
-	}
-}
-
-func applyMultiQueryBoost(seen map[string]*SearchResult) []SearchResult {
-	results := make([]SearchResult, 0, len(seen))
-	for _, r := range seen {
-		if r.MatchCount > 1 {
-			r.Score = r.Score * (1.0 - 0.1*float64(r.MatchCount-1))
-			if r.Score < 0 {
-				r.Score = 0
-			}
+		if seen[clean] {
+			continue
 		}
-		results = append(results, *r)
+		seen[clean] = true
+		result = append(result, normalizedQuery{
+			Original: strings.TrimSpace(q),
+			FTS:      clean,
+		})
 	}
-	return results
+	return result
 }

 func (s Store) searchNotesByFilter(db *sql.DB, opts SearchOptions, ca, cb, ua, ub *int64) ([]SearchResult, error) {
-	where := "WHERE 1=1"
+	var w strings.Builder
+	w.WriteString("WHERE 1=1")
 	var args []any

 	if ca != nil {
-		where += " AND n.created_at > ?"
+		w.WriteString(" AND n.created_at > ?")
 		args = append(args, *ca)
 	}
 	if cb != nil {
-		where += " AND n.created_at < ?"
+		w.WriteString(" AND n.created_at < ?")
 		args = append(args, *cb)
 	}
 	if ua != nil {
-		where += " AND n.updated_at > ?"
+		w.WriteString(" AND n.updated_at > ?")
 		args = append(args, *ua)
 	}
 	if ub != nil {
-		where += " AND n.updated_at < ?"
+		w.WriteString(" AND n.updated_at < ?")
 		args = append(args, *ub)
 	}

-	if len(opts.Tags) > 0 {
-		placeholders := make([]string, len(opts.Tags))
-		for i, t := range opts.Tags {
-			placeholders[i] = "?"
-			args = append(args, "%:"+strings.TrimSpace(t))
-		}
-		where += ` AND EXISTS (
-			SELECT 1 FROM note_tags nt
-			WHERE nt.note_id = n.note_id
-			  AND nt.tag LIKE ` + strings.Join(placeholders, " OR nt.tag LIKE ") + ")"
+	uniqTags := dedupAndFilterTags(opts.Tags)
+	for _, t := range uniqTags {
+		w.WriteString(` AND EXISTS (
+		SELECT 1 FROM note_tags nt
+		WHERE nt.note_id = n.note_id
+		  AND nt.tag LIKE ? ESCAPE '\')`)
+		args = append(args, "%:"+escapeTagPattern(strings.TrimSpace(t)))
 	}

-	query := "SELECT n.note_id, n.slug, n.title, n.rel_path, 0.0, '', n.content_hash FROM notes n " + where + " ORDER BY n.created_at DESC LIMIT ?"
+	query := "SELECT n.note_id, n.slug, n.title, n.rel_path, 0.0, '', n.content_hash, n.summary FROM notes n " + w.String() + " ORDER BY n.created_at DESC LIMIT ?"
 	args = append(args, opts.Limit)

 	rows, err := db.Query(query, args...)
@@ -421,7 +376,7 @@ func (s Store) searchNotesByFilter(db *sql.DB, opts SearchOptions, ca, cb, ua, u
 	results := make([]SearchResult, 0)
 	for rows.Next() {
 		var r SearchResult
-		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash); err != nil {
+		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash, &r.Summary); err != nil {
 			return nil, fmt.Errorf("scan filter result: %w", err)
 		}
 		results = append(results, r)
@@ -463,7 +418,11 @@ func (s Store) rerankByGraphLinks(db *sql.DB, results []SearchResult) []SearchRe

 	for i := range results {
 		if conn := connections[i]; conn > 0 {
-			results[i].Score = results[i].Score / (1.0 + 0.1*float64(conn))
+			n := float64(conn)
+			if n > 3 {
+				n = 3
+			}
+			results[i].Score *= 1.0 + 0.05*n
 		}
 	}

@@ -516,6 +475,59 @@ func (s Store) countIncomingLinks(db *sql.DB, topList, allIDs []string, idToIdx
 	_ = rows.Err()
 }

+func (s Store) populateSearchTags(db *sql.DB, results []SearchResult) error {
+	if len(results) == 0 {
+		return nil
+	}
+	ids := make([]string, len(results))
+	for i, r := range results {
+		ids[i] = r.NoteID
+	}
+	rows, err := db.Query(
+		`SELECT DISTINCT note_id,
+			CASE WHEN instr(tag, ':') > 0 THEN substr(tag, instr(tag, ':') + 1) ELSE tag END AS tag
+		 FROM note_tags WHERE note_id IN (`+placeholders(len(ids))+`) ORDER BY note_id, tag`,
+		stringSliceToAny(ids)...,
+	)
+	if err != nil {
+		return fmt.Errorf("query search tags: %w", err)
+	}
+	defer func() { _ = rows.Close() }()
+
+	tagsByID := make(map[string][]string, len(results))
+	for rows.Next() {
+		var noteID, tag string
+		if err := rows.Scan(&noteID, &tag); err != nil {
+			return fmt.Errorf("scan search tag: %w", err)
+		}
+		tagsByID[noteID] = append(tagsByID[noteID], tag)
+	}
+	if err := rows.Err(); err != nil {
+		return fmt.Errorf("iterate search tags: %w", err)
+	}
+	for i := range results {
+		tags := tagsByID[results[i].NoteID]
+		if tags == nil {
+			tags = []string{}
+		}
+		results[i].Tags = tags
+	}
+	return nil
+}
+
+func (s Store) applySnippetFallback(results []SearchResult) {
+	for i := range results {
+		if strings.TrimSpace(results[i].Snippet) != "" {
+			continue
+		}
+		if strings.TrimSpace(results[i].Summary) != "" {
+			results[i].Snippet = results[i].Summary
+		} else {
+			results[i].Snippet = results[i].Title
+		}
+	}
+}
+
 func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {
 	if len(results) == 0 {
 		return nil
@@ -526,15 +538,19 @@ func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {
 		ids[i] = r.NoteID
 	}

-	query := "SELECT l.note_id, l.to_note_id, n.note_id, n.slug, n.title, n.rel_path, l.source_kind " +
+	query := "SELECT * FROM (" +
+		"SELECT l.note_id, l.to_note_id, n.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, 'outgoing', " +
+		"CASE WHEN l.source_kind = 'relations_section' THEN 0 ELSE 1 END " +
 		"FROM links l " +
 		"JOIN notes n ON n.note_id = l.to_note_id " +
 		"WHERE l.note_id IN (" + placeholders(len(ids)) + ") AND l.to_note_id IS NOT NULL " +
 		"UNION ALL " +
-		"SELECT l.to_note_id, l.note_id, n.note_id, n.slug, n.title, n.rel_path, l.source_kind " +
+		"SELECT l.to_note_id, l.note_id, n.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, 'incoming', " +
+		"2 " +
 		"FROM links l " +
 		"JOIN notes n ON n.note_id = l.note_id " +
-		"WHERE l.to_note_id IN (" + placeholders(len(ids)) + ") AND l.to_note_id IS NOT NULL"
+		"WHERE l.to_note_id IN (" + placeholders(len(ids)) + ") AND l.to_note_id IS NOT NULL" +
+		") ORDER BY 10, relation_type, slug, note_id"

 	allIDs := append(stringSliceToAny(ids), stringSliceToAny(ids)...)

@@ -546,8 +562,9 @@ func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {

 	relatedByNoteID := make(map[string][]RelatedNote, len(results))
 	for rows.Next() {
-		var sourceID, targetID, nID, slug, title, path, rel string
-		if err := rows.Scan(&sourceID, &targetID, &nID, &slug, &title, &path, &rel); err != nil {
+		var sourceID, targetID, nID, slug, title, path, relType, sourceKind, direction string
+		var sortPri int
+		if err := rows.Scan(&sourceID, &targetID, &nID, &slug, &title, &path, &relType, &sourceKind, &direction, &sortPri); err != nil {
 			return fmt.Errorf("scan related note: %w", err)
 		}
 		relatedByNoteID[sourceID] = append(relatedByNoteID[sourceID], RelatedNote{
@@ -555,7 +572,9 @@ func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {
 			Slug:         slug,
 			Title:        title,
 			Path:         path,
-			RelationType: rel,
+			RelationType: relType,
+			SourceKind:   sourceKind,
+			Direction:    direction,
 		})
 	}
 	if err := rows.Err(); err != nil {
@@ -564,9 +583,7 @@ func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {

 	for i := range results {
 		rn := relatedByNoteID[results[i].NoteID]
-		if len(rn) > 3 {
-			rn = rn[:3]
-		}
+		rn = dedupRelatedNotes(rn)
 		if rn == nil {
 			rn = []RelatedNote{}
 		}
@@ -576,10 +593,74 @@ func (s Store) populateRelatedNotes(db *sql.DB, results []SearchResult) error {
 	return nil
 }

+const maxRelatedNotes = 3
+
+func dedupRelatedNotes(notes []RelatedNote) []RelatedNote {
+	seen := make(map[string]bool)
+	result := make([]RelatedNote, 0, maxRelatedNotes)
+
+	sortRelatedNotes(notes)
+
+	add := func(rn RelatedNote) {
+		if seen[rn.NoteID] {
+			return
+		}
+		seen[rn.NoteID] = true
+		result = append(result, rn)
+	}
+
+	collect := func(predicate func(RelatedNote) bool) bool {
+		for _, rn := range notes {
+			if predicate(rn) {
+				add(rn)
+				if len(result) >= maxRelatedNotes {
+					return false
+				}
+			}
+		}
+		return true
+	}
+
+	if !collect(func(rn RelatedNote) bool { return rn.SourceKind == "relations_section" }) {
+		return result
+	}
+	if !collect(func(rn RelatedNote) bool { return rn.Direction == "outgoing" && rn.SourceKind != "relations_section" }) {
+		return result
+	}
+	_ = collect(func(rn RelatedNote) bool { return rn.Direction == "incoming" })
+	return result
+}
+
+func sortRelatedNotes(notes []RelatedNote) {
+	sort.Slice(notes, func(i, j int) bool {
+		pi, pj := relatedNotePriority(notes[i]), relatedNotePriority(notes[j])
+		if pi != pj {
+			return pi < pj
+		}
+		if notes[i].RelationType != notes[j].RelationType {
+			return notes[i].RelationType < notes[j].RelationType
+		}
+		if notes[i].Slug != notes[j].Slug {
+			return notes[i].Slug < notes[j].Slug
+		}
+		return notes[i].NoteID < notes[j].NoteID
+	})
+}
+
+func relatedNotePriority(rn RelatedNote) int {
+	if rn.SourceKind == "relations_section" {
+		return 0
+	}
+	if rn.Direction == "outgoing" {
+		return 1
+	}
+	return 2
+}
+
 func runFTSSearch(db *sql.DB, query string, limit int, timeClause string, timeArgs []any, tagClause string, tagArgs []any) ([]SearchResult, error) {
 	sqlQuery := `SELECT n.note_id, n.slug, n.title, n.rel_path, bm25(notes_fts, 10.0, 5.0, 5.0, 2.0, 1.0) AS score,
 		       snippet(notes_fts, 5, '[', ']', '...', 12) AS snippet,
-		       n.content_hash
+		       n.content_hash, n.summary
 		FROM notes_fts
 		JOIN notes n ON n.note_id = notes_fts.note_id
 		WHERE notes_fts MATCH ?`
@@ -605,7 +686,7 @@ func runFTSSearch(db *sql.DB, query string, limit int, timeClause string, timeAr
 	results := make([]SearchResult, 0)
 	for rows.Next() {
 		var r SearchResult
-		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash); err != nil {
+		if err := rows.Scan(&r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash, &r.Summary); err != nil {
 			return nil, fmt.Errorf("scan fts result: %w", err)
 		}
 		results = append(results, r)
@@ -684,13 +765,37 @@ func buildTagFilterClause(tags []string) (string, []any) {
 	if len(tags) == 0 {
 		return "", nil
 	}
-	placeholders := make([]string, len(tags))
-	args := make([]any, len(tags))
-	for i, t := range tags {
-		placeholders[i] = "?"
-		args[i] = "%:" + strings.TrimSpace(t)
+	uniq := dedupAndFilterTags(tags)
+	if len(uniq) == 0 {
+		return "", nil
 	}
-	return "EXISTS (SELECT 1 FROM note_tags nt WHERE nt.note_id = n.note_id AND nt.tag LIKE " + strings.Join(placeholders, " OR nt.tag LIKE ") + ")", args
+	parts := make([]string, len(uniq))
+	args := make([]any, len(uniq))
+	for i, t := range uniq {
+		alias := fmt.Sprintf("nt%d", i)
+		parts[i] = fmt.Sprintf("EXISTS (SELECT 1 FROM note_tags %s WHERE %s.note_id = n.note_id AND %s.tag LIKE ? ESCAPE '\\')", alias, alias, alias)
+		args[i] = "%:" + escapeTagPattern(t)
+	}
+	return strings.Join(parts, " AND "), args
+}
+
+func dedupAndFilterTags(tags []string) []string {
+	seen := make(map[string]bool, len(tags))
+	result := make([]string, 0, len(tags))
+	for _, t := range tags {
+		trimmed := strings.TrimSpace(t)
+		if trimmed == "" || seen[trimmed] {
+			continue
+		}
+		seen[trimmed] = true
+		result = append(result, trimmed)
+	}
+	return result
+}
+
+func escapeTagPattern(tag string) string {
+	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
+	return replacer.Replace(tag)
 }

 func placeholders(n int) string {
@@ -712,7 +817,7 @@ func stringSliceToAny(s []string) []any {
 func sortSearchResults(results []SearchResult) {
 	for i := 0; i < len(results); i++ {
 		for j := i + 1; j < len(results); j++ {
-			if results[j].Score < results[i].Score || (results[j].Score == results[i].Score && results[j].Title < results[i].Title) {
+			if results[j].Score > results[i].Score || (results[j].Score == results[i].Score && results[j].Title < results[i].Title) {
 				results[i], results[j] = results[j], results[i]
 			}
 		}
@@ -775,7 +880,7 @@ func (s Store) LookupNoteByIdentifier(db *sql.DB, identifier string) (IndexedNot
 	var note IndexedNote
 	if err := row.Scan(&note.NoteID, &note.Slug, &note.Title, &note.Path); err != nil {
 		if err == sql.ErrNoRows {
-			return IndexedNote{}, fmt.Errorf("note %q not found", identifier)
+			return IndexedNote{}, apperr.NotFound(fmt.Sprintf("note %q not found", identifier), nil)
 		}
 		return IndexedNote{}, fmt.Errorf("query note %q: %w", identifier, err)
 	}
@@ -791,8 +896,11 @@ func (s Store) Backlinks(db *sql.DB, targetNoteID string, limit int) ([]Backlink
 	if targetNoteID == "" {
 		return nil, errors.New("target note id is required")
 	}
+	if limit < 0 {
+		return nil, errors.New("limit must be >= 0")
+	}

-	sqlQuery := `SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path, l.source_kind, l.source_line
+	sqlQuery := `SELECT l.link_id, l.note_id, n.slug, n.title, n.rel_path, l.relation_type, l.source_kind, l.source_line
 		 FROM links l
 		 JOIN notes n ON n.note_id = l.note_id
 		 WHERE l.to_note_id = ?
@@ -812,7 +920,7 @@ func (s Store) Backlinks(db *sql.DB, targetNoteID string, limit int) ([]Backlink
 	out := make([]Backlink, 0)
 	for rows.Next() {
 		var item Backlink
-		if err := rows.Scan(&item.LinkID, &item.NoteID, &item.Slug, &item.Title, &item.Path, &item.RelationType, &item.SourceLine); err != nil {
+		if err := rows.Scan(&item.LinkID, &item.NoteID, &item.Slug, &item.Title, &item.Path, &item.RelationType, &item.SourceKind, &item.SourceLine); err != nil {
 			return nil, fmt.Errorf("scan backlink: %w", err)
 		}
 		out = append(out, item)
@@ -899,6 +1007,86 @@ func (s Store) UnresolvedLinkCount() (int, error) {
 	return s.CountUnresolvedLinks(db)
 }

+// SearchCandidatesByTargets looks up candidate notes for multiple link targets in a single
+// composite query. Returns at most 3 candidates per target with deterministic ordering.
+func (s Store) SearchCandidatesByTargets(db *sql.DB, targets []string, limitPerTarget int) (map[string][]SearchResult, error) {
+	if db == nil {
+		return nil, errors.New("db is required")
+	}
+	if len(targets) == 0 {
+		return map[string][]SearchResult{}, nil
+	}
+	if limitPerTarget <= 0 {
+		limitPerTarget = 3
+	}
+
+	copied := append([]string(nil), targets...)
+	sort.Strings(copied)
+	sortedTargets := dedupSortedStrings(copied)
+
+	parts := make([]string, 0, len(sortedTargets))
+	args := make([]any, 0, len(sortedTargets)*3)
+	for _, target := range sortedTargets {
+		ftsQuery := sanitizeFTSQuery(target)
+		if strings.TrimSpace(ftsQuery) == "" {
+			continue
+		}
+		parts = append(parts, `SELECT * FROM (
+			SELECT ? AS _target, n.note_id, n.slug, n.title, n.rel_path,
+				bm25(notes_fts, 10.0, 5.0, 5.0, 2.0, 1.0) AS score,
+				snippet(notes_fts, 5, '[', ']', '...', 12) AS snippet,
+				n.content_hash, n.summary
+			FROM notes_fts
+			JOIN notes n ON n.note_id = notes_fts.note_id
+			WHERE notes_fts MATCH ?
+			ORDER BY score ASC, n.slug ASC, n.note_id ASC
+			LIMIT ?
+		)`)
+		args = append(args, target, ftsQuery, limitPerTarget)
+	}
+
+	if len(parts) == 0 {
+		return map[string][]SearchResult{}, nil
+	}
+
+	query := "SELECT * FROM (" + strings.Join(parts, " UNION ALL ") + ") ORDER BY _target, score, slug, note_id"
+	rows, err := db.Query(query, args...)
+	if err != nil {
+		return nil, fmt.Errorf("candidate search: %w", err)
+	}
+	defer func() { _ = rows.Close() }()
+
+	result := make(map[string][]SearchResult, len(targets))
+	for rows.Next() {
+		var target string
+		var r SearchResult
+		if err := rows.Scan(&target, &r.NoteID, &r.Slug, &r.Title, &r.Path, &r.Score, &r.Snippet, &r.ContentHash, &r.Summary); err != nil {
+			return nil, fmt.Errorf("scan candidate: %w", err)
+		}
+		result[target] = append(result[target], r)
+	}
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("iterate candidates: %w", err)
+	}
+
+	return result, nil
+}
+
+func dedupSortedStrings(sorted []string) []string {
+	if len(sorted) <= 1 {
+		return sorted
+	}
+	out := make([]string, 0, len(sorted))
+	prev := ""
+	for _, s := range sorted {
+		if s != prev {
+			out = append(out, s)
+			prev = s
+		}
+	}
+	return out
+}
+
 func (s Store) validateIndexPath() error {
 	if strings.TrimSpace(s.IndexPath) == "" {
 		return errors.New("index path is required")
@@ -929,7 +1117,6 @@ func openDB(path string) (*sql.DB, error) {
 		`PRAGMA foreign_keys = ON`,
 		`PRAGMA journal_mode = WAL`,
 		`PRAGMA busy_timeout = 5000`,
-		`PRAGMA user_version = 2`,
 	}
 	for _, pragma := range pragmas {
 		if _, err := db.Exec(pragma); err != nil {
```

## Файл: `scripts/mcp_smoke.go`

```go
diff --git a/scripts/mcp_smoke.go b/scripts/mcp_smoke.go
index 13a3d26..a777f03 100644
--- a/scripts/mcp_smoke.go
+++ b/scripts/mcp_smoke.go
@@ -293,7 +293,7 @@ func prepareSmokeProject() (string, []string, error) {
 	}

 	now := time.Now().UTC()
-	manifest := manifestfmt.NewMnemonicManifest()
+	manifest := manifestfmt.New()
 	manifest.ProjectID = idgen.NewUUID()
 	manifest.Name = "personal"
 	manifest.Slug = "personal"
```
