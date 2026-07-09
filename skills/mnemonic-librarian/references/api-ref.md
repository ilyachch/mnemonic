# Mnemonic Note MCP Reference

## Canonical Format

The only strictly required frontmatter field is `mnemonic_note_id`.

Canonical fields can include:

- `mnemonic_note_id`
- `title`
- `slug`
- `tags`
- `summary`
- `aliases`
- `type`

`tags` and `aliases` must be YAML lists of strings. Rendering strips legacy `permalink`, `created_at`, and `updated_at` fields.

## Required Tool Set

This skill expects:

- `search_notes`
- `read_notes`
- `list_tags`
- `create_note`
- `edit_note`

## `list_tags`

Lists existing tags and usage counts. Use before assigning tags unless the user supplied an authoritative set.

## `create_note`

| Parameter | Type | Notes |
|---|---|---|
| `title` | string | Required; stored as the title and used to derive the slug |
| `body` | string | Optional Markdown body |
| `tags` | array of strings | Optional frontmatter tags |

Returns `note_id`, `slug`, `path`, `content_hash`, `index_status`, and optional `index_error`.

The server generates `mnemonic_note_id`. Creation does not accept arbitrary frontmatter or a summary.

## `read_notes`

Batch-read up to 50 selectors. Recommended fields before editing:

```json
[
  "body",
  "summary",
  "tags",
  "aliases",
  "frontmatter",
  "content_hash",
  "path"
]
```

## `edit_note`

| Parameter | Type | Notes |
|---|---|---|
| `identifier` | string | Required ID, slug, path, or title |
| `append` | string | Append body text |
| `replace_body` | string | Replace body; requires `if_match_hash` |
| `merge_frontmatter` | object of string values | Scalar metadata; not tags or aliases |
| `tags` | array or empty array | Replace or clear tags |
| `aliases` | array or empty array | Replace or clear aliases |
| `if_match_hash` | string | Expected current hash |

Edit modes are mutually exclusive. Sequential body and metadata edits must use the hash returned by the preceding edit.

Presence semantics for `tags` and `aliases`:

- Absent: no change.
- Empty array: clear.
- Non-empty array: replace.

## Link Styles

Wiki:

```markdown
[[target-slug|Display Label]]
```

Regular:

```markdown
[Display Label](target-slug.md)
```

Use `format.links_style` from `mnemonic.toml` or MCP project instructions. If neither is available, do not guess a style.
