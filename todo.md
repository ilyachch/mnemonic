# Remaining Tasks `fixes-and-improvements`

## General Rules

- Do not add migrations.
- Do not support old contracts and formats.
- Do not add compatibility wrappers.
- Do not distinguish between specific old versions of data.
- Treat the SQLite index as a derived artifact that can be completely recreated from scratch.
- Do not classify errors by parsing their message text.
- Do not use direct `time.Now()` calls if a clock abstraction is available in the project.

---

# MNEMONIC-201 — Remove Version-Based SQLite Index Upgrades

## Objective

Remove the mechanism that detects legacy index versions and automatically adapts existing installations.

## Context

Currently, the schema version is incremented to `3`, and `CheckSchemaStatus` returns `NeedsRebuild` if any other version number is detected.

This establishes an upgrade path for previously created indexes, even though the project does not require backward compatibility.

## Requirements

Remove all usage of:

```sql
PRAGMA user_version

```

Remove the following record:

```text
meta.schema_version

```

Delete:

- comments such as `schema version 3`;
- comparisons of the current version with previous versions;
- the `version != N → rebuild` logic;
- phrases regarding "upgrading existing installations."

`ApplySchema` must only create the current schema in a new, empty database.

`reindex` must perform the following steps:

1. create a new temporary database;
2. apply the current schema;
3. fully populate the index;
4. atomically replace the old file.

The application must not attempt to convert an existing index.

If an incompatible or corrupted index is encountered, it is permitted to return a generic error:

```text
index is invalid; rebuild it

```

However, you must not determine exactly which old version was detected or execute separate logic for it.

## Acceptance Criteria

- `PRAGMA user_version` is completely absent from the code.
- `schema_version` is no longer stored in `meta`.
- There are no numeric SQLite schema versions in the code.
- No code branches exist for legacy schemas or upgrades.
- `reindex` always builds the index from scratch.
- The new index includes the `relation_type` field.
- A corrupted or incompatible index is never migrated automatically.

## Key Files

- `internal/store/sqliteindex/schema.go`
- `internal/store/sqliteindex/rebuild.go`
- Call sites of `CheckSchemaStatus`
- Documentation describing the rebuild process

---

# MNEMONIC-202 — Complete Logger Dependency Injection

## Objective

Pass the configured logger to all runtime paths without passing `nil` and without mutating services after instantiation.

## Issues

The Web adapter currently creates the stdio/MCP server as follows:

```go
stdio.NewServer(..., input.ReadOnly, nil)

```

`catalogsvc` initializes the index service with a `nil` logger during:

- `project add`;
- `project init`;
- import/reindex operations after import.

## Requirements

Add the logger to the input web server struct:

```go
type ServerInput struct {
    // existing fields
    Logger *slog.Logger
}

```

Pass it to:

```go
stdio.NewServer(...)

```

When instantiating `indexsvc.Service` inside `catalogsvc`, use `s.Logger`:

```go
indexsvc.New(resolved, s.Logger)

```

The configured logger must be passed via constructors to:

- CLI runtime;
- stdio MCP;
- web MCP;
- maintenance runtime;
- catalog operations;
- notes service;
- search service;
- index service.

Do not assign the logger by modifying public fields after a service has been created.

## Acceptance Criteria

- The Web MCP receives the configured logger.
- `project add`, `project init`, and import operations utilize the configured logger.
- The runtime code contains no `New(..., nil)` calls if the logger is available in the calling service.
- Stdio logs do not leak into stdout.
- The logger is set exclusively during object construction.

## Key Files

- `internal/adapter/web/manager.go`
- Web server startup command
- `internal/service/catalogsvc/service.go`
- `internal/app/app.go`
- `internal/app/runtime.go`

---

# MNEMONIC-203 — Remove String-Based Error Classification in `ShowMany`

## Objective

Make batch-read error classification resilient to modifications in error message text.

## Issue

The application currently relies on string matching checks:

```go
strings.Contains(msg, "parse note")
strings.Contains(msg, "parse frontmatter")
strings.Contains(msg, "read note")

```

Altering the error phrasing will break the external contract of `read_notes`.

## Requirements

`markdownstore.Show()` must return a typed `apperr.Error` for all expected errors.

Minimum classification mapping:

