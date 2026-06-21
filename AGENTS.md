# AGENTS.md

# Mnemonic architecture refactor instructions

## Goal

Refactor `mnemonic` toward a single-KB runtime architecture.

`mnemonic` may have many registered knowledge bases, but each runtime process must operate on exactly one selected knowledge base.

The target split is:

```text
Catalog context
  knows about many knowledge bases
  handles project management and selector resolution

Runtime context
  knows about exactly one selected knowledge base
  handles notes/search/tags/backlinks/reindex/doctor/web/stdio for that one KB

Maintenance context
  applies runtime operations to many KBs by iterating over catalog entries
```

## Current API direction

The public CLI API is still allowed to change because the project is in closed testing.

Target command organization:

```bash
mnemonic project init NAME
mnemonic project list
mnemonic project show PROJECT
mnemonic project import [PATH]
mnemonic project remove PROJECT

mnemonic --project work notes list
mnemonic --project work notes create --title "Hello"

mnemonic project reindex [PROJECT]
mnemonic project doctor [PROJECT]
mnemonic project reindex --all
mnemonic project doctor --all

mnemonic --project work web serve --port 8081
mnemonic --project work stdio
```

Root-level `mnemonic init` should be removed unless a temporary hidden compatibility alias is explicitly requested. Prefer no alias.

## Selector rules

For single-project runtime commands that accept an optional project argument:

```text
1. positional PROJECT
2. --project
3. MNEMONIC_PROJECT
4. usage error
```

Examples:

```bash
mnemonic project reindex work
mnemonic --project work project reindex
MNEMONIC_PROJECT=work mnemonic project reindex
```

`project reindex` without positional `PROJECT`, without `--project`, and without `MNEMONIC_PROJECT` must return a usage error.

`project reindex --all` and `project doctor --all` are mass operations.

Rules for `--all`:

```text
--all must not be combined with positional PROJECT.
--all must not be combined with --project.
--all uses maintenance service, not RuntimeApp directly from CLI.
```

## Core architecture rule

```text
Catalog resolves and enumerates knowledge bases.
Runtime operates on exactly one KnowledgeBase.
Maintenance applies runtime operations to many KnowledgeBases.
Adapters parse UX and choose catalog/runtime/maintenance path.
```

## Allowed dependencies by layer

### domain

Allowed:

```text
standard library only, unless explicitly justified
```

Forbidden:

```text
internal/app
internal/service
internal/store
internal/adapter
cobra
sql
filesystem access
environment access
```

### platform

Allowed:

```text
low-level config, paths, fs, lock, clock, id generation, buildinfo
```

Forbidden:

```text
business use cases
CLI/web/stdio concerns
registry selector logic
```

### format

Allowed:

```text
markdown parsing/rendering
frontmatter
wikilinks
tags
observations
relations
```

Forbidden:

```text
registry resolving
project selector resolving
index paths
CLI/web/stdio concerns
```

### store

Allowed:

```text
low-level persistence
registry scanning
markdown file operations
sqlite index operations
```

Forbidden:

```text
CLI flags
environment selector lookup
cobra
web server logic
stdio tool registration
application orchestration
```

### service

Allowed:

```text
business use cases
orchestration over stores
runtime command logic
catalog command logic
maintenance command logic
```

Forbidden in runtime services:

```text
MNEMONIC_PROJECT
--project
projectSelectorValue
registry.Resolve
registry.Scan
registry.Slugs
ProjectResolver
ProjectResolution
sql.Open in adapters
index.Path(projectID) in runtime paths
```

### app

Allowed:

```text
Bootstrap construction
RuntimeApp construction
wiring services and stores
```

Forbidden:

```text
business logic
CLI output formatting
SQL queries
markdown parsing details
```

### adapter/cli

Allowed:

```text
cobra commands
flags
environment boundary reads
choosing catalog/runtime/maintenance path
printing human/json output
exit code mapping
```

Forbidden:

```text
SQL queries
direct sqlite opening
registry scanning/resolving directly
markdown persistence directly
runtime business logic
doctor check implementation
reindex implementation
```

### adapter/web

Allowed:

```text
HTTP transport
middleware
request/response mapping
auth/token handling passed from CLI/bootstrap
calling runtime services
```

Forbidden:

```text
project selector resolving
MNEMONIC_PROJECT
registry access
direct sqlite opening
raw index DB ownership
```

