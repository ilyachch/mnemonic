---
name: mnemonic-librarian
description: >-
  Use this skill for ordinary creation and content editing of mnemonic notes: canonical Markdown structure, summaries, tags, aliases, links, and optimistic concurrency. Do not use it for bulk diagnostics, broken-link repair, deletion, duplicate identity repair, or index recovery; use mnemonic-operator for maintenance.
compatibility: >-
  Requires a mnemonic MCP server exposing search_notes, read_notes, list_tags, create_note, and edit_note, with note mutations enabled.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Content Librarian

Create readable, connected, safely versioned notes. Preserve user content, minimize unrelated changes, and use the current hash for every edit.

See [the note API reference](references/api-ref.md) for exact tool behavior.

## Canonical Note Principles

- `mnemonic_note_id` is the only strictly required frontmatter field.
- `create_note` generates `mnemonic_note_id`. Never invent an ID or put simulated frontmatter in the body.
- Mnemonic stores the supplied title and derives the slug from it.
- Start newly authored body content with one H1 matching the title.
- Add a concise summary after creation.
- `tags` and `aliases` must be lists of strings.
- Do not write `created_at` or `updated_at`; current mnemonic resolves timestamps from the filesystem and strips legacy timestamp fields on render.
- Do not use `permalink`.

## Link Style

Use the project's configured style.

Wiki:

```markdown
[[target-slug|Display Label]]
```

Regular:

```markdown
[Display Label](target-slug.md)
```

If link style is unavailable, inspect project instructions or configuration. Do not guess. Omit a new link when it is nonessential; otherwise request the project configuration before adding it.

## Creation Checklist

### 1. Choose the Title

Use a title that is unique, stable, and focused on one primary concept or decision.

Anticipate the derived slug, but do not fabricate a separate slug unless explicitly required and safely supported.

### 2. Discover Existing Tags

Call `list_tags` before assigning tags unless the user supplied a complete authoritative set.

Prefer existing vocabulary. Add a new tag only when no existing tag fits. Keep tags categorical and reusable.

### 3. Prepare the Body

Use this structure when it fits:

```markdown
# Note Title

A concise opening paragraph that states the subject.

## Context

Why this information matters.

## Details

Durable facts, reasoning, examples, or implementation notes.

## Related Notes

- [[related-slug|Related Note]]
```

Do not force empty sections. The H1 is required for newly authored notes; other sections depend on content.

### 4. Write a Summary

Create a one- or two-sentence summary that explains durable value. Avoid vague text such as "Notes about Go".

### 5. Create the Note

Call `create_note` with title, H1-led body, and selected tags. The result supplies `note_id`, derived `slug`, path, and `content_hash`.

### 6. Add the Summary Safely

Use the creation hash:

```json
{
  "identifier": "generated-slug",
  "merge_frontmatter": {
    "summary": "A concise description of the note's durable content."
  },
  "if_match_hash": "hash-returned-by-create-note"
}
```

### 7. Verify

```json
{
  "identifiers": ["generated-slug"],
  "fields": ["summary", "tags", "body", "frontmatter", "content_hash", "path"]
}
```

Verify the ID, H1, summary, tags, links, and final hash.

## Editing Protocol

### 1. Locate the Note

Use `search_notes` when the selector is uncertain. Do not edit a guessed title or slug.

### 2. Read Before Writing

```json
{
  "identifiers": ["target-note"],
  "fields": ["body", "summary", "tags", "aliases", "frontmatter", "content_hash", "path"]
}
```

### 3. Choose One Edit Mode

Modes are mutually exclusive:

- `append`
- `replace_body`
- `merge_frontmatter`
- typed `tags` and `aliases`

Use the smallest suitable mode:

- `append` only for material that belongs at the end.
- `replace_body` for insertion into a section or restructuring.
- `merge_frontmatter` for scalar metadata such as `summary` or `type`.
- Typed fields for `tags` and `aliases`.

### 4. Pass the Hash

Always pass the current `content_hash` as `if_match_hash`.

On mismatch:

1. Reread the note.
2. Compare the new version with the planned edit.
3. Rebase the change.
4. Submit with the new hash.

Never bypass a mismatch by omitting the hash.

### 5. Preserve Structure

When replacing the body:

- Keep the H1 stable unless the user requested a title change.
- Preserve unrelated sections and links.
- Avoid duplicate headings.
- Integrate material near the relevant section.
- Update the summary when scope changes materially.

Body and summary changes may require two calls. Use the hash returned by the first edit for the second.

### 6. Verify

Read the note again and confirm the requested content, metadata, links, and final hash.

## Link Management

Before adding a link:

1. Search for the target.
2. Use its canonical slug.
3. Follow the configured style.
4. Use a natural display label when needed.
5. Do not link to an unverified slug.

## Imported or Manual Files

When a file lacks `mnemonic_note_id`, do not invent one through a body edit. Use the project import or sync workflow so mnemonic hydrates metadata consistently.

## Evaluation Example

For "Add information about the empty interface to the note about Go interfaces":

1. Search when the selector is uncertain.
2. Read body and `content_hash`.
3. Integrate the material in the correct section.
4. Call `edit_note` with `replace_body` and `if_match_hash`.
5. Read again to verify.

## Completion Checklist

- [ ] Used a stable title and matching H1 for new notes.
- [ ] Confirmed existing tags when needed.
- [ ] Relied on mnemonic to generate ID and slug.
- [ ] Added a concise summary.
- [ ] Used configured link style and verified targets.
- [ ] Read `content_hash` before every edit.
- [ ] Passed `if_match_hash` with every edit.
- [ ] Preserved unrelated content and verified the result.