```text
CodeNotFound   → missing
CodeAmbiguous  → issue kind "ambiguous"
CodeCorrupted  → issue kind "corrupted"
CodeIO         → issue kind "io_error"
all others     → issue kind "internal"

```

If a dedicated `CodeIO` does not currently exist, add it or use an alternative existing code with unambiguous semantics.

Inside `notesvc.classifyShowError`, you are strictly allowed to use only:

```go
errors.As
errors.Is

```

Do not use:

- `strings.Contains`;
- direct comparisons with `err.Error()`;
- assumptions regarding the error message text.

## Acceptance Criteria

- Not found errors map exclusively to `missing`.
- Ambiguous selectors map to `issues` as `ambiguous`.
- Parsing errors map to `issues` as `corrupted`.
- Read or permission errors map to `issues` as `io_error`.
- Changing the message string does not alter the error classification.
- `classifyShowError` contains no string analysis logic.

## Key Files

- `internal/store/markdownstore/store.go`
- `internal/service/notesvc/service.go`
- `internal/apperr/*`

---

# MNEMONIC-204 — Add Duration to Batch-Read Logging

## Objective

Complete operational logging for `read_notes`.

## Requirements

At the beginning of `ShowMany`, capture the current time using the project's clock abstraction.

After execution, record a single aggregated event:

```text
batch read completed
requested_count
found_count
missing_count
issue_count
duration

```

Do not log:

- note bodies;
- frontmatter;
- the full text of the notes.

Do not trigger a separate info event for each selector.

Individual selector errors may be logged at the debug level.

## Acceptance Criteria

- The aggregated event includes the `duration` field.
- The duration is computed using the project clock.
- A single batch-read creates exactly one aggregated operational event.
- Note contents do not appear in the log.

## Key Files

- `internal/service/notesvc/service.go`
- The project's clock package

---

# MNEMONIC-205 — Remove Direct `time.Now()` Calls from Diagnostics Suggestions

## Objective

Use a single source of time to ensure deterministic service behavior.

## Issue

`searchCandidatesByTarget` currently invokes:

```go
now := time.Now()

```

## Requirements

Pass the clock into `indexsvc.Service` or utilize the existing project clock.

Target interface:

```go
type Clock interface {
    Now() time.Time
}

```

`indexsvc.New` must receive the clock via dependency injection or use the shared project-wide `clock.NowUTC()`.

The time must always be in UTC.

You must not call:

```go
time.Now()

```

directly within the diagnostics service.

## Acceptance Criteria

- Diagnostics contains no direct `time.Now()` calls.
- Candidate search retrieves time from the injected clock.
- The production runtime utilizes the real clock.
- Behavior can be reliably reproduced using a fixed time stub.

## Key Files

- `internal/service/indexsvc/service.go`
- `internal/service/indexsvc/diagnostics.go`
- `internal/app/runtime.go`
- The project's clock package

---

# MNEMONIC-206 — Enforce `tags` and `aliases` Strictly as Lists

## Objective

Maintain a single canonical frontmatter format and drop support for alternative legacy syntax styles.

## Issue

Currently, a single string value is accepted:

```yaml
tags: payment
```

and automatically converted into:

```go
[]string{"payment"}

```

## Requirements

The following fields:

```text
tags
aliases

```

must exclusively accept a YAML list of strings:

```yaml
tags:
  - payment
  - spei
```

The following variations must be treated as invalid:

```yaml
tags: payment
aliases: "old title"
tags: 123
tags:
  - payment
  - 123
```

Upon encountering an error, return a `FrontmatterFieldError`:

```text
Kind = invalid_string_list
Field = tags | aliases

```

Do not perform implicit scalar-to-list conversions.

## Acceptance Criteria

- A list of strings parses successfully.
- A single string scalar is rejected.
- A numeric value is rejected.
- A list containing non-string elements is rejected.
- The renderer always outputs a list format.
- Diagnostics correctly returns the field name.

## Key Files

- `internal/format/markdown/note.go`
- `internal/format/markdown/render.go`
- `internal/format/markdown/errors.go`

---

# MNEMONIC-207 — Centralize Input Validation Limits

## Objective

Enforce identical input validation rules across the MCP, CLI, and service layers.

## Issues

Currently, the majority of input constraints are located solely within the stdio adapter.

The following are not fully handled:

- negative values;
- the maximum number of diagnostic kinds;
- Unicode query length;
- excessive CLI limits.

