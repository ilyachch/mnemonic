# Mnemonic System Architecture

This document describes the architecture of the `mnemonic` personal knowledge base, defining component boundaries, data flows, and key system invariants.

---

## 1. Separation of Execution Contexts

The application is divided into three isolated execution contexts to ensure predictable dependency management.

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

- **Scope:** Global management level across multiple knowledge bases.
- **Responsibilities:** Scanning the project registry, importing existing catalogs, primary initialization (`init`), deleting projects, and matching user selectors to specific physical paths.
- **Key Component:** `catalogsvc.Service`.

### Runtime Context

- **Scope:** Operations restricted to exactly **one** selected knowledge base.
- **Responsibilities:** Mutating documents (creating, updating, deleting notes), executing search queries within the project (index searches, extracting tags, backlinks), and maintaining the local search index.
- **Invariants:** Internal runtime services do not have access to the global project registry and do not parse CLI project selection flags.
- **Key Components:** `notesvc.Service`, `searchsvc.Service`, `indexsvc.Service`.

### Maintenance Context

- **Scope:** Batch operations across the entire project catalog.
- **Responsibilities:** Rebuilding indexes or performing system integrity checks (`doctor`) for all registered projects in a single run.
- **Key Component:** `maintsvc.Service`.

---

## 2. Data Storage Layout

`mnemonic` separates user note files from service configuration and application state files.

```
~/.config/mnemonic/config.toml     <-- Global user configuration
~/.mnemonic/                       <-- Project registry (Memories Home)
  ├── personal/
  │     └── mnemonic.toml          <-- Central project (manifest)
  └── work.toml                    <-- Pointer file to a local project

~/.local/state/mnemonic/projects/  <-- State directory (machine-generated data)
  └── <project-uuid>/
        ├── index.sqlite           <-- Local search index
        ├── index.sqlite-wal
        ├── locks/
        │     └── write.lock       <-- Note write operation lock
        └── reindex.lock           <-- Index rebuild lock
```

### Decentralized Registry (Registry Store)

The registry is implemented as a flat file layout within the `MemoriesHome` directory (defaults to `~/.mnemonic`):

- **Central Projects:** Represented as subdirectories containing a `mnemonic.toml` manifest file.
- **Local Projects:** Represented as pointer files (`[slug].toml`) containing the absolute path to the `mnemonic.toml` manifest situated in an external workspace (e.g., a development Git repository).
- **Resilience to Corruption:** Errors reading individual manifests or the relocation of external local project directories do not disrupt registry scanning. Problematic projects are flagged as `[CORRUPTED]` or `[ORPHANED/MISSING]` in list outputs.

### State Files

All SQLite indexes, WAL journals, and lock files are stored in the system's state directory, isolated by the project's UUID. This prevents collisions when directories share identical names or slugs.

---

## 3. Mutation Safety and Transactional File Operations

### Optimistic Version Control via Hashing (`if_match_hash`)

- Note update and delete operations support an `if_match_hash` parameter.
- Before writing, the system calculates the SHA-256 hash of the current file content and compares it with the provided value. If they do not match, the transaction aborts with an `apperr.CodeUnsafe` error to prevent overwriting concurrently modified data.

### Lock Usage

- **Write Lock (`write.lock`):** Acquired during the creation, modification, or deletion of note files to prevent concurrent mutations in the same directory by different processes.
- **Indexing Lock (`reindex.lock`):** Prevents concurrent index regeneration runs.
- Lock acquisition wait times are capped at 150 ms to avoid blocking user interfaces.

### Atomic File Overwrites

To prevent data corruption during failures, writing to note files and the SQLite index database is performed using a temporary file replacement strategy:

1. Data is written to a temporary file (e.g., `<name>.*.tmp` or `.new.sqlite`) in the target directory.
2. A disk buffer flush is enforced using `Sync()`.
3. The file descriptor is closed.
4. An atomic rename system call (`rename`) replaces the target file.
5. `Sync()` is called on the parent directory to commit changes to the filesystem.

---

## 4. Indexing and Full-Text Search

Full-text search is implemented using the built-in **SQLite FTS5** extension with the standard `unicode61` tokenizer and `bm25` ranking.

```
Markdown Files ──► Parser ──► NoteDoc ──► index.new.sqlite ──► PRAGMA quick_check ──► Swap to index.sqlite
```

### Index Rebuild Lifecycle

The index database schema is treated as immutable. If schema version mismatches or corruption are detected, the database is rebuilt from scratch:

1. All Markdown files in the project's root directory are scanned (excluding system files and the `.trash` directory).
2. Metadata from the YAML frontmatter, wikilinks, inline tags, observations, and declared relations are extracted from each document.
3. A temporary database file (`index.new.sqlite`) is created.
4. The schema tables (`notes`, `note_tags`, `observations`, `links`, `notes_fts`) are applied.
5. Data is written to the temporary database, and relations between notes are resolved.
6. Structural integrity is validated via `PRAGMA quick_check`.
7. The connection is closed, old index files (including `-wal` and `-shm`) are deleted, and the temporary file is atomically renamed to the primary file name.

---

## 5. Scope of Responsibility and Package Import Rules

- **Adapters (`internal/adapter`):** Receive external requests and format responses. They must not contain SQL queries, scan paths directly, or parse Markdown files.
- **Services (`internal/service`):** Implement business scenarios. They operate on interfaces and domain structures, remaining independent of the CLI context or web request parameters.
- **Stores (`internal/store`):** Manage physical data representation. They do not read environment variables directly or reference global configuration values.
