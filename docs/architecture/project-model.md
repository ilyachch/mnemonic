# Project Model

## Purpose

The project model explains how `mnemonic` chooses a working project and where that project's Markdown lives. The implementation supports repository-attached projects via `.mnemonic` and non-local project manifests via `mnemonic.toml`.

## Invariants

- Project selection order is CLI flag, `MNEMONIC_PROJECT`, then current-directory discovery.
- If discovery is ambiguous or absent, commands fail instead of choosing a hidden default.
- `.mnemonic` supports `regular` and `local` projects.
- `mnemonic.toml` supports `regular` and `detached` projects.
- Manifest and `.mnemonic` schema versions must be explicitly present and equal to `1`.

## Current model

### `.mnemonic`

A repository-root `.mnemonic` file contains one or more `[[projects]]` entries. Each entry has a stable `id`, human-facing `name`, `slug`, `kind`, `memories_path`, and `markdown_format_version`.

### `mnemonic.toml`

A non-local memories root contains `mnemonic.toml` with `project_id`, `name`, `slug`, `kind`, `markdown_format_version`, layout defaults, and generator metadata.

### Project kinds

- `local` stores notes under the repository, usually inside `.mnemonic-memories/<slug>`.
- `regular` stores notes in the configured memories home while remaining attached to a repository.
- `detached` stores notes in the memories home without repository attachment.

## Related packages

- `internal/project` for parsing, validation, init, import, discovery, and resolution.
- `internal/cli` for `project init`, `project import`, `project discover`, `project show`, and `project list`.
- `internal/config` and `internal/paths` for configured memories-home resolution.
