---
name: mnemonic-architect
description: >-
  Use this skill when creating, importing, registering, relocating, configuring, or removing mnemonic projects and knowledge bases. It manages central versus local projects, mnemonic.toml, registry entries, filesystem onboarding, synchronization, and derived indexes. Do not use it for ordinary note editing or diagnostic repair.
compatibility: >-
  Requires the mnemonic CLI in PATH and filesystem access to target project directories.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Project Architect

Manage project lifecycle without confusing registry metadata, Markdown sources, and derived index state.

See [the project CLI reference](references/api-ref.md) for exact commands and storage layout.

## Storage Model

Mnemonic separates:

1. Project registry under the memories home, normally `~/.mnemonic/`.
2. Project source files and `mnemonic.toml`.
3. Derived state under the XDG state directory, normally `~/.local/state/mnemonic/projects/<project-id>/`.

Inspect effective paths before making assumptions:

```bash
mnemonic config show
```

### Central Projects

```text
~/.mnemonic/<slug>/
  mnemonic.toml
  *.md
```

### Local Projects

A pointer file is stored at:

```text
~/.mnemonic/<slug>.toml
```

It references a manifest beneath the current working directory:

```text
<working-directory>/.mnemonic-memories/<slug>/mnemonic.toml
```

## `mnemonic.toml`

Important fields include:

- `project_id`
- `name`
- `slug`
- `type`
- `description`
- `custom_instructions`
- `format.links_style`
- `layout.notes_glob`
- `layout.ignore`

Treat `project_id` as stable identity. Do not copy a manifest to create a new project; use `project init` or in-place `project import`.

## Project Context Fields

`description` and `custom_instructions` are optional, but they materially affect how an MCP client understands and uses the project. Keep them stable, project-specific, and free of secrets.

### Runtime Effect

Mnemonic uses these fields when an MCP server starts:

- `description` is appended to the global server instructions as **Project Description**. It is also prepended to the `search_notes` and `create_note` tool descriptions. A good description helps the agent decide whether this knowledge base is relevant and what information belongs in it.
- `custom_instructions` is appended to the global server instructions as **Custom Instructions**. It controls project-specific working conventions, but it does not grant permissions, enable unavailable tools, or override read-only mode.

After changing either field, restart or reconnect the MCP server so a new session receives the updated context. Reindexing is not required solely for these metadata changes.

### Writing `description`

Use `description` to answer: **What knowledge does this project contain?**

Include the durable parts of the scope:

- Domain, product, repository, or research area.
- Main kinds of material, such as decisions, implementation notes, runbooks, meeting outcomes, or source analysis.
- Important boundaries when similarly named projects could be confused.

Keep it factual and concise, normally one to three sentences. Do not put behavioral commands, temporary status, credentials, or a copy of the project name alone in this field.

Useful pattern:

```text
Contains <knowledge domain> for <project or audience>, including <main artifact types>. Excludes <important boundary>, when needed.
```

Good example:

```toml
description = "Engineering knowledge for the Heimdall Python project, including architecture decisions, CLI behavior, implementation notes, and operational runbooks. It does not contain general Python reference material."
```

Weak example:

```toml
description = "Heimdall notes"
```

### Writing `custom_instructions`

Use `custom_instructions` to answer: **How should the agent work inside this project?**

Prefer short imperative rules covering only project-specific behavior, for example:

- Required language or terminology.
- Evidence and citation expectations.
- Note granularity and preferred note types.
- Linking, tagging, and naming conventions.
- What outcomes should be recorded as durable knowledge.
- What must not be stored, such as secrets or temporary debugging output.

Do not repeat Mnemonic's built-in tool instructions. Do not tell the agent to bypass hash checks, ignore read-only mode, perform destructive actions without confirmation, or save every conversation indiscriminately.

Example:

```toml
custom_instructions = """
Use English for note titles and bodies.
Treat repository source code and committed documentation as primary evidence.
Record accepted architecture changes as decision notes and link them to affected components.
Do not store credentials, access tokens, personal data, or transient debugging output.
"""
```

### Configure and Verify

Supply the description during initialization when the scope is already known:

```bash
mnemonic project init research --local \
  --description "Research notes on local-first knowledge systems, including source summaries, experiments, and design decisions."
```

The CLI currently has no dedicated command for updating `custom_instructions`. Resolve and edit the manifest directly:

```bash
manifest_path=$(mnemonic --json project show research | jq -r '.location.manifest_abs')
$EDITOR "$manifest_path"
```

Preserve `project_id`, keep both values as TOML strings, then verify the parsed result:

```bash
mnemonic --json project show research | jq '{description, custom_instructions}'
```

If the project purpose is not known well enough to write accurate instructions, prefer a narrow factual description and leave `custom_instructions` empty rather than inventing policy.

## New Environment Workflow

