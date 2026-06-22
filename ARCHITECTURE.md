# Mnemonic System Architecture

This document defines the verified, actual architecture of the `mnemonic` knowledge base system. It outlines the design boundaries, operational flows, data storage layouts, and core invariants of the codebase.

---

## 1. Architectural Philosophy & Context Split

The application strictly divides its logic into three operational contexts to maintain predictable dependency flows.

```
                  ┌────────────────────────────────────────┐
                  │                Adapters                │
                  │       (cli, stdio_mcp, web_mcp)        │
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼
                  ┌────────────────────────────────────────┐
                  │               Bootstrap                │
                  │            (internal/app)              │
                  └─────────┬────────────────────┬─────────┘
                            │                    │
                            ▼                    ▼
              ┌──────────────────────────┐  ┌──────────────────────────┐
              │     Catalog Context      │  │     Runtime Context      │
              │  (Multi-KB, Registry)    │  │  (Single KB Operations)  │
              └──────────────────────────┘  └──────────────────────────┘
```

### Catalog Context

- **Scope:** Knows about multiple projects (knowledge bases).
- **Responsibilities:** Registry scanning, project initialization (`init`), imports, removal, and selector resolution (mapping a user-supplied string to a physical project).
- **Primary Service:** `catalogsvc.Service`.

### Runtime Context (`RuntimeApp`)

- **Scope:** Knows about **exactly one** active knowledge base.
- **Responsibilities:** Document mutation (create, edit, delete), index queries (search, backlinks, tags), and single-project indexing.
- **Invariants:** Runtime services do not access the global registry, do not parse project selector CLI flags, and do not handle multi-project states.
- **Primary Service Group:** `notesvc.Service`, `searchsvc.Service`, `indexsvc.Service`.

### Maintenance Context

- **Scope:** Iterates across catalog entries to execute target tasks.
- **Responsibilities:** Aggregating indexing or verification runs across multiple independent projects.
- **Primary Service:** `maintsvc.Service`.

---

## 2. Storage & Directory Layout

`mnemonic` distinguishes between human-authored source files and machine-generated operational data.

```
~/.config/mnemonic/config.toml     <-- User Configuration
~/.mnemonic/                       <-- Registry (Memories Home)
  ├── personal/
  │     └── mnemonic.toml          <-- Central Project (Manifest)
  └── work.toml                    <-- Local Project Pointer File

~/.local/state/mnemonic/projects/  <-- Machine Operational State
  └── <project-uuid>/
        ├── index.sqlite           <-- Derived Search Index
        ├── index.sqlite-wal
        ├── locks/
        │     └── write.lock       <-- Writer Concurrency Lock
        └── reindex.lock           <-- Indexer Concurrency Lock
```

### The Registry Store (File-Based)

Unlike previous designs utilizing a shared SQLite catalog database, `mnemonic` implements a **fully decentralized file-based registry** under `MemoriesHome` (defaulting to `~/.mnemonic`):

- **Central Projects:** Represented by directories containing a `mnemonic.toml` manifest.
- **Local Projects:** Represented by pointer files (`[slug].toml`) mapping a remote or workspace directory containing `mnemonic.toml` back to the registry.
- **Corruptions & Orphans:** If a directory or pointer is unparseable, or if a local project's target directory is moved or missing, the registry scanner marks it as `[CORRUPTED]` or `[ORPHANED]` rather than failing the scan.

### Operational State Directories

All index, cache, lock, and WAL files reside within user state homes (following XDG paths or overrides). They are keyed directly by the project's unique UUID, ensuring that moving or renaming a note directory does not cause database collisions.

---

## 3. Note Lifecycles and Mutation Safety

To ensure data integrity, any operation that alters a Markdown file or rebuilds an index must adhere to specific safety protocols.

### Optimistic Lock-Free Concurrency (`if_match_hash`)

- Write operations (updates, replacements, deletions) can receive an `if_match_hash` parameter.
- The system reads the target file, calculates its current SHA-256 hash, and compares it with the requested hash. If they do not match, the transaction is rejected with an `apperr.CodeUnsafe` error to prevent lost updates.

### File Locking Invariants

- **Write Lock (`write.lock`):** Acquired during note mutations (create, edit, delete). It prevents simultaneous processes from modifying the same project folder.
- **Indexer Lock (`reindex.lock`):** Acquired during index rebuilds.
- Both locks have short timeout phases (150ms) to maintain responsiveness.

### Atomic Writing Lifecycle

When writing notes or indices, direct file truncation is prohibited:

1. Stage the data in a temporary file (e.g., `<name>.*.tmp` or `.new.sqlite`) in the target directory.
2. Call `Sync()` on the file descriptor to flush bytes to disk.
3. Close the file.
4. Perform an atomic rename over the destination path.
5. Sync the parent directory to commit metadata changes.

---

## 4. Indexing and Search System

The search and query capabilities are powered entirely by **SQLite FTS5**. No vector or external model runtimes are supported.

```
Markdown Files ──► Parser ──► NoteDoc ──► index.new.sqlite ──► PRAGMA quick_check ──► Swap to index.sqlite
```

### Index Rebuild Lifecycle

The indexing engine (`sqliteindex.Store`) treats database schemas as immutable. If the database file is missing or contains an outdated schema version, it is rebuilt from scratch rather than migrated in place:

1. Scan all active Markdown files in the project root (excluding `.trash`).
2. Parse frontmatter metadata, wiki-links, inline tags, observations, and explicit relations.
3. Construct a temporary SQLite index database file (`index.new.sqlite`).
4. Apply the defined schema, including tables for `notes`, `note_tags`, `observations`, `links`, and the virtual `notes_fts` table.
5. Populate tables and resolve target links.
6. Run SQLite `PRAGMA quick_check` against the temp database.
7. Close connections, remove the old index database and sidecar files (`-wal`, `-shm`), and rename the temporary file to its stable path.

---

## 5. Dependency Rules & Boundaries

To preserve testability and architectural clean lines, packages must respect boundary invariants:

- **Adapters (`internal/adapter`)** act as transport mapping layers. They must not write raw SQL queries, scan file systems for projects directly, or parse markdown files.
- **Services (`internal/service`)** implement business use cases. They interact strictly with interfaces or typed parameters, remaining agnostic of Cobra CLI context or HTTP request payloads.
- **Stores (`internal/store`)** implement physical storage drivers. They must not reference configuration layers, environment variables, or execution engines.
- **Standard Library Only for SQLite:** The project uses the pure Go driver (`modernc.org/sqlite`) to avoid CGO compiler toolchain requirements.