## Requirements

Extract validation logic into the service layer or a shared package.

### Search

Validate that:

```text
1 <= limit <= 100
queries count <= 8
query length <= 500 Unicode characters

```

If `limit` is missing, apply a default value of `10`.

Compute length using:

```go
utf8.RuneCountInString

```

### Read Notes

Validate that:

```text
identifiers count: 1..50
max_body_chars: 0..100000

```

A negative `max_body_chars` value must return an error.

### Diagnostics

Validate that:

```text
limit: 1..200
cursor >= 0
number of kinds does not exceed the count of supported kinds

```

An unrecognized kind must trigger a validation error.

### General Regulations

- Do not silently truncate values.
- The MCP and CLI must receive the same error structure.
- The adapter may validate inputs early, but the service layer remains the single source of truth.

## Acceptance Criteria

- The CLI cannot bypass MCP layer constraints.
- A Russian language query of 500 characters is accepted.
- A Russian language query of 501 characters is rejected.
- A negative `max_body_chars` value is rejected.
- A negative `cursor` value is rejected.
- An unknown diagnostic kind is rejected.
- Errors expose a stable application error code.

## Key Files

- `internal/service/searchsvc/service.go`
- `internal/service/notesvc/service.go`
- `internal/service/indexsvc/service.go`
- `internal/adapter/stdio/tools.go`
- `internal/adapter/cli/notes_search.go`
- `internal/adapter/cli/project_doctor.go`

---

# MNEMONIC-208 — Execute Candidate Suggestions in a Single Batch Query

## Objective

Avoid executing a separate SQLite search operation for each unique broken-link target.

## Issue

Targets are already deduplicated and the database is opened once, but a separate call is made for every target:

```go
SearchAdvanced(...)

```

## Requirements

Add a store-level method:

```go
SearchCandidatesByTargets(
    db *sql.DB,
    targets []string,
    limitPerTarget int,
    now time.Time,
) (map[string][]SearchResult, error)

```

This method must:

- accept all unique targets;
- run a single composite SQL query or a bounded, fixed number of queries;
- return a maximum of 3 candidates per target;
- preserve deterministic ordering;
- skip related-notes expansion;
- skip graph reranking if it is unnecessary for suggestions.

`indexsvc` must invoke this method exactly once.

Remove the loop that runs an individual `SearchAdvanced` call per target.

## Acceptance Criteria

- The number of SQL search operations does not grow linearly with the count of targets.
- No more than 3 candidates are returned for any single target.
- The same target is looked up only once.
- A failure in suggestions does not suppress the core diagnostics output.
- Candidate ordering remains stable.

## Key Files

- `internal/store/sqliteindex/store.go`
- `internal/service/indexsvc/diagnostics.go`

---

# MNEMONIC-209 — Return Original `matched_queries`

## Objective

Expose original query variants to the agent instead of internal FTS syntax.

## Issue

`dedupQueries` currently returns values modified by `sanitizeFTSQuery`, which flow directly into:

```json
matched_queries

```

## Requirements

Introduce a distinct structure:

```go
type normalizedQuery struct {
    Original string
    FTS      string
}

```

Rules:

1. `Original` — the trimmed, initial user query string.
2. `FTS` — the result after sanitization.
3. Deduplication is performed based on the `FTS` value.
4. If multiple original queries yield the same FTS query, preserve the first one.
5. Pass `FTS` to SQLite.
6. Return `Original` in the `matched_queries` output.

Do not expose internal escaping patterns or FTS operators appended by the application.

## Acceptance Criteria

Input:

```json
{
  "queries": ["chargeback process", " chargeback process "]
}
```

triggers a single FTS search and returns:

```json
{
  "matched_queries": ["chargeback process"]
}
```

`matched_queries` contains no internal sanitized syntax elements.

## Key Files

- `internal/store/sqliteindex/store.go`
- `internal/service/searchsvc/service.go`

---

# MNEMONIC-210 — Unify Timestamp Contract in MCP

## Objective

Enforce a single format across all timestamp fields sharing the same semantic name.

## Issue

Frontmatter, the index database, and search filters all utilize Unix seconds, whereas `read_notes` returns RFC3339 formatted strings.

## Requirements

Modify the DTO fields:

```go
CreatedAt *int64 `json:"created_at,omitempty"`
UpdatedAt *int64 `json:"updated_at,omitempty"`

```

