# Technical Specification (Terms of Reference)
## Minimal Frontmatter Migration & Hybrid Metadata Resolution

---

## 1. Objective and Context

The goal is to simplify the Markdown note format processed by `mnemonic`. Previously, every note was forced to contain a heavy YAML frontmatter block containing a UUID, slug, title, tags, summary, and creation/modification timestamps.

We are shifting to a **minimal frontmatter model**:
*   The **only strictly required** on-disk YAML field is `mnemonic_note_id`.
*   All other fields (`title`, `slug`, `created_at`, `updated_at`) are now **completely optional** in the YAML block.
*   If these optional fields are missing on disk, they must be resolved **dynamically at runtime** during indexing, reading, and search operations, while keeping the Markdown file clean.
*   The priority for target operating systems is **Linux** and **macOS (Darwin)**. Portable fallbacks must be implemented for other operating systems.

---

## 2. Directory Layout & Architecture Overview

Before starting, familiarize yourself with these packages:
1.  `internal/format/markdown`: Handles Markdown parsing, Goldmark AST manipulation, and YAML frontmatter reading/writing.
2.  `internal/platform/fs`: Deals with raw disk operations and filesystem locks. You will add platform-specific file timing code here.
3.  `internal/store/markdownstore`: High-level reader/writer of Markdown files.
4.  `internal/store/sqliteindex`: Scans notes, parses relationships, and populates the SQLite FTS5 database.

---

## 3. Detailed Component Modifications

### Step 1: Platform-Specific Creation Time (`btime`) Extraction
DONE

---

### Step 2: Update the `Note` Model in `internal/format/markdown/note.go`
DONE

---

### Step 3: Update Note Rendering in `internal/format/markdown/render.go`
DONE

---

### Step 4: Update the Storage Layer (`internal/store/markdownstore/store.go`)
DONE

---

### Step 5: Update the Database Indexer (`internal/store/sqliteindex/scan.go`)
DONE

---

### Step 6: Cleanup Diagnostics & Doctor Checks
DONE

---

## 4. Testing Plan (Instructions for Verification)
DONE
