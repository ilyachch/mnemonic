---
name: mnemonic-search-expert
description: >-
  Use this skill when the user wants to find information in a mnemonic knowledge base, recover a note from incomplete clues, search within a time window or confirmed tag category, or explore relationships and backlinks between notes. It builds diverse search variants and expands overly narrow searches when initial results are incomplete.
compatibility: >-
  Requires a configured mnemonic MCP server exposing search_notes, read_notes, list_tags, and list_backlinks.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Search Expert

Use `mnemonic` as the primary source for questions that fall within the active knowledge base.

See [the MCP search reference](references/api-ref.md) for exact parameters and limits.

## Core Rules

1. Search before answering from memory when the requested information may be stored in the project.
2. Prefer one `search_notes` call with several semantically distinct query variants over many nearly identical calls.
3. Respect the hard limits:
   - At most 8 entries in `queries`.
   - At most 500 Unicode characters per query.
   - At most 100 results.
4. Treat `tags` as an AND filter. Every listed tag must be present on a result.
5. Do not infer a mnemonic tag from a topic name. A request about Go does not prove that a `go` tag exists.
6. Batch-read selected results in one `read_notes` call.
7. Inspect backlinks when the user asks for context, relationships, usage, later decisions, contradictions, or superseding information. Backlinks are optional for a simple note lookup.
8. Distinguish absence of evidence from evidence of absence. Report the search scope and filters that may have excluded material.

## Reciprocal Rank Fusion Strategy

`search_notes` combines rankings from all query variants using Reciprocal Rank Fusion. Query diversity matters more than superficial rewording.

Build 3-5 variants for ambiguous or incomplete requests. Use up to 8 only when the topic has several distinct names or technical vocabularies.

A useful variant set can include:

- The user's original wording.
- A concise concept-focused formulation.
- An implementation or domain-specific formulation.
- A likely synonym, abbreviation, or older term.
- A decision, incident, or outcome-oriented formulation.

Avoid variants that differ only by word order, punctuation, or filler words.

Example:

```json
{
  "queries": [
    "Go architecture",
    "Golang project structure",
    "ports adapters hexagonal architecture Go",
    "Go package boundaries dependency direction"
  ]
}
```

## Search Workflow

### 1. Extract Constraints

Identify:

- Subject and likely terminology.
- Explicit time window.
- Explicit or already confirmed tags.
- Whether the user wants direct matches or surrounding context.
- Whether incomplete clues require broader recall.

Do not invent tags or time windows. Derive relative durations only from the user's request.

### 2. Construct the Initial Search

Use:

- 3-5 semantically distinct `queries` for ambiguous requests.
- `tags` only when the user explicitly refers to a known mnemonic tag or `list_tags` confirms it.
- `created_since` for notes created within a period.
- `updated_since` for notes written or modified within a period.
- `include_related: true` when graph context is central.
- A practical initial `limit`, normally 10-30.

For "everything I wrote about Go architecture during the last week", start without an unconfirmed tag:

```json
{
  "queries": [
    "Go architecture",
    "Golang project structure",
    "ports adapters hexagonal architecture Go"
  ],
  "updated_since": "7d",
  "limit": 30
}
```

If a tag filter would materially improve precision, call `list_tags`. Only after confirming `go` may a later search add:

```json
{
  "tags": ["go"]
}
```

### 3. Inspect Results

Use `matched_queries`, snippets, summaries, tags, and related-note metadata to select the strongest candidates.

Batch-read selected notes in one call. Request only the fields needed for the task, typically:

```json
{
  "identifiers": ["first-slug", "second-slug"],
  "fields": ["summary", "tags", "body", "path", "updated_at"]
}
```

### 4. Inspect Relationships When Relevant

Call `list_backlinks` when the task involves:

- Recovering surrounding context.
- Finding where a decision or fact is used.
- Tracing later implementation or follow-up work.
- Resolving conflicts or possible supersession.
- Exploring relationships between notes.

Example:

```json
{
  "identifier": "relevant-note-slug",
  "limit": 50
}
```

Read backlink sources before using them as evidence. Do not treat titles or graph edges alone as proof.

### 5. Synthesize With Provenance

State:

- What was found.
- Which notes support the conclusion.
- Relevant dates or update windows.
- Important relationships when investigated.
- Remaining uncertainty or excluded scope.

## Expanding Search Algorithm

Apply this sequence when the precise search returns no useful results:

1. Remove tag filters and rerun the same diverse queries.
2. Preserve the user's `updated_since` or other time window. Do not invent a narrower period.
3. Call `list_tags` to discover the project's vocabulary.
4. Replace unconfirmed tags with confirmed alternatives and rerun only when helpful.
5. Broaden terminology by removing overly specific phrases and adding synonyms or implementation terms.
6. If the index appears missing or invalid, stop retrying and hand off to `mnemonic-operator`.

The sequence broadens categorical constraints while preserving temporal intent.

## Incomplete-Clue Recovery

When the user remembers only fragments:

1. Create variants for each remembered fragment.
2. Include exact phrases and conceptual alternatives.
3. Search without tags first unless a tag is already confirmed.
4. Use a wider result limit.
5. Inspect `matched_queries` to understand why results surfaced.
6. Read candidates in one batch.
7. Follow backlinks from the strongest candidate when surrounding context is part of the request.

## Edge Cases

### No Results After Expansion

Report the queries, time window, and tag changes attempted. Suggest checking project selection or index health.

### Too Many Results

Narrow with a confirmed tag, an explicit time filter, or more discriminating query variants. Do not add arbitrary filters.

### Conflicting Notes

Read all conflicting candidates. Inspect backlinks and update times when they may reveal later decisions. Prefer a later explicit decision over older exploration, but label this as an interpretation unless a note states that it supersedes another.

### Read-Only Mode

Search, read, tag listing, and backlink inspection remain available. Do not attempt mutations.

## Completion Checklist

- [ ] Used `search_notes` before answering within project scope.
- [ ] Generated semantically distinct query variants.
- [ ] Stayed within 8 queries and 500 characters per query.
- [ ] Applied only justified time and confirmed tag filters.
- [ ] Batch-read selected notes.
- [ ] Inspected backlinks only when context or relationships required them.
- [ ] Explained uncertainty and search scope.
