# ADR 0001: No Default Project

## Context

Most CLI and MCP operations need a concrete project context to resolve note roots, indexes, and registry state. The code supports selection by CLI flag, environment variable, or current-directory discovery from `.mnemonic`. Hidden fallback to some remembered project would make note mutations and search results ambiguous.

## Decision

`mnemonic` does not use an implicit default project. Project resolution order is explicit selector first, then `MNEMONIC_PROJECT`, then nearest `.mnemonic` discovery. If no project is resolved, or if `.mnemonic` contains multiple projects and the selector is absent, the command returns an error.

## Consequences

- CLI and MCP behavior stays predictable and auditable.
- Repository-local workflows remain ergonomic when `.mnemonic` contains one project.
- Multi-project repositories require an explicit `--project` or `MNEMONIC_PROJECT`.
- Future features must preserve explicit failure on ambiguous project context.
