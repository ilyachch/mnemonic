---
name: mnemonic-search-expert
description: Searches mnemonic knowledge bases with FTS5, Reciprocal Rank Fusion, time and tag filters, and graph relationships. Use when the user needs to find notes, recover context from incomplete clues, investigate related information, or explore how notes are referenced.
compatibility: Requires a configured mnemonic MCP server exposing search_notes, read_notes, list_tags, and list_backlinks.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Search Expert

Use `mnemonic` as the primary source for questions that fall within the active knowledge base.

## Core Rules

1. Search before answering from memory when the requested information may be stored in the project.
2. Prefer one `search_notes` call with several semantically distinct query variants over many nearly identical calls.
3. Respect the hard search limits:
   - At most 8 entries in `queries`.
   - At most 500 Unicode characters per query.
   - At most 100 results.
4. Treat `tags` as an AND filter. Every listed tag must be present on a result.
5. Batch-read selected results in one `read_notes` call.
6. After identifying a relevant note, call `list_backlinks` for that note before presenting the final synthesis. This is mandatory because backlinks reveal where the information is used, challenged, extended, or superseded.
7. Distinguish absence of evidence from evidence of absence. Report the search scope and any filters that may have excluded material.

See [the MCP search reference](references/api-ref.md) for exact parameters and limits.

## Reciprocal Rank Fusion Strategy

`search_notes` combines rankings from all query variants using Reciprocal Rank Fusion. Query diversity matters more than superficial rewording.

Build 3-5 variants for ambiguous or incomplete requests. Use up to 8 only when the topic has several distinct names or technical vocabularies.

A useful variant set usually includes:

- The user's original wording.
- A concise concept-focused formulation.
- An implementation or domain-specific formulation.
- A likely synonym, abbreviation, or older term.
- A decision, incident, or outcome-oriented formulation when relevant.

Avoid variants that differ only by word order, punctuation, or filler words.

Example for "Go architecture":

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
- Explicit tags.
- Whether the user wants direct matches only or surrounding context.
- Whether incomplete clues require broader recall.

Do not invent tags or time windows. Derive relative durations only from the user's request.

### 2. Construct the Initial Search

Use:

- 3-5 semantically distinct `queries` for ambiguous requests.
- `tags` only when the tag is explicit or already confirmed by `list_tags`.
- `created_since` for notes created within a period.
- `updated_since` for notes written or modified within a period.
- `include_related: true` when graph context is central to the request.
- A practical initial `limit`, normally 10-30.

For "everything I wrote about Go architecture during the last week", call:

```json
{
  "queries": [
    "Go architecture",
    "Golang project structure",
    "ports adapters hexagonal architecture Go"
  ],
  "tags": ["go"],
  "updated_since": "7d",
  "limit": 30,
  "include_related": true
}
```

### 3. Inspect Results

Use `matched_queries`, snippets, summaries, tags, and related notes to select the strongest candidates.

Batch-read all selected notes in one call. Request only the fields needed for the task, typically:

```json
{
  "identifiers": ["first-slug", "second-slug"],
  "fields": ["summary", "tags", "body", "path", "updated_at"]
}
```

### 4. Inspect Backlinks

For every note that materially supports the answer, call:

```json
{
  "identifier": "relevant-note-slug",
  "limit": 50
}
```

Read backlink sources when their titles or context suggest contradiction, later decisions, implementation details, or superseding information.

### 5. Synthesize With Provenance

State:

- What was found.
- Which notes support the conclusion.
- Relevant dates or update windows.
- Important relationships or backlinks.
- Remaining uncertainty or excluded scope.

Do not present related notes as direct evidence until their content has been read.

## Expanding Search Algorithm

Apply this sequence when the precise search returns no useful results:

1. Remove tag filters and rerun the same diverse queries. A missing or inconsistent tag is a common cause of false negatives.
2. Preserve or add `updated_since` when the user supplied a recency requirement. Use the user's actual window, such as `7d`; do not invent a narrower period.
3. Call `list_tags` to discover the vocabulary used by the knowledge base.
4. Replace unconfirmed tags with plausible existing tags and rerun the search.
5. Broaden terminology by removing overly specific phrases and adding synonyms or implementation terms.
6. If the index appears missing or invalid, stop searching and recommend the operator workflow rather than repeatedly retrying.

The expansion sequence broadens categorical constraints while preserving the user's temporal intent.

## Incomplete-Clue Recovery

When the user remembers only fragments:

1. Create variants for each remembered fragment.
2. Include both exact phrases and conceptual alternatives.
3. Search without tags first unless a tag is certain.
4. Use a wider result limit.
5. Inspect `matched_queries` to understand why each result surfaced.
6. Read candidates in a single batch.
7. Follow backlinks from the strongest candidate to reconstruct the surrounding discussion.

## Edge Cases

### No Results After Expansion

Report the queries, time window, and tag changes attempted. Suggest checking project selection or index health.

### Too Many Results

Narrow with a confirmed tag, an explicit time filter, or more discriminating query variants. Do not add arbitrary filters.

### Conflicting Notes

Read all conflicting candidates and their backlinks. Prefer later explicit decisions over older exploratory notes, but label this as an interpretation unless a note states that it supersedes another.

### Read-Only Mode

Search, read, tag listing, backlink inspection, and diagnostics remain available. Do not attempt mutations.

## Completion Checklist

- [ ] Used `search_notes` before answering.
- [ ] Generated semantically distinct query variants.
- [ ] Stayed within 8 queries and 500 characters per query.
- [ ] Applied only justified time and tag filters.
- [ ] Batch-read selected notes.
- [ ] Called `list_backlinks` for every materially relevant note.
- [ ] Explained uncertainty and search scope.
