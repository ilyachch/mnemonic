# ADR 0003: SQLite FTS5 Before Vector Search

## Context

Search is a core capability, but the current product constraints favor deterministic local behavior, offline execution, and simple deployment. Introducing embeddings or vector indexes would add model dependencies, larger operational surface, and less predictable results.

## Decision

The current search implementation uses SQLite FTS5 as the primary and only search engine. Ranking, snippets, and filtering are derived from the SQLite index built from Markdown notes. Vector search is out of scope for the implemented system.

## Consequences

- Search stays offline, deterministic, and easy to ship.
- Index rebuild logic remains centered on SQLite schema and note parsing.
- Search quality improvements should first come from better note extraction, query shaping, and schema changes within SQLite.
- Features must not add embedding pipelines or vector-specific storage as if they were already part of the supported architecture.
