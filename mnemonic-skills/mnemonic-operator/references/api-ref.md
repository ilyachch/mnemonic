# Mnemonic Maintenance MCP Reference

## `doctor`

Runs broad index and content health checks for the active project.

- Arguments: none.
- Read-only.
- Use first for classification and after repairs for final verification.

## `diagnose_notes`

Scans notes for detailed metadata, identity, link, and content issues.

| Parameter | Type | Rules |
|---|---|---|
| `kinds` | array of strings | Optional diagnostic filters |
| `limit` | integer | 1-200; default 50 |
| `cursor` | integer | Zero-based offset; must be non-negative |
| `include_suggestions` | boolean | Adds candidate notes for unresolved or ambiguous links |

Valid kinds:

- `invalid_frontmatter`
- `missing_required_field`
- `missing_summary`
- `duplicate_slug`
- `duplicate_alias`
- `unresolved_link`
- `ambiguous_link`
- `empty_body`

Each issue may include `kind`, `note_id`, `slug`, `path`, `field`, `source_line`, `source_kind`, `link_style`, `detail`, `target`, and `candidates`.

## `read_notes`

Use before every mutation.

Recommended fields for repair work:

```json
[
  "body",
  "frontmatter",
  "tags",
  "aliases",
  "path",
  "content_hash"
]
```

Maximum identifiers per call: 50.

## `edit_note`

Mutation modes are mutually exclusive:

- `append`
- `replace_body`
- `merge_frontmatter`
- typed `tags` and `aliases`

Use `if_match_hash` from a fresh `read_notes` result. `replace_body` requires it. This skill requires it for every edit.

Do not pass `tags` or `aliases` through `merge_frontmatter`.

## `list_backlinks`

Required before deletion, slug changes, or note consolidation.

```json
{
  "identifier": "note-id-or-slug",
  "limit": 0
}
```

## `delete_note`

| Parameter | Type | Notes |
|---|---|---|
| `identifier` | string | Required |
| `hard_delete` | boolean | False moves to trash; true permanently removes the file |
| `if_match_hash` | string | Required by the server for hard deletion; required by this skill for all deletions |

## `rebuild_index`

Rebuilds the FTS index from Markdown source files. The index is disposable and structurally validated. Rebuilding does not repair source-note defects.

## Relevant CLI Commands

```bash
mnemonic --json project doctor PROJECT
mnemonic --json project doctor PROJECT --kind unresolved_link --kind ambiguous_link
mnemonic --json project reindex PROJECT
```

The CLI diagnostic command does not expose candidate suggestions. Use the MCP `diagnose_notes` tool when suggestions are required.
