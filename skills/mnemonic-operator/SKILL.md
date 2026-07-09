---
name: mnemonic-operator
description: >-
  Use this skill for diagnostic-driven mnemonic maintenance and recovery: health checks, broken-link repair, duplicate identities, invalid frontmatter, stale or invalid indexes, bulk validation, and safe deletion. Do not use it for ordinary note authoring or routine content edits; use mnemonic-librarian instead.
compatibility: >-
  Requires mnemonic MCP maintenance and note tools. Raw repair of unparseable frontmatter requires filesystem access. The bundled loop additionally requires the mnemonic CLI, Bash, and jq.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic System Operator

Protect the knowledge base first. Diagnose before mutating, use optimistic concurrency, and revalidate every repair batch.

See [the maintenance API reference](references/api-ref.md) for exact diagnostic kinds, mutation contracts, CLI JSON fields, and raw-file fallback rules.

## Diagnostic Model

Current `diagnose_notes` kinds are:

- `invalid_frontmatter`
- `missing_required_field`
- `missing_summary`
- `duplicate_slug`
- `duplicate_alias`
- `unresolved_link`
- `ambiguous_link`
- `empty_body`

Do not use removed legacy kinds such as `missing_timestamp` or `invalid_timestamp`.

## `doctor` Versus `rebuild_index`

Use `doctor` for a broad, read-only assessment of index and content health.

Use `diagnose_notes` for actionable note-level issues and candidate suggestions.

Use `rebuild_index` only when:

- The index is missing, invalid, stale, or structurally incompatible.
- Notes changed outside mnemonic and explicit reconciliation is required.
- A broad health check identifies an index problem after a repair batch.

Do not treat `rebuild_index` as a universal repair. It regenerates a derived artifact; it does not fix malformed notes or broken links.

## Plan-Validate-Execute Cycle

### 1. Plan

Run `doctor` first and classify the problem:

- Index or state problem.
- Metadata or frontmatter problem.
- Link integrity problem.
- Duplicate identity problem.
- Empty or incomplete content problem.

### 2. Validate

Run `diagnose_notes` with only relevant kinds. For broken or ambiguous links, enable suggestions:

```json
{
  "kinds": ["unresolved_link"],
  "include_suggestions": true,
  "limit": 200,
  "cursor": 0
}
```

Paginate until `next_cursor` is absent or zero.

Before proposing a change, inspect:

- The affected note and exact source line.
- Candidate count and candidate quality.
- Current `content_hash` when the note is parseable.
- Backlinks when changing identity or deleting a note.

### 3. Execute

Group the repair plan by issue type and risk.

Apply deterministic repairs when the user authorized repair and the result is unambiguous. Stop for a decision when:

- Several link candidates are plausible.
- A duplicate slug requires a semantic choice between competing identities.
- A frontmatter repair would discard unknown data.
- A change affects many inbound links.
- Permanent deletion is involved.

For every MCP edit:

1. Call `read_notes` with `body`, `frontmatter`, `tags`, `aliases`, `path`, and `content_hash`.
2. Preserve unrelated content and metadata.
3. Call `edit_note` with `if_match_hash`.
4. On hash mismatch, reread and rebase. Never retry with the old hash.

After each repair batch:

1. Rerun the same `diagnose_notes` query.
2. Run `doctor` again.
3. Rebuild only when health output justifies it.
4. Report fixed, skipped, and unresolved issues separately.

## Broken-Link Repair Workflow

1. Diagnose `unresolved_link` with `include_suggestions: true`.
2. Inspect `target`, `source_line`, `link_style`, and `candidates`.
3. If exactly one candidate is strongly supported, read the source and target notes.
4. Replace only the broken target while preserving display label and link style.
5. Edit with `if_match_hash`.
6. Rerun link diagnostics.

Do not select the first candidate merely because it ranks first.

## Duplicate-Slug Workflow

`edit_note.merge_frontmatter` accepts the canonical scalar `slug`; a separate rename tool is not required.

