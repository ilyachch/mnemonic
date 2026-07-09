# Mnemonic Maintenance Reference

## `doctor`

Runs broad index and content health checks for the active project.

- Arguments: none.
- Read-only.
- Use first for classification and after repairs for verification.

## `diagnose_notes`

Scans notes for metadata, identity, link, and content issues.

| Parameter | Type | Rules |
|---|---|---|
| `kinds` | array of strings | Optional diagnostic filters |
| `limit` | integer | 1-200; default 50 |
| `cursor` | integer | Zero-based offset; non-negative |
| `include_suggestions` | boolean | Candidate notes for unresolved or ambiguous links |

Valid kinds:

- `invalid_frontmatter`
- `missing_required_field`
- `missing_summary`
- `duplicate_slug`
- `duplicate_alias`
- `unresolved_link`
- `ambiguous_link`
- `empty_body`

An issue can include `kind`, `note_id`, `slug`, `path`, `field`, `source_line`, `source_kind`, `link_style`, `detail`, `target`, and `candidates`.

## `read_notes`

Use before every MCP mutation. Recommended repair fields:

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

Maximum selectors per call: 50. Invalid frontmatter can cause a selector issue with kind `corrupted`; use the raw-file fallback in that case.

## `edit_note`

Mutation modes are mutually exclusive:

- `append`
- `replace_body`
- `merge_frontmatter`
- typed `tags` and `aliases`

Use `if_match_hash` from a fresh read. Do not pass `tags` or `aliases` through `merge_frontmatter`.

Canonical scalar fields such as `slug`, `title`, `summary`, and `type` may be changed through `merge_frontmatter`. Protected or removed fields include `mnemonic_note_id`, `created_at`, `updated_at`, and `permalink`.

Duplicate-slug repair example:

```json
{
  "identifier": "note-id",
  "merge_frontmatter": {"slug": "new-unique-slug"},
  "if_match_hash": "fresh-content-hash"
}
```

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
| `hard_delete` | boolean | False moves to trash; true permanently removes |
| `if_match_hash` | string | Server-required for hard delete; skill-required for all deletes |

## `rebuild_index`

Rebuilds the disposable FTS index from Markdown sources. It does not repair source-note defects.

## Relevant CLI Commands

```bash
mnemonic --json project doctor PROJECT
mnemonic --json project doctor PROJECT --kind unresolved_link --kind ambiguous_link
mnemonic --json project reindex PROJECT
mnemonic project sync PROJECT
```

The CLI diagnostic command does not expose `limit`, `cursor`, or candidate-suggestion flags. It therefore returns the service's default first page, currently up to 50 issues. Use MCP `diagnose_notes` when explicit pagination or suggestions are required.

## CLI Diagnostic JSON Contract Used by `repair-loop.sh`

Kind-filtered `mnemonic --json project doctor` output is expected to contain:

```json
{
  "issues": [],
  "total_count": 0,
  "next_cursor": 0
}
```

- `issues` is always an array.
- `total_count` is the total matching issue count.
- `next_cursor` is omitted or zero when pagination is complete.
- A nonzero `next_cursor` indicates that more matching issues exist, but the current CLI command cannot request that cursor directly.

The script validates the object and `issues` array before invoking a repair hook. Each pass gives the hook only the returned page. After those issues are repaired, the next pass reruns diagnostics and surfaces remaining issues from the new first page. Increase `MAX_PASSES` for large repositories.

## Raw-File Fallback

Filesystem access is required only when a malformed note cannot be read through MCP.

1. Use the diagnostic `path` within the selected project's Markdown root.
2. Back up the file.
3. Repair the YAML delimiter, syntax, or list shape without altering unrelated body content.
4. Preserve `mnemonic_note_id` when recoverable.
5. Run project sync or reindex and then doctor.
