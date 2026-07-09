---
name: mnemonic-librarian
description: Creates and edits high-quality mnemonic notes with canonical Markdown structure, YAML frontmatter, generated slugs, consistent tags, summaries, links, and optimistic concurrency. Use whenever adding knowledge, updating existing notes, restructuring content, or preserving note-format integrity.
compatibility: Requires a mnemonic MCP server exposing search_notes, read_notes, list_tags, create_note, and edit_note, with note mutations enabled.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Content Librarian

Create readable, connected, and safely versioned notes. Preserve user content, minimize unrelated changes, and use the current note hash for every edit.

See [the note API reference](references/api-ref.md) for exact tool behavior.

## Canonical Note Principles

- `mnemonic_note_id` is the only strictly required frontmatter field.
- `create_note` generates `mnemonic_note_id` automatically. Never invent an ID or place YAML frontmatter inside the body merely to simulate it.
- Mnemonic stores the supplied title and derives the slug from it.
- Start newly authored body content with one H1 heading matching the note title.
- Add a concise summary to frontmatter after creation.
- Prefer wiki-links in projects configured with wiki link style:

```markdown
[[target-slug|Display Label]]
```

- When the project specifies regular links, use:

```markdown
[Display Label](target-slug.md)
```

- `tags` and `aliases` must be lists of strings.
- Do not write `created_at` or `updated_at`; current mnemonic resolves timestamps from the filesystem and strips legacy timestamp fields on render.
- Do not use the legacy `permalink` field.
- If the configured link style is unavailable, inspect project instructions.
- Do not guess a link style; omit new links or ask for the project configuration.

## Creation Checklist

### 1. Choose the Title and Expected Slug

Use a title that is:

- Specific enough to be unique.
- Stable over time.
- Focused on one primary concept or decision.

Anticipate the generated slug, but do not fabricate a separate slug unless the user explicitly requires one and the current tool contract supports the change safely.

### 2. Discover Existing Tags

Call `list_tags` before assigning tags unless the user supplied a complete authoritative set.

Prefer existing vocabulary. Add a new tag only when no existing tag accurately represents the note.

Keep tags categorical and reusable. Do not turn sentences into tags.

### 3. Prepare the Body

Use this default structure when it fits the content:

```markdown
# Note Title

A concise opening paragraph that states the subject.

## Context

Why this information matters.

## Details

The durable facts, reasoning, examples, or implementation notes.

## Related Notes

- [[related-slug|Related Note]]
```

Do not force empty sections. The H1 is mandatory for newly authored notes; lower sections depend on content.

### 4. Write a Summary

Create a one- or two-sentence summary that explains the note's durable value. Avoid vague summaries such as "Notes about Go".

### 5. Create the Note

Call `create_note` with:

- `title`
- Body beginning with the H1.
- Selected tags.

The result supplies `note_id`, generated `slug`, `path`, and `content_hash`.

### 6. Add the Summary Safely

Immediately call `edit_note` in `merge_frontmatter` mode using the hash returned by `create_note`:

```json
{
  "identifier": "generated-slug",
  "merge_frontmatter": {
    "summary": "A concise description of the note's durable content."
  },
  "if_match_hash": "hash-returned-by-create-note"
}
```

This ensures the final note has a summary without manually constructing frontmatter.

### 7. Verify

Call `read_notes` with:

```json
{
  "identifiers": ["generated-slug"],
  "fields": ["summary", "tags", "body", "frontmatter", "content_hash", "path"]
}
```

Verify:

- `mnemonic_note_id` exists.
- H1 matches the title.
- Summary is present and accurate.
- Tags are lists and use project vocabulary.
- Links follow the configured style.

## Editing Protocol

### 1. Locate the Note

Use `search_notes` when the selector is not certain. Do not edit a guessed title or slug.

### 2. Read Before Writing

Call `read_notes` with at least:

```json
{
  "identifiers": ["target-note"],
  "fields": ["body", "summary", "tags", "aliases", "frontmatter", "content_hash", "path"]
}
```

### 3. Choose One Edit Mode

`edit_note` modes are mutually exclusive:

- `append`
- `replace_body`
- `merge_frontmatter`
- typed `tags` and `aliases`

Use the smallest mode that satisfies the request.

- Use `append` only when the new material belongs at the end and does not require restructuring.
- Use `replace_body` when inserting content into a specific section or reorganizing the note.
- Use `merge_frontmatter` for scalar metadata such as `summary` or `type`.
- Use typed `tags` and `aliases` fields for those lists.

### 4. Pass the Hash

Always pass the current `content_hash` as `if_match_hash`, even when the server does not strictly require it for that mode.

If a hash mismatch occurs:

1. Reread the note.
2. Compare the new version with the planned edit.
3. Reapply the change to the new content.
4. Submit with the new hash.

Never bypass a mismatch by omitting the hash.

### 5. Preserve Structure

When replacing the body:

- Keep the H1 stable unless the user requested a rename.
- Preserve unrelated sections and links.
- Avoid duplicate headings.
- Integrate new material near the relevant section.
- Update the summary when the note's scope changes materially.

Because edit modes are mutually exclusive, body and summary changes may require two sequential edits. Use the hash returned by the first edit for the second.

### 6. Verify the Result

Read the note again and confirm the requested change, metadata, links, and hash.

## Link Management

Before adding a link:

1. Search for the target note.
2. Use the target's canonical slug.
3. Use a display label when the sentence needs natural language.
4. Avoid links to unverified slugs.

Preferred wiki form:

```markdown
Go's empty interface is discussed in [[go-empty-interface|Empty Interface in Go]].
```

## Imported or Manually Created Files

When a Markdown file does not have `mnemonic_note_id`, do not invent one through a normal body edit. Use the project import or sync workflow so mnemonic hydrates metadata consistently.

## Evaluation Example

For "Add information about the empty interface to the note about Go interfaces":

1. Search for the Go interfaces note if its selector is uncertain.
2. Call `read_notes` and request body plus `content_hash`.
3. Integrate the new section into the correct location.
4. Call `edit_note` with `replace_body` and `if_match_hash`.
5. Read the note again to verify the result.

## Completion Checklist

- [ ] Used a stable title and matching H1.
- [ ] Confirmed existing tags with `list_tags`.
- [ ] Relied on mnemonic to generate `mnemonic_note_id` and slug.
- [ ] Added a concise summary.
- [ ] Used canonical link style and verified target slugs.
- [ ] Read `content_hash` before every edit.
- [ ] Passed `if_match_hash` with every edit.
- [ ] Preserved unrelated content and verified the final note.