### 1. Choose Central or Local

Use central when the knowledge base should live under the memories home.

Use local when knowledge belongs near a repository or workspace and should be represented globally by a pointer.

### 2. Initialize

Central:

```bash
mnemonic project init research --description "Research notes and source analysis"
```

Local, from the intended workspace:

```bash
mnemonic project init research --local --description "Research notes and source analysis"
```

Initialization creates the manifest, registers the project, and builds an initial index.

### 3. Configure the Manifest

Inspect the project:

```bash
mnemonic project show research
```

Edit `mnemonic.toml` only when needed for project context, custom MCP instructions, link style, note globs, or ignore patterns. Follow the field guidance above and preserve `project_id`.

### 4. Establish Tag Vocabulary

Mnemonic has no project-level base-tags declaration. Tags exist on notes and appear after indexing.

Establish vocabulary through project conventions and consistent note tags. Do not claim that initialization configures tags automatically.

### 5. Add Content

Choose the correct path:

- Raw Markdown directory that should become its own project: `project import`.
- Structured project with `mnemonic.toml`: `project add`.
- Files copied into an existing project: filesystem copy, then `project sync`.

### 6. Synchronize and Verify

After external file operations:

```bash
mnemonic project sync research
# Run only if sync reports a stale index or after complex bulk changes:
mnemonic project reindex research
mnemonic project doctor research
```

A non-dry-run `project sync` hydrates missing IDs and attempts an index rebuild. Do not perform a redundant explicit reindex unless there is a reason.

## Import Semantics

`mnemonic project import PATH` onboards `PATH` as its own project in place:

1. Creates `PATH/mnemonic.toml` when absent.
2. Adds missing `mnemonic_note_id` values.
3. Registers a pointer under `~/.mnemonic/`.
4. Rebuilds that project's index.

It does not copy files into a previously initialized project and does not accept a destination selector.

```bash
mnemonic project import "$HOME/Downloads/docs" --dry-run
mnemonic project import "$HOME/Downloads/docs"
```

## Importing Files Into an Existing Project

There is no `project import SOURCE --into PROJECT` command.

To place external Markdown files into an existing project:

1. Resolve `memories_abs`:

```bash
mnemonic --json project show PROJECT
```

2. Copy only intended Markdown files into that path.
3. Do not overwrite `mnemonic.toml` or existing notes.
4. Run `mnemonic project sync PROJECT`.
5. Run `mnemonic project doctor PROJECT`.
6. Reindex only if sync reports stale state or the bulk operation warrants it.

For "create local project research and import files from `~/Downloads/docs` into it", do not claim that `project import` targets `research`. Initialize the project, resolve its path, copy the files, sync, and validate.

## Registering an Existing Structured Project

When a directory already contains a valid manifest:

```bash
mnemonic project add /path/to/project
```

Use `project add` when preserving existing identity and configuration is required.

## Registry Management

```bash
mnemonic project list
mnemonic project show research
```

### Safe Removal

```bash
mnemonic project remove research
```

Default removal removes the registry entry and derived state while keeping Markdown files.

```bash
mnemonic project remove research --wipe
```

`--wipe` also removes the Markdown root. Require explicit confirmation and a verified backup.

For central projects, preserve a copy of `mnemonic.toml` when future re-registration with the same identity may be needed.

## Project Relocation

For a moved local project:

1. Remove the stale registry entry without `--wipe`.
2. Confirm the moved directory contains `mnemonic.toml`.
3. Run `mnemonic project add /new/path`.
4. Run `mnemonic project doctor SLUG`.

Do not edit pointer files by hand unless the CLI cannot be used and the manifest path is fully understood.

## Failure Handling

### Orphaned Local Project

Locate or restore the target directory, then re-add it. Do not wipe until source-note location is confirmed.

### Duplicate Project Slug

Choose a unique project name or intentionally resolve the existing lifecycle. Do not overwrite a registry entry.

### Stale Index

Run doctor and reindex the affected project only when needed. Index files are derived.

### Invalid Manifest

Repair TOML syntax and required identity fields while preserving the original `project_id`.

## Completion Checklist

- [ ] Inspected effective paths when location mattered.
- [ ] Chose central or local mode deliberately.
- [ ] Preserved project identity and `mnemonic.toml`.
- [ ] Wrote `description` as factual knowledge scope rather than behavioral policy.
- [ ] Kept `custom_instructions` project-specific, actionable, and free of secrets.
- [ ] Restarted or reconnected the MCP server after changing project context fields.
- [ ] Used `project import` only for in-place raw-directory onboarding.
- [ ] Used `project add` for an existing structured project.
- [ ] Did not claim that import copies into an existing project.
- [ ] Used safe removal without `--wipe` by default.
- [ ] Synchronized, conditionally reindexed, and ran doctor after bulk operations.