### adapter/stdio

Allowed:

```text
MCP/stdio transport
tool registration
input/output mapping
calling runtime services
```

Forbidden:

```text
project selector resolving
registry access
direct sqlite opening
GetMemoriesRoot style callbacks
GetIndexDB style callbacks
direct reindex orchestration
```

## Runtime service rules

Runtime services must be built from an already resolved `kb.KnowledgeBase`.

Runtime services must not accept project selectors.

Correct:

```go
type SearchInput struct {
    Query string
    Limit int
    Tag   string
}
```

Wrong:

```go
type SearchInput struct {
    ProjectSelector string
    Query           string
}
```

## KnowledgeBase model

Use one explicit domain model for a selected knowledge base:

```go
type KnowledgeBase struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    Slug         string `json:"slug"`
    Kind         string `json:"kind"`
    RootDir      string `json:"root_dir"`
    RepoRootDir  string `json:"repo_root_dir,omitempty"`
    ManifestPath string `json:"manifest_path,omitempty"`
    StateDir     string `json:"state_dir"`
    IndexPath    string `json:"index_path"`
}
```

`RootDir`, `StateDir`, and `IndexPath` must be resolved before RuntimeApp is created.

## Reindex after writes

Markdown files are the source of truth.

For write operations:

```text
notes create -> markdown write -> index rebuild
notes edit   -> markdown write -> index rebuild
notes delete -> markdown write -> index rebuild
```

If markdown mutation succeeds but index rebuild fails, do not pretend the markdown mutation was rolled back.

Prefer result DTOs with explicit index status:

```go
IndexStatus string `json:"index_status"` // ok, stale, skipped
IndexError  string `json:"index_error,omitempty"`
```

If preserving old error behavior is easier in an intermediate step, document it in the step notes and fix it before final audit.

## Mass operations

Use a dedicated maintenance service for operations over many KBs.

Example package:

```text
internal/service/maintsvc
```

Maintenance service may use:

```text
catalog service
runtime factory
indexsvc.Rebuild
indexsvc.Doctor
```

Maintenance service must not duplicate reindex or doctor logic.

Maintenance service should aggregate per-project results.

A failure in one KB should not stop the whole mass operation unless catalog enumeration itself fails.

For `--all`, return non-zero exit code if one or more projects failed, while still printing the full report.

## Legacy policy

Legacy packages are temporary migration scaffolding only.

Final state should remove or fully absorb:

```text
internal/notes
internal/index
internal/search
internal/graph
internal/project
internal/registry
internal/cli
internal/mcp
internal/web
```

Expected final homes:

```text
internal/store/markdownstore
internal/store/sqliteindex
internal/store/registry
internal/format/markdown
internal/service/catalogsvc
internal/service/notesvc
internal/service/searchsvc
internal/service/indexsvc
internal/service/maintsvc
internal/adapter/cli
internal/adapter/stdio
internal/adapter/web
```

Avoid naming two packages `markdown` if that creates import ambiguity. Prefer:

```text
internal/store/markdownstore
internal/format/markdown
```

## Work protocol for every step

Each step must be implemented as an isolated change suitable for one commit.

Before starting a step:

```bash
git status
```

The working tree should be clean unless continuing an interrupted step.

After code changes:

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

If available:

```bash
go vet ./...
```

Review:

```bash
git status
git diff
```

Check there are no:

```text
debug prints
commented-out old code
temporary files
accidental sqlite/cache/lock files
unrelated changes
```

Use targeted grep checks listed in each step.

## Commit protocol

Each successful step should become one commit.

Use the commit message specified by the step unless the implementation materially differs.

Do not squash during the step-by-step process. Squash only after the full refactor is complete and verified.

## If interrupted

If work stops mid-step:

1. Run `git status`.
2. Inspect the last completed commit.
3. Re-run the verification commands for the last completed step.
4. Continue from the first step whose criteria are not fully satisfied.

Do not restart the whole refactor if previous steps are already committed and passing.

## Final architecture invariant

After the refactor:

```text
Runtime services do not know about registry.
Runtime services do not know about project selectors.
Web and stdio receive RuntimeApp or runtime services.
CLI is a thin adapter.
SQL lives in sqlite index store.
Markdown persistence lives in markdown store.
Registry resolving lives in catalog/store registry only.
Mass operations live in maintenance service.
```
