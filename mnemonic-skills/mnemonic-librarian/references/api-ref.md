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

`tags` and `aliases` must be YAML lists of strings. Current rendering strips legacy `permalink`, `created_at`, and `updated_at` fields.

## `list_tags`

Lists existing tags and their usage counts. Use before assigning tags to new notes.

## `create_note`

| Parameter | Type | Notes |
|---|---|---|
| `title` | string | Required; mnemonic derives the slug from this title |
| `body` | string | Optional Markdown body |
| `tags` | array of strings | Optional frontmatter tags |

Returns:

- `note_id`
- `slug`
- `path`
- `content_hash`
- `index_status`
- Optional `index_error`

The server generates `mnemonic_note_id`. The tool does not accept arbitrary frontmatter or a summary at creation time.

## `read_notes`

Batch-read up to 50 selectors.

Recommended fields before editing:

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
| `identifier` | string | Required note ID, slug, path, or title |
| `append` | string | Append body text |
| `replace_body` | string | Replace the complete body; requires `if_match_hash` |
| `merge_frontmatter` | object of string values | Update scalar metadata; do not use for tags or aliases |
| `tags` | array or empty array | Replace or clear tags |
| `aliases` | array or empty array | Replace or clear aliases |
| `if_match_hash` | string | Expected current content hash |

Edit modes are mutually exclusive. A body change and a summary change therefore require separate calls, using the returned hash from the first call in the second.

Presence semantics for `tags` and `aliases`:

- Field absent: no change.
- Empty array: clear the list.
- Non-empty array: replace the list.

## Link Styles

Wiki style:

```markdown
[[target-slug|Display Label]]
```

Regular style:

```markdown
[Display Label](target-slug.md)
```

Use the style configured in `mnemonic.toml` under `format.links_style` or supplied by the MCP project instructions.
