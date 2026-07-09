---
name: mnemonic-operator
description: Diagnoses and repairs mnemonic knowledge-base integrity problems, including broken links, invalid frontmatter, duplicate slugs, stale or invalid indexes, and unsafe deletions. Use for health checks, repository maintenance, bulk validation, repair planning, and index recovery.
compatibility: Requires mnemonic MCP tools. The bundled repair loop additionally requires the mnemonic CLI, Bash, and jq.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic System Operator

Protect the knowledge base first. Diagnose before mutating, use optimistic concurrency for edits, and revalidate after every repair batch.

See [the maintenance API reference](references/api-ref.md) for the exact diagnostic kinds and tool contracts.

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

The following three are especially important for this skill:

- `unresolved_link`: a link target cannot be resolved.
- `duplicate_slug`: more than one note uses the same canonical slug.
- `invalid_frontmatter`: YAML frontmatter cannot be parsed or violates the note schema.

Do not use removed legacy kinds such as `missing_timestamp` or `invalid_timestamp`.

## `doctor` Versus `rebuild_index`

Use `doctor` for a broad, read-only health assessment of index and content state.

Use `diagnose_notes` for actionable, note-level issues with paths, fields, source lines, link targets, and optional candidate suggestions.

Use `rebuild_index` only when:

- The index is missing, invalid, stale, or structurally incompatible.
- Notes were changed outside mnemonic and the index must be regenerated.
- A completed repair batch requires explicit index reconciliation.

Do not treat `rebuild_index` as a universal repair. It regenerates a derived artifact; it does not fix malformed notes or broken links.

## Plan-Validate-Execute Cycle

### 1. Plan

Run `doctor` first.

Classify the failure:

- Index or state problem.
- Metadata problem.
- Link integrity problem.
- Duplicate identity problem.
- Empty or incomplete content problem.

### 2. Validate

Run `diagnose_notes` with only the relevant kinds. For broken links, always enable suggestions:

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

- The affected note.
- The exact source line and link target.
- Candidate count and candidate quality.
- Whether the candidate is uniquely determined.
- The note's current `content_hash`.
- Backlinks when changing identity, slug, or deleting a note.

### 3. Execute

Create a repair plan grouped by issue type and risk.

Apply deterministic repairs when the user's request already authorizes repair and there is one unambiguous result. Ask for a decision when:

- Several link candidates are plausible.
- A duplicate slug requires choosing a canonical note.
- A frontmatter repair would discard unknown fields.
- A change would alter many inbound links.
- A hard deletion or wipe is involved.

For every edit:

1. Call `read_notes` with `body`, `frontmatter`, `tags`, `aliases`, `path`, and `content_hash`.
2. Preserve unrelated content and metadata.
3. Call `edit_note` with `if_match_hash`.
4. If the hash mismatches, reread the note and rebase the planned change. Never retry with the old hash.

After the repair batch:

1. Rerun the same `diagnose_notes` query.
2. Run `doctor` again.
3. Rebuild the index only if health output indicates that it is needed.
4. Report fixed, skipped, and unresolved issues separately.

## Broken-Link Repair Workflow

1. Run `diagnose_notes` with `kinds: ["unresolved_link"]` and `include_suggestions: true`.
2. For each issue, inspect `target`, `source_line`, `link_style`, and `candidates`.
3. If exactly one candidate is strongly supported, read both the source and target note.
4. Replace only the broken link target while preserving its display label and link style.
5. Edit with `if_match_hash`.
6. Rerun diagnostics after each batch.

Do not auto-select the first candidate merely because it ranks first.

## Duplicate-Slug Workflow

1. Diagnose `duplicate_slug`.
2. Read all colliding notes and inspect their IDs, titles, paths, summaries, backlinks, and content hashes.
3. Select a canonical slug based on content identity and usage, not file order.
4. Propose new unique slugs for the remaining notes.
5. Update affected inbound links before or in the same controlled repair batch.
6. Revalidate both `duplicate_slug` and link diagnostics.

## Invalid-Frontmatter Workflow

1. Read the raw frontmatter and body.
2. Preserve unknown valid fields where possible.
3. Ensure `mnemonic_note_id` exists and remains stable.
4. Ensure `tags` and `aliases` are YAML lists of strings, not scalars.
5. Do not reintroduce legacy fields stripped by current rendering, including `permalink`, `created_at`, or `updated_at`.
6. Apply the smallest safe correction and revalidate.

## Safe Deletion Protocol

Before every `delete_note` call:

1. Call `list_backlinks` for the note.
2. Call `read_notes` and request `content_hash`, `path`, `summary`, and `body`.
3. If backlinks exist, warn that deletion will break relationships and list the affected source notes.
4. Prefer trash deletion over hard deletion.
5. Pass `if_match_hash` even for trash deletion as a safety policy.
6. Require explicit confirmation before `hard_delete: true`.
7. After deletion, rerun link diagnostics.

Example safe deletion input:

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
2. Confirm that the source Markdown files are readable.
3. Run `rebuild_index` once.
4. Run `doctor` again.
5. If rebuilding fails, report the exact error and do not modify source notes merely to silence an index error unless diagnostics identify a source-note defect.

## Bundled Repair Loop

Run:

```bash
scripts/repair-loop.sh PROJECT [MAX_PASSES]
```

The script repeatedly runs CLI diagnostics and delegates actual repairs to a hook specified by `MNEMONIC_REPAIR_HOOK`. This avoids unsafe automatic candidate selection while still automating diagnose-repair-validate cycles.

The hook receives two arguments:

1. Project selector.
2. Path to the current JSON diagnostic report.

Example:

```bash
MNEMONIC_REPAIR_HOOK="$HOME/bin/fix-mnemonic-report" \
  scripts/repair-loop.sh personal 5
```

Without a hook, the script produces a report and exits without modifying notes.

## Completion Checklist

- [ ] Ran `doctor` before repair.
- [ ] Used current diagnostic kinds only.
- [ ] Enabled suggestions for unresolved or ambiguous links.
- [ ] Read affected notes and hashes before edits.
- [ ] Checked backlinks before deletion or identity changes.
- [ ] Used `if_match_hash` for every mutation.
- [ ] Revalidated diagnostics after changes.
- [ ] Rebuilt the index only when justified.
