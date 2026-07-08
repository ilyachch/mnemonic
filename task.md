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
You need to implement a cross-platform helper to extract a file’s creation time (`btime`) and modification time (`mtime`) on macOS and Linux.

Create a new file structure inside `internal/platform/fs/`:

#### A. Create `internal/platform/fs/times_darwin.go`
This file uses Go build tags to compile only on macOS (Darwin). It reads `Birthtimespec` from the syscall structure.
```go
//go:build darwin

package fs

import (
	"os"
	"syscall"
	"time"
)

// GetFileTimes returns (birthTime, modTime, error) for Darwin.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		// Fall back to modtime if sys conversion fails
		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
	}

	// Birthtimespec represents the creation time on macOS.
	birthTime := time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec).UTC()
	return birthTime, fi.ModTime().UTC(), nil
}
```

#### B. Create `internal/platform/fs/times_linux.go`
This file compiles only on Linux. It uses the modern `statx` system call from `golang.org/x/sys/unix` to read `btime` if supported by the underlying filesystem (e.g., ext4, xfs, btrfs).
```go
//go:build linux

package fs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// GetFileTimes returns (birthTime, modTime, error) for Linux.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	var statx unix.Statx_t
	// AT_FDCWD means relative paths are evaluated relative to current directory
	err = unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &statx)
	if err != nil {
		// Fall back to standard ModTime if statx fails (e.g. unsupported filesystem or older kernel)
		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
	}

	// Verify that the kernel actually populated the Btime attribute
	if (statx.Mask & unix.STATX_BTIME) != 0 {
		birthTime := time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)).UTC()
		return birthTime, fi.ModTime().UTC(), nil
	}

	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
}
```

#### C. Create `internal/platform/fs/times_fallback.go`
This file serves as a fallback for any other platform (like Windows, FreeBSD, etc.).
```go
//go:build !darwin && !linux

package fs

import (
	"os"
	"time"
)

// GetFileTimes is a portable fallback that returns modtime for both attributes.
func GetFileTimes(path string) (time.Time, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
}
```

---

### Step 2: Update the `Note` Model in `internal/format/markdown/note.go`

1.  **Remove timestamp fields** from the `Note` struct:
    *   Delete the `CreatedAt` and `UpdatedAt` fields from the struct.
    *   Keep: `MnemonicNoteID`, `Title`, `Slug`, `Tags`, `Summary`, `Aliases`, `Type`, and `Body`.
2.  **Remove timestamp parsing** from `populateFromRaw`:
    *   Delete the lines reading `created_at` and `updated_at`.
    *   Delete the `noteTimeField` helper function from `note.go`.
3.  **Implement dynamic fallback resolvers** directly on the `Note` struct:
    *   `GetOrDeriveTitle(relPath string) string`:
        1.  If `n.Title != ""` is already present, return it.
        2.  Otherwise, call `ExtractH1Title(n.Body)` (which is defined in `h1_title.go` in the same package).
        3.  If that is also empty, use the filename from `relPath` (e.g. `filepath.Base(relPath)` with the `.md` extension stripped).
    *   `GetOrDeriveSlug(relPath string) string`:
        1.  If `n.Slug != ""` is present, return it.
        2.  Otherwise, call `slug.Slugify(...)` on the derived title obtained from `GetOrDeriveTitle(relPath)`.
        3.  If `slug.Slugify` fails (returns an error), implement a local cleanup fallback that lowercases the filename and replaces any non-alphanumeric characters with `-`.

---

### Step 3: Update Note Rendering in `internal/format/markdown/render.go`

Make sure that when `RenderNote()` is called, we do not write empty or missing values into the YAML frontmatter.

1.  Remove `"created_at"` and `"updated_at"` from the `canonicalFrontmatterKeys` map.
2.  Add `"created_at"` and `"updated_at"` to the `removedFrontmatterKeys` map (this ensures that even if legacy files contain them, they are stripped when edited/written).
3.  Update `renderCanonicalFrontmatter` so it checks if properties are set before appending them to the block.
    *   Example:
        ```go
        if note.Title != "" {
            pairs = append(pairs, struct { key string; value any }{"title", note.Title})
        }
        if slug != "" {
            pairs = append(pairs, struct { key string; value any }{"slug", slug})
        }
        ```
    *   Completely remove references to `note.CreatedAt` and `note.UpdatedAt` inside this function.

---

### Step 4: Update the Storage Layer (`internal/store/markdownstore/store.go`)

Now we must adjust the code that reads and lists files so it fetches timestamps and derived attributes.

1.  **Modify `ShowResult`**:
    Add `UpdatedAt time.Time` to the struct:
    ```go
    type ShowResult struct {
        ResolvedNote
        ContentHash string `json:"content_hash"`
        RawMarkdown []byte `json:"-"`
        UpdatedAt   time.Time // System-derived modification timestamp
    }
    ```
