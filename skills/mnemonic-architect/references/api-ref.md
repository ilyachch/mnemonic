# Mnemonic Project CLI Reference

## Effective Paths

```bash
mnemonic config show
```

Default layout:

```text
~/.config/mnemonic/config.toml
~/.mnemonic/
~/.local/state/mnemonic/projects/<project-id>/
```

## `project init`

```bash
mnemonic project init NAME [--local] [--description TEXT]
```

- Central mode stores the project under the memories home.
- Local mode creates `<cwd>/.mnemonic-memories/<slug>/mnemonic.toml` and a pointer file under the memories home.
- Initialization builds the initial index.

## Project Description and Custom Instructions

Both fields are optional strings in `mnemonic.toml`:

```toml
description = "Engineering knowledge for the Heimdall Python project, including architecture decisions, implementation notes, and runbooks."
custom_instructions = """
Use English for note titles and bodies.
Record accepted architecture changes as decision notes.
Do not store secrets or transient debugging output.
"""
```

Runtime behavior:

- `description` is added to MCP global instructions under `Project Description`. It is also included in the descriptions of the `search_notes` and `create_note` tools.
- `custom_instructions` is added to MCP global instructions under `Custom Instructions`.
- Empty or whitespace-only values are omitted.
- The fields are loaded when the project runtime and MCP server are created. Restart or reconnect the server after editing them.
- Changing these fields alone does not require index rebuild.

Use `description` for factual scope: what the knowledge base contains, which project or domain it covers, and any important exclusions. Use `custom_instructions` for project-specific agent behavior: language, evidence rules, note conventions, linking policy, durable-write criteria, and prohibited content.

Initialization accepts only the description directly:

```bash
mnemonic project init NAME [--local] --description TEXT
```

To update either value, resolve the manifest and edit it as TOML:

```bash
manifest_path=$(mnemonic --json project show SLUG | jq -r '.location.manifest_abs')
$EDITOR "$manifest_path"
mnemonic --json project show SLUG | jq '{description, custom_instructions}'
```

Preserve `project_id`. Do not place credentials, tokens, temporary task state, or instructions that attempt to bypass read-only mode or mutation safety in either field.

## `project import`

```bash
mnemonic project import [PATH] [--dry-run]
```

Onboards a raw Markdown directory in place:

- Creates `mnemonic.toml` when missing.
- Hydrates notes missing `mnemonic_note_id`.
- Registers the directory through a pointer file.
- Rebuilds the index.
- Does not copy files to another project.

## `project add`

```bash
mnemonic project add [PATH]
```

Registers an existing structured project that already contains a valid `mnemonic.toml`, then rebuilds its index.

## `project sync`

```bash
mnemonic project sync PROJECT_SELECTOR [FILES...] [--dry-run]
```

Hydrates missing note IDs for all project files or selected files. A non-dry-run sync attempts to rebuild the index.

## `project reindex`

```bash
mnemonic project reindex [PROJECT]
mnemonic project reindex --all
```

Rebuilds disposable SQLite indexes from Markdown source files.

## `project doctor`

```bash
mnemonic project doctor [PROJECT]
mnemonic project doctor --all
mnemonic project doctor PROJECT --kind unresolved_link
```

Runs project health checks. Kind-filtered mode exposes detailed note diagnostics.

## `project remove`

```bash
mnemonic project remove SLUG
mnemonic project remove SLUG --wipe
```

Default behavior:

- Removes the registry entry.
- Deletes index and lock state.
- Keeps Markdown notes.

With `--wipe`:

- Also removes the entire Markdown root.

## `project list` and `project show`

```bash
mnemonic project list
mnemonic project show SLUG
mnemonic --json project show SLUG
```

`project show` returns project identity, type, description, link style, state path, memories path, manifest path, and repository root.

## Project Manifest Example

```toml
version = 1
project_id = "550e8400-e29b-41d4-a716-446655440000"
name = "Research"
slug = "research"
type = "local"
markdown_format_version = 1

description = "Research notes and source analysis"
custom_instructions = "Prefer source-linked summaries."

[format]
links_style = "wiki"

[layout]
notes_glob = ["**/*.md"]
ignore = ["mnemonic.toml", ".trash/**"]
```

Preserve `project_id` when editing an existing manifest.
