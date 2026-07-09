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
