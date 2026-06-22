# Markdown Format

## Purpose

The Markdown format defines the stable on-disk note shape that `mnemonic` reads and writes. The parser keeps raw body text intact while extracting the subset of metadata and structures that the rest of the system indexes.

## Invariants

- Notes are plain Markdown files with optional YAML frontmatter.
- Canonical frontmatter fields include `mnemonic_note_id`, `title`, `slug`, `tags`, `created_at`, `updated_at`, and `type`.
- `permalink` is accepted as a compatibility fallback for slug resolution when `slug` is missing.
- Body parsing must preserve note text while extracting tags, relations, and observations.
- Rendering should emit canonical fields and preserve non-canonical extra frontmatter where supported.

## Parsed structures

- Frontmatter is split from the body without altering body bytes.
- Inline tags are parsed from `#tag` tokens outside fenced code blocks.
- Observations use `- [category] content` lines.
- Relations come from wiki-links and from the `## Relations` section.
- Wiki-links support raw targets and optional aliases.

## Current write behavior

New and edited notes are rendered with canonical YAML frontmatter. Rendering preserves extra frontmatter fields but does not write legacy `permalink` back out as a canonical field.

## Related packages

- `internal/format/markdown` for parsing and rendering.
- `internal/notes` for create/edit/delete flows.
- `internal/index` for turning parsed notes into index rows.
