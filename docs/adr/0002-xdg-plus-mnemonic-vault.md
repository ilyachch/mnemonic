# ADR 0002: XDG State with Separate Memories Vault

## Context

The system stores two different classes of data: human-authored Markdown notes and machine-managed state such as registry databases, indexes, caches, and locks. Mixing them in a single directory would make backups, cleanup, and portability harder.

## Decision

Human-authored note vaults stay in the configured memories home, while machine-managed files follow XDG-style config/data/state/cache separation. Local projects may keep notes inside the repository, but registry, index, cache, and lock files still belong under XDG-managed locations.

## Consequences

- User notes remain readable and movable as plain files.
- Operational state can be rebuilt or cleaned without touching source notes.
- Features must not write registry, cache, or lock state into `~/.mnemonic` or `.mnemonic-memories` as a shortcut.
- Tests and docs need to keep XDG overrides explicit when creating isolated environments.