2.  **Update `Show()`**:
    *   Call `os.Stat()` on the file's absolute path.
    *   Retrieve the modification time using `.ModTime().UTC()`.
    *   Populate the new `UpdatedAt` field in `ShowResult`.
3.  **Update `List()`**:
    *   When constructing the list of notes via `parallel.MapIndexed`, read each file’s metadata (`os.Stat`).
    *   Determine the updated time: use `note.UpdatedAt` (Wait, we removed it from Note! Use `st.ModTime().UTC()`).
    *   Set `Slug` to `note.GetOrDeriveSlug(relPath)` and `Title` to `note.GetOrDeriveTitle(relPath)`.
4.  **Update `hydrateNote()`**:
    *   We want to be minimally intrusive. When `Hydrate` is executed, **only** generate and write `mnemonic_note_id` if it is missing.
    *   Ensure `hydrateNote` no longer generates or populates `title`, `slug`, `created_at`, or `updated_at` on the target note's structural fields.
5.  **Update `Edit()`**:
    *   When returning `EditResult`, set `UpdatedAt` to `now().UTC().Format(time.RFC3339)`. Remove `CreatedAt` from the return struct.

---

### Step 5: Update the Database Indexer (`internal/store/sqliteindex/scan.go`)

The SQLite schema still contains columns `created_at` and `updated_at` (integers representing Unix epochs) to power time-based advanced searches. We must populate them using our hybrid resolution rules when rebuilding the index.

1.  **Update `scanOneNote`**:
    *   Use the newly created platform helper: `btime, mtime, err := fs.GetFileTimes(absPath)`.
    *   When initializing `NoteDoc`, assign the dynamic derived title and slug:
        ```go
        Slug:  note.GetOrDeriveSlug(relPath),
        Title: note.GetOrDeriveTitle(relPath),
        ```
2.  **Update `populateNoteDocData`**:
    *   Accept `btime` and `mtime` as function parameters.
    *   Assign timestamps to `NoteDoc` using the hybrid resolution rules:
        *   `CreatedAt`:
            1.  If YAML had it (you can check if `note.Frontmatter["created_at"]` exists and can be parsed as an integer).
            2.  Else, use the system `btime`.
            3.  Else, fall back to system `mtime`.
        *   `UpdatedAt`:
            1.  If YAML had it (parse `note.Frontmatter["updated_at"]`).
            2.  Else, use the system `mtime`.

---

### Step 6: Cleanup Diagnostics & Doctor Checks

Because timestamps are no longer validated or expected in the frontmatter, we must delete obsolete diagnostics.

1.  **In `internal/service/indexsvc/diagnostics.go`**:
    *   Remove `KindMissingTimestamp` and `KindInvalidTimestamp` from the `DiagnosticKind` declarations.
    *   Remove these keys from the `validDiagnosticKinds` map.
    *   Delete the `checkTimestamps` helper method and its call inside `checkContentFields`.
    *   In `checkNoteFields`, calculate `slug := note.GetOrDeriveSlug(path)` and `title := note.GetOrDeriveTitle(path)` before running validation rules.
2.  **In `internal/adapter/cli/project_doctor.go`**:
    *   Remove references to `missing_timestamp` and `invalid_timestamp` from the CLI help messages.
3.  **In `internal/service/indexsvc/service.go`**:
    *   Update `countNoteIDs` to resolve the derived slug using `note.GetOrDeriveSlug(rel)` so that `doctor` performs duplicate checking using the correct derived names.

---

## 4. Testing Plan (Instructions for Verification)

To verify that everything is running correctly, execute these tests and write new test scenarios:

1.  **Parser Verification (`internal/format/markdown/note_test.go`)**:
    *   *Add test:* Pass a file containing only `mnemonic_note_id` in its frontmatter. Verify that `ParseNote()` parses it without error.
    *   *Add test:* Call `GetOrDeriveTitle()` on a note without a YAML title but with a `# My Title` H1 header. Assert it returns `My Title`.
    *   *Add test:* Call `GetOrDeriveTitle()` on a note without any header and verify it extracts the filename.
2.  **Rendereer Verification (`internal/format/markdown/render_test.go`)**:
    *   *Add test:* Render a note with only `mnemonic_note_id`. Assert that the resulting string contains no `created_at`, `updated_at`, `title`, or `slug` parameters in its frontmatter block.
3.  **Platform Verification (`internal/platform/fs/times_test.go`)**:
    *   Write a unit test that creates a temporary file on disk, waits a brief moment (e.g. 50ms), modifies it, and asserts that `GetFileTimes()` returns `birthTime` <= `modTime` correctly on the target operating systems.
4.  **Full Rebuild Integration Verification (`internal/store/sqliteindex/rebuild_test.go`)**:
    *   Ensure that running `Rebuild()` successfully populates the SQLite tables `notes` with the extracted file timestamps and that searches using relative intervals like `--created-since 1h` correctly find the indexed files.
