---
name: mnemonic-architect
description: Designs and manages mnemonic project lifecycles, including central and local project initialization, Markdown directory onboarding, registry operations, mnemonic.toml configuration, project removal, synchronization, and index creation. Use when setting up, importing, registering, relocating, or decommissioning knowledge bases.
compatibility: Requires the mnemonic CLI in PATH and filesystem access to the target project directories.
metadata:
  author: mnemonic
  version: "1.0.0"
---

# Mnemonic Project Architect

Manage project lifecycle and configuration without confusing registry metadata, source Markdown, and derived index state.

See [the project CLI reference](references/api-ref.md) for exact commands and storage layout.

## Storage Model

Mnemonic separates three concerns:

1. Project registry under the memories home, normally `~/.mnemonic/`.
2. Project source files and `mnemonic.toml`.
3. Derived state under the XDG state directory, normally `~/.local/state/mnemonic/projects/<project-id>/`.

Inspect effective paths before making assumptions:

```bash
mnemonic config show
```

### Central Projects

A central project is stored under:

```text
~/.mnemonic/<slug>/
  mnemonic.toml
  *.md
```

### Local Projects

A local project is backed by a pointer file:

```text
~/.mnemonic/<slug>.toml
```

The pointer references a manifest created beneath the current working directory:

```text
<working-directory>/.mnemonic-memories/<slug>/mnemonic.toml
```

Local projects are useful when knowledge should live near a repository or workspace while remaining discoverable through the global registry.

## `mnemonic.toml`

The project manifest defines identity and project behavior. Important fields include:

- `project_id`
- `name`
- `slug`
- `type`
- `description`
- `custom_instructions`
- `format.links_style`
- `layout.notes_glob`
- `layout.ignore`

Treat `project_id` as stable identity. Do not copy a manifest to create a new project without generating a new project ID through `project init` or `project import`.

## New Environment Workflow

### 1. Choose Central or Local

Use central when the knowledge base should live entirely under the memories home.

Use local when the knowledge base belongs to a repository or workspace and should be represented in the global registry by a pointer.

### 2. Initialize

Central:

```bash
mnemonic project init research --description "Research notes and source analysis"
```

Local, from the intended workspace directory:

```bash
mnemonic project init research --local --description "Research notes and source analysis"
```

Initialization creates the manifest, registers the project, and builds an initial index.

### 3. Configure the Manifest

Inspect the project:

```bash
mnemonic project show research
```

Edit `mnemonic.toml` only when needed for:

- Project description.
- Custom MCP instructions.
- Link style.
- Note globs and ignore patterns.

Keep the TOML syntactically valid and preserve `project_id`.

### 4. Establish Tag Vocabulary

Mnemonic currently has no project-level "base tags" declaration. Tags exist on notes and appear in `mnemonic tags list` after indexing.

Establish a vocabulary by:

- Applying agreed tags to imported notes.
- Creating a small project conventions note.
- Avoiding near-duplicate spellings and singular/plural variants.

Do not claim that project initialization configures tags automatically.

### 5. Add Content

Choose the correct onboarding path:

- Existing raw Markdown directory: `project import`.
- Existing structured project with a valid `mnemonic.toml`: `project add`.
- New files copied into an existing project: `project sync` followed by `project reindex` when needed.

### 6. Rebuild and Verify

After external file operations:

```bash
mnemonic project sync research
# Run only when sync reports a stale index or after complex bulk changes:
mnemonic project reindex research
mnemonic project doctor research
```

`project sync` hydrates missing `mnemonic_note_id` fields and rebuilds the index when not in dry-run mode. An explicit final reindex is useful after complex bulk operations or when the index is reported stale.

## Import Semantics

`mnemonic project import PATH` onboards the directory at `PATH` as its own project in place:

1. Creates `PATH/mnemonic.toml` when absent.
2. Adds missing `mnemonic_note_id` values to Markdown notes.
3. Registers a pointer in `~/.mnemonic/`.
4. Rebuilds the new project's index.

It does **not** copy files into a previously initialized project and does not accept a destination project selector.

Example:

```bash
mnemonic project import "$HOME/Downloads/docs"
```

Use `--dry-run` first for unfamiliar directories:

```bash
mnemonic project import "$HOME/Downloads/docs" --dry-run
```

## Importing Files Into an Existing Project

There is no native `project import SOURCE --into PROJECT` command in the current CLI.

To place external Markdown files into an existing project:

1. Resolve the project's `memories_abs` path with `mnemonic --json project show PROJECT`.
2. Copy only the intended Markdown files into that path using a filesystem tool.
3. Avoid overwriting `mnemonic.toml` or existing notes.
4. Run `mnemonic project sync PROJECT`.
5. Run `mnemonic project doctor PROJECT`.
6. Reindex explicitly if sync reports a stale index.

For a request such as "create local project research and import all Markdown files from `~/Downloads/docs` into it", do not run `project import` after `project init` and claim that it targeted `research`. Explain the CLI limitation, create the project, copy the files into its resolved memories path, then sync and validate.

## Registering an Existing Structured Project

When a directory already contains a valid `mnemonic.toml`:

```bash
mnemonic project add /path/to/project
```

This registers the project and rebuilds its index without hydrating a raw directory.

Use `project add` instead of `project import` when preserving an existing project identity and manifest is required.

## Registry Management

List projects:

```bash
mnemonic project list
```

Inspect one project:

```bash
mnemonic project show research
```

### Safe Removal

Default removal:

```bash
mnemonic project remove research
```

This removes the registry entry and derived index/state while leaving Markdown note files intact.

Destructive removal:

```bash
mnemonic project remove research --wipe
```

`--wipe` also removes the project's Markdown root. Require explicit confirmation and a verified backup before using it.

For a central project, default removal removes its manifest registry entry but leaves the remaining Markdown files. Preserve a copy of `mnemonic.toml` when future re-registration with the same identity may be needed.

## Project Relocation

For a local project moved to a new path:

1. Remove the stale registry entry without `--wipe`.
2. Confirm the moved directory still contains `mnemonic.toml`.
3. Run `mnemonic project add /new/path`.
4. Run `mnemonic project doctor SLUG`.

Do not edit pointer files by hand unless the CLI cannot be used and the manifest path is fully understood.

## Failure Handling

### Orphaned Local Project

A pointer exists but its target manifest is missing. Locate or restore the project, then re-add it. Do not wipe until source-note location is confirmed.

### Duplicate Slug

Choose a unique project name or update the existing project's lifecycle intentionally. Do not overwrite a registry entry.

### Stale Index

Run project doctor, then reindex the affected project. Index files are derived and may be safely rebuilt from valid source notes.

### Invalid Manifest

Repair TOML syntax and required identity fields. Preserve the original `project_id` for an existing project.

## Completion Checklist

- [ ] Inspected effective paths with `mnemonic config show` when location mattered.
- [ ] Chose central or local mode deliberately.
- [ ] Preserved project identity and `mnemonic.toml`.
- [ ] Used `project import` only for in-place raw-directory onboarding.
- [ ] Used `project add` for an existing structured project.
- [ ] Did not claim that import copies into an existing project.
- [ ] Used safe removal without `--wipe` by default.
- [ ] Synchronized, reindexed when necessary, and ran doctor after bulk operations.