1. Diagnose `duplicate_slug`.
2. Read all colliding notes and inspect IDs, titles, paths, summaries, backlinks, and hashes.
3. Choose the canonical identity based on content and usage, not file order.
4. Choose unique slugs for the other notes.
5. Change one slug at a time using a fresh hash:

```json
{
  "identifier": "note-id-of-collision",
  "merge_frontmatter": {
    "slug": "new-unique-slug"
  },
  "if_match_hash": "fresh-content-hash"
}
```

6. Update inbound links that referred to the changed identity.
7. Revalidate `duplicate_slug`, `unresolved_link`, and `ambiguous_link`.

Do not change `mnemonic_note_id`.

## Invalid-Frontmatter Workflow

An invalid YAML document may be unreadable through `read_notes`. Use two branches.

### Parseable Through MCP

1. Read frontmatter, body, path, and hash.
2. Preserve unknown valid fields.
3. Keep `mnemonic_note_id` stable.
4. Ensure `tags` and `aliases` are lists of strings.
5. Do not reintroduce `permalink`, `created_at`, or `updated_at`.
6. Apply the smallest safe edit with `if_match_hash` and revalidate.

### Unparseable Through MCP

If `read_notes` reports `corrupted` or cannot return the note:

1. Stop MCP mutation for that note.
2. Use filesystem access to inspect the raw Markdown path reported by diagnostics.
3. Create a backup before editing.
4. Repair only YAML syntax or schema defects; preserve the body and existing `mnemonic_note_id` when recoverable.
5. Run `mnemonic project sync PROJECT` or `rebuild_index` as appropriate.
6. Rerun diagnostics and `doctor`.

Do not pretend that MCP can safely rewrite a document it cannot parse.

## Safe Deletion Protocol

Before every `delete_note` call:

1. Call `list_backlinks`.
2. Read `content_hash`, `path`, `summary`, and `body`.
3. If backlinks exist, warn and list affected source notes.
4. Prefer trash deletion.
5. Pass `if_match_hash` for all deletions as a safety policy.
6. Require explicit confirmation for `hard_delete: true`.
7. Rerun link diagnostics after deletion.

```json
{
  "identifier": "obsolete-note",
  "hard_delete": false,
  "if_match_hash": "sha256-from-read-notes"
}
```

## Index Recovery

When search reports a missing or invalid index:

1. Run `doctor`.
2. Confirm source Markdown is readable.
3. Run `rebuild_index` once.
4. Run `doctor` again.
5. If rebuilding fails, report the exact error. Do not rewrite notes unless diagnostics identify a source defect.

## Bundled Repair Loop

Run from the skill root:

```bash
bash scripts/repair-loop.sh PROJECT [MAX_PASSES]
```

The script repeatedly runs CLI diagnostics and delegates repairs to `MNEMONIC_REPAIR_HOOK`. It never selects candidates and never reindexes automatically. The CLI command returns only its default diagnostic page and exposes no cursor option, so one pass processes one returned page. Increase `MAX_PASSES` for large issue sets, or use MCP `diagnose_notes` when explicit pagination is required.

The hook receives:

1. Project selector.
2. Path to the current JSON diagnostic report.

```bash
MNEMONIC_REPAIR_HOOK="$HOME/bin/fix-mnemonic-report" \
  bash scripts/repair-loop.sh personal 5
```

Without a hook, the script copies the report to the current directory and exits without modifying notes.

## Completion Checklist

- [ ] Ran `doctor` before repair.
- [ ] Used current diagnostic kinds only.
- [ ] Enabled suggestions for unresolved or ambiguous links.
- [ ] Read affected notes and fresh hashes before MCP edits.
- [ ] Used raw-file fallback only for unparseable frontmatter.
- [ ] Checked backlinks before deletion or identity changes.
- [ ] Used `if_match_hash` for every MCP mutation.
- [ ] Revalidated after changes.
- [ ] Rebuilt the index only when justified.
