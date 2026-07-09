# Mnemonic Search MCP Reference

## `search_notes`

Multi-query FTS5 search with Reciprocal Rank Fusion, time filters, tag filters, and graph-aware reranking.

| Parameter | Type | Rules |
|---|---|---|
| `queries` | array of strings | Optional; maximum 8; maximum 500 Unicode characters each |
| `tags` | array of strings | Optional; AND logic |
| `created_before` | integer | Optional Unix timestamp upper bound |
| `created_after` | integer | Optional Unix timestamp lower bound |
| `updated_before` | integer | Optional Unix timestamp upper bound |
| `updated_after` | integer | Optional Unix timestamp lower bound |
| `created_since` | string | Optional relative duration such as `24h` or `7d` |
| `updated_since` | string | Optional relative duration such as `24h` or `7d` |
| `limit` | integer | 1-100; default 10 |
| `include_related` | boolean | Include forward links and backlinks per hit |
| `debug` | boolean | Include path, score, and content hash |

Each hit can include `note_id`, `slug`, `title`, `snippet`, `summary`, `tags`, `matched_queries`, and optionally `related_notes`, `path`, `score`, and `content_hash`.

## `read_notes`

Batch-read up to 50 notes by note ID, slug, path, or title.

| Parameter | Type | Rules |
|---|---|---|
| `identifiers` | array of strings | Required; maximum 50 |
| `fields` | array of strings | Optional values: `summary`, `tags`, `body`, `path`, `frontmatter`, `content_hash`, `aliases`, `created_at`, `updated_at` |
| `max_body_chars` | integer | Optional; maximum 100000 |

Unresolved selectors appear in `missing`. Per-selector failures appear in `issues` with kinds such as `ambiguous`, `corrupted`, `io_error`, or `internal`.

## `list_tags`

Lists tags and usage counts.

- `limit: 0` means no explicit limit.
- A positive limit caps the result count.
- A negative limit is invalid.

## `list_backlinks`

Lists notes that link to a selected note.

| Parameter | Type | Rules |
|---|---|---|
| `identifier` | string | Required note ID, slug, or title |
| `limit` | integer | `0` means no explicit limit; negative values are invalid |

## Search Semantics

- FTS uses SQLite FTS5 with the `unicode61` tokenizer.
- Rankings from query variants are combined using Reciprocal Rank Fusion.
- Tag filters use AND logic.
- `include_related` returns graph neighbors but does not replace reading their content.
- File timestamps are used when legacy YAML timestamps are absent.