Populate them using:

```go
createdAt := resolved.Note.CreatedAt.Unix()
updatedAt := resolved.Note.UpdatedAt.Unix()

```

Stop formatting dates via RFC3339 strings.

Identically named fields across all external contracts must uniformly represent Unix timestamp seconds.

## Acceptance Criteria

`read_notes` returns:

```json
{
  "created_at": 1783024200,
  "updated_at": 1783025200
}
```

The values are serialized as JSON numbers.

The tool description contains no references to RFC3339.

## Key Files

- `internal/adapter/stdio/tools.go`
- CLI note output formatting (if it outputs identical fields)
- User-facing documentation

---

# MNEMONIC-211 — Finalize Legacy Search API Removal

## Objective

Remove the residual legacy single-query API from the SQLite store.

## Context

`searchsvc.Search` has been removed, but the `sqliteindex.Store` still retains the old method:

```go
Search(db, query, limit, tag)

```

## Requirements

Locate all call sites of `sqliteindex.Store.Search`.

If zero call sites remain:

- remove the method;
- remove associated private helpers used exclusively by it;
- drop the obsolete payload schemas and comments.

If any call sites remain:

- refactor them to use `SearchAdvanced`;
- then safely remove the old method.

Do not leave compatibility wrappers for the legacy API behind.

## Acceptance Criteria

- The single-query `Search` method is absent from `sqliteindex.Store`.
- All search call sites interact via `SearchAdvanced` or modern specialized methods.
- The codebase is free of the term `legacy search`.
- No adapters for the old search contract exist.

## Key Files

- `internal/store/sqliteindex/store.go`
- All call sites identified via code search

---

# MNEMONIC-212 — Update Documentation for Active Contracts

## Objective

Synchronize all project documentation with the actual implementation details.

## Requirements

Update:

- `README.md`;
- `PROMPTS.md`;
- `ARCHITECTURE.md`;
- `AGENTS.md`;
- CLI reference manuals;
- Markdown format specifications.

The documentation must accurately describe:

- `search_notes.queries`;
- RRF and graph-aware reranking;
- the default limit value of 10;
- `summary`, `tags`, and `matched_queries`;
- `read_notes.fields`;
- `read_notes.missing` and `read_notes.issues`;
- Unix timestamps usage;
- strict list formatting for tags and aliases;
- `links_style`;
- `missing_timestamp`;
- structured diagnostic fields;
- current operational CLI commands;
- request validation limits;
- full index rebuild behavior without legacy upgrade/migration tracks.

Completely purge references to:

- `read_note`;
- permalink;
- PageRank;
- schema upgrade paths;
- migrations;
- backward compatibility;
- the legacy single-query API;
- schema versions 2 and 3.

## Acceptance Criteria

- All commands documented actually exist in the runtime.
- All MCP parameters match their respective DTO fields perfectly.
- There are no statements claiming automatic upgrades of existing indexes.
- No schema version history remains.
- Manifest and note examples align perfectly with the parser.
- Timestamp examples strictly showcase Unix seconds.

## Key Files

- `README.md`
- `PROMPTS.md`
- `ARCHITECTURE.md`
- `AGENTS.md`
- `README.cli.md`
- `internal/format/markdown/README.md`

---

## Recommended Order of Execution

1. [x] `MNEMONIC-201` — Strip schema upgrades and versioning logic.
2. [x] `MNEMONIC-203` — Implement typed errors for `ShowMany`.
3. [x] `MNEMONIC-206` — Enforce strict validation rules for tags/aliases lists.
4. [x] `MNEMONIC-207` — Centralize shared input validation limits.
5. [x] `MNEMONIC-202` — Fix logger dependency injection paths.
6. [x] `MNEMONIC-204` — Append duration metrics to batch-read logging events.
7. [x] `MNEMONIC-205` — Inject the clock abstraction into the diagnostics path.
8. [x] `MNEMONIC-209` — Retain original queries in matched results.
9. [x] `MNEMONIC-210` — Unify Unix timestamp utilization across MCP tools.
10. [x] `MNEMONIC-208` — Batch candidate suggestions query execution.
11. [x] `MNEMONIC-211` — Clean up the legacy store search engine.
12. [x] `MNEMONIC-212` — Overhaul documentation assets.
