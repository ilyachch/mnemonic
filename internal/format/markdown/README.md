# `internal/format/markdown`

## What this package owns

This package owns the on-disk Markdown note format understood by `mnemonic`: frontmatter splitting, canonical note parsing, note rendering, inline tag parsing, observation parsing, relation parsing, and wiki-link parsing.

## What this package does not own

This package does not own file I/O, note locking, project resolution, index storage, or search ranking. It should stay focused on parsing and rendering Markdown structures.

## Important invariants

- Frontmatter is optional, but when present it is YAML delimited by `---` lines.
- Canonical fields include `mnemonic_note_id`, `title`, `slug`, `tags`, `summary`, `aliases`, `created_at`, `updated_at`, and `type`. The `slug` field is the canonical identifier.
- `tags` and `aliases` must be YAML lists of strings. Scalar values for these fields are rejected.
- Body parsing must preserve note text while extracting tags, observations, and relations.
- Rendering writes canonical fields and preserves extra frontmatter fields where possible.
- Legacy fields (such as `permalink`) are silently stripped during render and rejected by edit operations.

## Tests to update when changing this package

- `internal/format/markdown/frontmatter_test.go`
- `internal/format/markdown/note_test.go`
- `internal/format/markdown/render_test.go`
- `internal/format/markdown/tag_test.go`
- `internal/format/markdown/observation_test.go`
- `internal/format/markdown/relation_test.go`
- `internal/format/markdown/wikilink_test.go`
- `internal/format/markdown/basic_memory_test.go`
- `internal/notes/*_test.go` and `internal/index/*_test.go` when parser output changes downstream behavior
