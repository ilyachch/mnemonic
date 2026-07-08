# Изменения в ветке `simplification-frontmatter` относительно `main`

## Список измененных файлов:
- `AGENTS.md`
- `README.cli.md`
- `README.md`
- `go.mod`
- `internal/adapter/cli/notes_mutation_output.go`
- `internal/adapter/cli/notes_show.go`
- `internal/adapter/cli/project_doctor.go`
- `internal/adapter/stdio/tools.go`
- `internal/format/markdown/errors.go`
- `internal/format/markdown/note.go`
- `internal/format/markdown/render.go`
- `internal/platform/fs/times_darwin.go`
- `internal/platform/fs/times_fallback.go`
- `internal/platform/fs/times_linux.go`
- `internal/service/indexsvc/diagnostics.go`
- `internal/service/indexsvc/service.go`
- `internal/service/notesvc/service.go`
- `internal/store/markdownstore/store.go`
- `internal/store/sqliteindex/scan.go`

---

## Файл: `AGENTS.md`

```diff
diff --git a/AGENTS.md b/AGENTS.md
index 27e8f63..0692cd1 100644
--- a/AGENTS.md
+++ b/AGENTS.md
@@ -89,6 +89,16 @@ Errors must be wrapped in the `apperr.Error` struct to return the correct exit c
 - It does **not** use version numbers, `PRAGMA user_version`, or any migration-like mechanisms.
 - The index is a disposable derived artifact — incompatibility is resolved by an explicit `mnemonic project reindex`, never automatically.

+### 7. Minimal Frontmatter Model
+
+The `Note` struct (`internal/format/markdown/note.go`) does **not** carry `CreatedAt`/`UpdatedAt` fields. Timestamps are resolved at runtime:
+
+- **Indexer** (`sqliteindex/scan.go`): uses `fs.GetFileTimes` to obtain `btime`/`mtime`. `CreatedAt` = YAML `frontmatter["created_at"]` if present, else `btime`, else `mtime`. `UpdatedAt` = YAML `frontmatter["updated_at"]` if present, else `mtime`.
+- **read_notes / show**: `ShowResult.UpdatedAt` (system mtime) is the fallback when YAML timestamps are absent.
+- **RenderNote**: omits empty optional fields; `created_at`/`updated_at` are in `removedFrontmatterKeys` and are stripped from legacy files on write.
+- **Hydrate**: only writes `mnemonic_note_id`; `title` and `slug` are derived dynamically via `Note.GetOrDeriveTitle(relPath)` and `Note.GetOrDeriveSlug(relPath)`.
+- **Diagnostics**: `KindMissingTimestamp` and `KindInvalidTimestamp` no longer exist.
+
 ---

 ## Guidelines for Extending Code
```

## Файл: `README.cli.md`

```diff
diff --git a/README.cli.md b/README.cli.md
index 93a6a85..c7edc48 100644
--- a/README.cli.md
+++ b/README.cli.md
@@ -485,7 +485,7 @@ mnemonic project doctor [PROJECT] [flags]
 ```
       --all            run doctor across all active projects
   -h, --help           help for doctor
-      --kind strings   filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, missing_timestamp, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)
+      --kind strings   filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)
 ```

 ### Options inherited from parent commands
```

## Файл: `README.md`

```diff
diff --git a/README.md b/README.md
index 82c4b17..3e0b4a7 100644
--- a/README.md
+++ b/README.md
@@ -9,8 +9,8 @@ Notes are indexed into a local SQLite database with FTS5 (Full-Text Search) for
 ## Features

 - **Local-first Architecture**: Plain Markdown files and an SQLite index stored in the user's workspace.
-- **Markdown Frontmatter**: Every note carries YAML frontmatter with `mnemonic_note_id`, `slug`, `title`, `tags`, `summary`, `created_at`, `updated_at`, `type`, and `aliases`.
-- **Unix Integer Timestamps**: `created_at` and `updated_at` are stored and returned as Unix epoch seconds (e.g. `1741737600`).
+- **Markdown Frontmatter**: Every note carries YAML frontmatter with a required `mnemonic_note_id`. Optional fields (`slug`, `title`, `tags`, `summary`, `type`, `aliases`) are written when present; `title` and `slug` are dynamically derived from the H1 heading or filename at runtime when absent.
+- **Hybrid Timestamp Resolution**: `created_at` and `updated_at` are no longer stored in frontmatter by default. The SQLite index resolves them at rebuild time: YAML values take precedence if present (legacy files), otherwise the file's filesystem birth time (`btime`) and modification time (`mtime`) are used. The MCP `read_notes` tool surfaces `updated_at` from the filesystem mtime when YAML is absent.
 - **Slug-based Wiki-Links**: `[[target-slug]]` and `[[target-slug|Display Label]]` syntax for bidirectional linking. Standard markdown links are also supported: `[Label](target-slug.md)`.
 - **Inline Metadata**:
   - Inline hashtags (`#tag`) extracted alongside frontmatter tags.
@@ -20,7 +20,7 @@ Notes are indexed into a local SQLite database with FTS5 (Full-Text Search) for
 - **Graph-aware Reranking**: Top results are multiplicatively boosted based on link connections to higher-ranked documents.
 - **Related Notes**: Opt-in per-query retrieval of backlinks and forward links with `relation_type`, `source_kind`, and `direction`.
 - **Batch Read**: Read multiple notes in a single `read_notes` call by providing an array of identifiers (note_id, slug, path, or title). Optional field selection controls payload size.
-- **Repository Diagnostics**: `diagnose_notes` scans for invalid frontmatter, missing required fields, missing/invalid timestamps, duplicate slugs/aliases, unresolved or ambiguous wiki-links, and empty bodies. Optionally resolves broken links via search and suggests candidate targets.
+- **Repository Diagnostics**: `diagnose_notes` scans for invalid frontmatter, missing required fields, missing summary, duplicate slugs/aliases, unresolved or ambiguous wiki-links, and empty bodies. Optionally resolves broken links via search and suggests candidate targets.
 - **MCP Transport**: stdio and HTTP/SSE transport with optional Bearer token authentication.

 ---
@@ -37,7 +37,7 @@ go install ./cmd/mnemonic

 ## Note Format

-Every note is a Markdown file with YAML frontmatter. Timestamps are Unix epoch seconds as integers.
+Every note is a Markdown file with YAML frontmatter. The only strictly required field is `mnemonic_note_id`. All other fields (`title`, `slug`, `tags`, `summary`, `type`, `aliases`) are optional — when absent, `title` and `slug` are derived at runtime from the first H1 heading or the filename. Timestamps (`created_at`, `updated_at`) are no longer written to frontmatter; the index resolves them from the filesystem.

 ```markdown
 ---
@@ -48,8 +48,6 @@ tags:
   - project
   - draft
 summary: A brief description of the note content.
-created_at: 1741737600
-updated_at: 1741824000
 type: note
 aliases:
   - intro-note
@@ -345,7 +343,7 @@ Scan for metadata issues, broken links, and content problems.
 | `cursor` | `int` | Zero-based page offset (>= 0) |
 | `include_suggestions` | `bool` | Resolve broken links via search |

-Diagnostic kinds: `invalid_frontmatter`, `missing_required_field`, `missing_summary`, `missing_timestamp`, `invalid_timestamp`, `duplicate_slug`, `duplicate_alias`, `unresolved_link`, `ambiguous_link`, `empty_body`.
+Diagnostic kinds: `invalid_frontmatter`, `missing_required_field`, `missing_summary`, `duplicate_slug`, `duplicate_alias`, `unresolved_link`, `ambiguous_link`, `empty_body`.

 ### `list_notes`
 List all notes with pagination (`limit`, `cursor`).
```

## Файл: `go.mod`

```diff
diff --git a/go.mod b/go.mod
index 8d70620..4380af7 100644
--- a/go.mod
+++ b/go.mod
@@ -13,6 +13,7 @@ require (
 	github.com/spf13/cobra v1.10.2
 	github.com/stretchr/testify v1.11.1
 	github.com/yuin/goldmark v1.7.13
+	golang.org/x/sys v0.40.0
 	gopkg.in/yaml.v3 v3.0.1
 	modernc.org/sqlite v1.37.1
 )
@@ -38,7 +39,6 @@ require (
 	go.yaml.in/yaml/v3 v3.0.4 // indirect
 	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
 	golang.org/x/oauth2 v0.34.0 // indirect
-	golang.org/x/sys v0.40.0 // indirect
 	gopkg.in/yaml.v2 v2.3.0 // indirect
 	modernc.org/libc v1.65.7 // indirect
 	modernc.org/mathutil v1.7.1 // indirect
```

## Файл: `internal/adapter/cli/notes_mutation_output.go`

```go
diff --git a/internal/adapter/cli/notes_mutation_output.go b/internal/adapter/cli/notes_mutation_output.go
index d2948fc..f247251 100644
--- a/internal/adapter/cli/notes_mutation_output.go
+++ b/internal/adapter/cli/notes_mutation_output.go
@@ -16,7 +16,6 @@ type notesEditOutput struct {
 	Slug        string `json:"slug"`
 	Path        string `json:"path"`
 	ContentHash string `json:"content_hash"`
-	CreatedAt   string `json:"created_at"`
 	UpdatedAt   string `json:"updated_at"`
 	IndexStatus string `json:"index_status"`
 	IndexError  string `json:"index_error,omitempty"`
@@ -47,7 +46,6 @@ func newNotesEditOutput(result notesvc.EditResult) notesEditOutput {
 		Slug:        result.Slug,
 		Path:        result.Path,
 		ContentHash: result.ContentHash,
-		CreatedAt:   result.CreatedAt,
 		UpdatedAt:   result.UpdatedAt,
 		IndexStatus: result.IndexStatus,
 		IndexError:  result.IndexError,
```

## Файл: `internal/adapter/cli/notes_show.go`

```go
diff --git a/internal/adapter/cli/notes_show.go b/internal/adapter/cli/notes_show.go
index 1a84a58..ab63380 100644
--- a/internal/adapter/cli/notes_show.go
+++ b/internal/adapter/cli/notes_show.go
@@ -12,30 +12,30 @@ func newNotesShowCommand() *cobra.Command {
 		Short: "Show a note",
 		Args:  cobra.ExactArgs(1),
 		RunE: func(cmd *cobra.Command, args []string) error {
-		runtime, err := runtimeAppForSelectedProject(cmd)
-		if err != nil {
-			return err
-		}
+			runtime, err := runtimeAppForSelectedProject(cmd)
+			if err != nil {
+				return err
+			}

-		resolved, err := runtime.Services.Notes.Show(args[0])
-		if err != nil {
-			return err
-		}
+			resolved, err := runtime.Services.Notes.Show(args[0])
+			if err != nil {
+				return err
+			}

-		output := notesShowOutput{
-			Note: notesShowItem{
-				NoteID:      resolved.Note.MnemonicNoteID,
-				Slug:        resolved.Note.EffectiveSlug(),
-				Title:       resolved.Note.Title,
-				Path:        resolved.Path,
-				Frontmatter: resolved.Note.Frontmatter,
-				Body:        string(resolved.Note.Body),
-				ContentHash: resolved.ContentHash,
-				UpdatedAt:   resolved.Note.UpdatedAt.UTC().Format(time.RFC3339),
-			},
-		}
-		return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), string(resolved.RawMarkdown), output)
-	},
+			output := notesShowOutput{
+				Note: notesShowItem{
+					NoteID:      resolved.Note.MnemonicNoteID,
+					Slug:        resolved.Note.EffectiveSlug(),
+					Title:       resolved.Note.Title,
+					Path:        resolved.Path,
+					Frontmatter: resolved.Note.Frontmatter,
+					Body:        string(resolved.Note.Body),
+					ContentHash: resolved.ContentHash,
+					UpdatedAt:   resolved.UpdatedAt.UTC().Format(time.RFC3339),
+				},
+			}
+			return PrintOutput(cmd.OutOrStdout(), jsonOutputEnabled(cmd), string(resolved.RawMarkdown), output)
+		},
 	}
 }
```

## Файл: `internal/adapter/cli/project_doctor.go`

```go
diff --git a/internal/adapter/cli/project_doctor.go b/internal/adapter/cli/project_doctor.go
index f9eb534..54250ae 100644
--- a/internal/adapter/cli/project_doctor.go
+++ b/internal/adapter/cli/project_doctor.go
@@ -21,7 +21,7 @@ func newProjectDoctorCommand() *cobra.Command {
 		RunE:              runProjectDoctor,
 	}
 	cmd.Flags().Bool("all", false, "run doctor across all active projects")
-	cmd.Flags().StringSlice("kind", nil, "filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, missing_timestamp, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)")
+	cmd.Flags().StringSlice("kind", nil, "filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)")
 	return cmd
 }
```

## Файл: `internal/adapter/stdio/tools.go`

```go
diff --git a/internal/adapter/stdio/tools.go b/internal/adapter/stdio/tools.go
index 0a3f355..7af99db 100644
--- a/internal/adapter/stdio/tools.go
+++ b/internal/adapter/stdio/tools.go
@@ -63,7 +63,7 @@ Returns paginated diagnostic issues. Set include_suggestions to true to receive
 Use this tool periodically to verify repository integrity after bulk changes.

 Parameters:
-- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "missing_timestamp", "invalid_timestamp", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
+- kinds ([]string, optional): filter by diagnostic kind. Valid values: "invalid_frontmatter", "missing_required_field", "missing_summary", "duplicate_slug", "duplicate_alias", "unresolved_link", "ambiguous_link", "empty_body".
 - limit (int, optional): maximum issues per page (default 50, max 200).
 - cursor (int, optional): zero-based page offset.
 - include_suggestions (bool, optional): resolve broken links via search and include candidate notes.`
@@ -195,7 +195,6 @@ type EditNoteOutput struct {
 	Slug        string `json:"slug"`
 	Path        string `json:"path"`
 	ContentHash string `json:"content_hash"`
-	CreatedAt   string `json:"created_at"`
 	UpdatedAt   string `json:"updated_at"`
 	IndexStatus string `json:"index_status"`
 	IndexError  string `json:"index_error,omitempty"`
@@ -506,7 +505,6 @@ func RegisterEditNote(server *sdkmcp.Server, deps Dependencies) {
 			Slug:        edited.Slug,
 			Path:        edited.Path,
 			ContentHash: edited.ContentHash,
-			CreatedAt:   edited.CreatedAt,
 			UpdatedAt:   edited.UpdatedAt,
 			IndexStatus: edited.IndexStatus,
 			IndexError:  edited.IndexError,
@@ -837,16 +835,40 @@ func setReadNotesExtraFields(note *ReadNotesNote, fields map[string]bool, resolv
 		h := resolved.ContentHash
 		note.ContentHash = &h
 	}
-	if fields["created_at"] && !resolved.Note.CreatedAt.IsZero() {
-		ca := resolved.Note.CreatedAt.Unix()
-		note.CreatedAt = &ca
+	if fields["created_at"] {
+		if ca, ok := frontmatterUnixTime(resolved.Note.Frontmatter, "created_at"); ok {
+			note.CreatedAt = &ca
+		}
 	}
-	if fields["updated_at"] && !resolved.Note.UpdatedAt.IsZero() {
-		ua := resolved.Note.UpdatedAt.Unix()
-		note.UpdatedAt = &ua
+	if fields["updated_at"] {
+		if ua, ok := frontmatterUnixTime(resolved.Note.Frontmatter, "updated_at"); ok {
+			note.UpdatedAt = &ua
+		} else if !resolved.UpdatedAt.IsZero() {
+			ua := resolved.UpdatedAt.Unix()
+			note.UpdatedAt = &ua
+		}
 	}
 }

+// frontmatterUnixTime extracts a Unix epoch integer from raw frontmatter.
+// YAML unmarshals integers as `int` (or `int64`); both are accepted.
+func frontmatterUnixTime(fm map[string]any, key string) (int64, bool) {
+	if fm == nil {
+		return 0, false
+	}
+	value, ok := fm[key]
+	if !ok || value == nil {
+		return 0, false
+	}
+	switch v := value.(type) {
+	case int:
+		return int64(v), true
+	case int64:
+		return v, true
+	}
+	return 0, false
+}
+
 func buildReadNotesNote(fields map[string]bool, resolved notesvc.ShowResult, maxBodyChars int) ReadNotesNote {
 	note := ReadNotesNote{
 		NoteID: resolved.Note.MnemonicNoteID,
```

## Файл: `internal/format/markdown/errors.go`

```go
diff --git a/internal/format/markdown/errors.go b/internal/format/markdown/errors.go
index 1d1c551..c82e71a 100644
--- a/internal/format/markdown/errors.go
+++ b/internal/format/markdown/errors.go
@@ -23,7 +23,6 @@ const (
 	FieldErrKindInvalidType       = "invalid_type"
 	FieldErrKindInvalidString     = "invalid_string"
 	FieldErrKindInvalidStringList = "invalid_string_list"
-	FieldErrKindInvalidTimestamp  = "invalid_timestamp"
 )

 func newStringFieldError(key string, err error) error {
@@ -33,7 +32,3 @@ func newStringFieldError(key string, err error) error {
 func newStringSliceFieldError(key string, err error) error {
 	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidStringList, Err: err}
 }
-
-func newTimeFieldError(key string, err error) error {
-	return &FrontmatterFieldError{Field: key, Kind: FieldErrKindInvalidTimestamp, Err: err}
-}
```

## Файл: `internal/format/markdown/note.go`

```go
diff --git a/internal/format/markdown/note.go b/internal/format/markdown/note.go
index 3e0c93f..b416ad3 100644
--- a/internal/format/markdown/note.go
+++ b/internal/format/markdown/note.go
@@ -2,8 +2,10 @@ package markdown

 import (
 	"fmt"
-	"time"
+	"path/filepath"
+	"strings"

+	"github.com/ilyachch/mnemonic/internal/domain/slug"
 	"gopkg.in/yaml.v3"
 )

@@ -16,8 +18,6 @@ type Note struct {
 	Tags           []string
 	Summary        string
 	Aliases        []string
-	CreatedAt      time.Time
-	UpdatedAt      time.Time
 	Type           string
 	Body           []byte
 }
@@ -78,12 +78,6 @@ func (n *Note) populateFromRaw(raw map[string]any) error {
 	if n.Aliases, errField = noteStringSliceField(raw, "aliases"); errField != nil {
 		return errField
 	}
-	if n.CreatedAt, errField = noteTimeField(raw, "created_at"); errField != nil {
-		return errField
-	}
-	if n.UpdatedAt, errField = noteTimeField(raw, "updated_at"); errField != nil {
-		return errField
-	}
 	if n.Type, errField = noteStringField(raw, "type"); errField != nil {
 		return errField
 	}
@@ -129,18 +123,48 @@ func noteStringSliceField(raw map[string]any, key string) ([]string, error) {
 	}
 }

-func noteTimeField(raw map[string]any, key string) (time.Time, error) {
-	value, ok := raw[key]
-	if !ok || value == nil {
-		return time.Time{}, nil
+// GetOrDeriveTitle returns the note's title, deriving it dynamically when the
+// YAML title is absent. Derivation order: explicit Title field, then the first
+// H1 heading in the body, then the file's base name (without extension).
+func (n Note) GetOrDeriveTitle(relPath string) string {
+	if n.Title != "" {
+		return n.Title
 	}
-
-	switch typed := value.(type) {
-	case int:
-		return time.Unix(int64(typed), 0).UTC(), nil
-	case int64:
-		return time.Unix(typed, 0).UTC(), nil
-	default:
-		return time.Time{}, newTimeFieldError(key, fmt.Errorf("must be a Unix timestamp as an integer, got %T", value))
+	if title := ExtractH1Title(n.Body); title != "" {
+		return title
 	}
+	return strings.TrimSuffix(filepath.Base(relPath), ".md")
+}
+
+// GetOrDeriveSlug returns the note's slug, deriving it dynamically when the
+// YAML slug is absent. Derivation uses Slugify on the derived title; if that
+// fails (e.g. non-ASCII input), a local cleanup fallback lowercases the
+// filename and replaces any non-alphanumeric characters with "-".
+func (n Note) GetOrDeriveSlug(relPath string) string {
+	if n.Slug != "" {
+		return n.Slug
+	}
+	derived, err := slug.Slugify(n.GetOrDeriveTitle(relPath))
+	if err == nil && derived != "" {
+		return derived
+	}
+	return slugFromBaseName(relPath)
+}
+
+// slugFromBaseName produces a best-effort slug from a file's base name by
+// stripping the extension and lowercasing. Used when Slugify rejects the
+// title (e.g. non-ASCII input).
+func slugFromBaseName(relPath string) string {
+	base := strings.TrimSuffix(filepath.Base(relPath), ".md")
+	base = strings.ToLower(base)
+	var b strings.Builder
+	for _, r := range base {
+		switch {
+		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
+			b.WriteRune(r)
+		default:
+			b.WriteByte('-')
+		}
+	}
+	return strings.Trim(b.String(), "-")
 }
```

## Файл: `internal/format/markdown/render.go`

```go
diff --git a/internal/format/markdown/render.go b/internal/format/markdown/render.go
index 7a6aea4..f3a0391 100644
--- a/internal/format/markdown/render.go
+++ b/internal/format/markdown/render.go
@@ -16,13 +16,13 @@ var canonicalFrontmatterKeys = map[string]struct{}{
 	"tags":             {},
 	"summary":          {},
 	"aliases":          {},
-	"created_at":       {},
-	"updated_at":       {},
 	"type":             {},
 }

 var removedFrontmatterKeys = map[string]struct{}{
-	"permalink": {},
+	"permalink":  {},
+	"created_at": {},
+	"updated_at": {},
 }

 // RenderNote serializes a note into canonical YAML frontmatter plus body.
@@ -32,9 +32,6 @@ func RenderNote(note Note) ([]byte, error) {
 	}

 	slug := note.EffectiveSlug()
-	if slug == "" {
-		return nil, errors.New("slug is required")
-	}

 	var buf bytes.Buffer
 	buf.WriteString("---\n")
@@ -63,10 +60,12 @@ func renderCanonicalFrontmatter(buf *bytes.Buffer, note Note, slug string) error
 			value any
 		}{"title", note.Title})
 	}
-	pairs = append(pairs, struct {
-		key   string
-		value any
-	}{"slug", slug})
+	if slug != "" {
+		pairs = append(pairs, struct {
+			key   string
+			value any
+		}{"slug", slug})
+	}
 	if len(note.Tags) > 0 {
 		pairs = append(pairs, struct {
 			key   string
@@ -85,18 +84,6 @@ func renderCanonicalFrontmatter(buf *bytes.Buffer, note Note, slug string) error
 			value any
 		}{"aliases", note.Aliases})
 	}
-	if !note.CreatedAt.IsZero() {
-		pairs = append(pairs, struct {
-			key   string
-			value any
-		}{"created_at", note.CreatedAt.Unix()})
-	}
-	if !note.UpdatedAt.IsZero() {
-		pairs = append(pairs, struct {
-			key   string
-			value any
-		}{"updated_at", note.UpdatedAt.Unix()})
-	}
 	if note.Type != "" {
 		pairs = append(pairs, struct {
 			key   string
```

## Файл: `internal/platform/fs/times_darwin.go`

```go
diff --git a/internal/platform/fs/times_darwin.go b/internal/platform/fs/times_darwin.go
new file mode 100644
index 0000000..c13b6a0
--- /dev/null
+++ b/internal/platform/fs/times_darwin.go
@@ -0,0 +1,26 @@
+//go:build darwin
+
+package fs
+
+import (
+	"os"
+	"syscall"
+	"time"
+)
+
+// GetFileTimes returns (birthTime, modTime, error) for Darwin.
+func GetFileTimes(path string) (time.Time, time.Time, error) {
+	fi, err := os.Stat(path)
+	if err != nil {
+		return time.Time{}, time.Time{}, err
+	}
+	stat, ok := fi.Sys().(*syscall.Stat_t)
+	if !ok {
+		// Fall back to modtime if sys conversion fails
+		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
+	}
+
+	// Birthtimespec represents the creation time on macOS.
+	birthTime := time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec).UTC()
+	return birthTime, fi.ModTime().UTC(), nil
+}
```

## Файл: `internal/platform/fs/times_fallback.go`

```go
diff --git a/internal/platform/fs/times_fallback.go b/internal/platform/fs/times_fallback.go
new file mode 100644
index 0000000..4238edc
--- /dev/null
+++ b/internal/platform/fs/times_fallback.go
@@ -0,0 +1,17 @@
+//go:build !darwin && !linux
+
+package fs
+
+import (
+	"os"
+	"time"
+)
+
+// GetFileTimes is a portable fallback that returns modtime for both attributes.
+func GetFileTimes(path string) (time.Time, time.Time, error) {
+	fi, err := os.Stat(path)
+	if err != nil {
+		return time.Time{}, time.Time{}, err
+	}
+	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
+}
```

## Файл: `internal/platform/fs/times_linux.go`

```go
diff --git a/internal/platform/fs/times_linux.go b/internal/platform/fs/times_linux.go
new file mode 100644
index 0000000..078d858
--- /dev/null
+++ b/internal/platform/fs/times_linux.go
@@ -0,0 +1,34 @@
+//go:build linux
+
+package fs
+
+import (
+	"os"
+	"time"
+
+	"golang.org/x/sys/unix"
+)
+
+// GetFileTimes returns (birthTime, modTime, error) for Linux.
+func GetFileTimes(path string) (time.Time, time.Time, error) {
+	fi, err := os.Stat(path)
+	if err != nil {
+		return time.Time{}, time.Time{}, err
+	}
+
+	var statx unix.Statx_t
+	// AT_FDCWD means relative paths are evaluated relative to current directory
+	if err = unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &statx); err != nil {
+		// Fall back to standard ModTime if statx fails (e.g. unsupported filesystem or older kernel).
+		// Intentionally swallow statx errors: btime is best-effort here.
+		return fi.ModTime().UTC(), fi.ModTime().UTC(), nil //nolint:nilerr // intentional fallback
+	}
+
+	// Verify that the kernel actually populated the Btime attribute
+	if (statx.Mask & unix.STATX_BTIME) != 0 {
+		birthTime := time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)).UTC()
+		return birthTime, fi.ModTime().UTC(), nil
+	}
+
+	return fi.ModTime().UTC(), fi.ModTime().UTC(), nil
+}
```

## Файл: `internal/service/indexsvc/diagnostics.go`

```go
diff --git a/internal/service/indexsvc/diagnostics.go b/internal/service/indexsvc/diagnostics.go
index cbec34b..9a14845 100644
--- a/internal/service/indexsvc/diagnostics.go
+++ b/internal/service/indexsvc/diagnostics.go
@@ -22,8 +22,6 @@ const (
 	KindInvalidFrontmatter   DiagnosticKind = "invalid_frontmatter"
 	KindMissingRequiredField DiagnosticKind = "missing_required_field"
 	KindMissingSummary       DiagnosticKind = "missing_summary"
-	KindMissingTimestamp     DiagnosticKind = "missing_timestamp"
-	KindInvalidTimestamp     DiagnosticKind = "invalid_timestamp"
 	KindDuplicateSlug        DiagnosticKind = "duplicate_slug"
 	KindDuplicateAlias       DiagnosticKind = "duplicate_alias"
 	KindUnresolvedLink       DiagnosticKind = "unresolved_link"
@@ -134,8 +132,6 @@ var validDiagnosticKinds = map[DiagnosticKind]bool{
 	KindInvalidFrontmatter:   true,
 	KindMissingRequiredField: true,
 	KindMissingSummary:       true,
-	KindMissingTimestamp:     true,
-	KindInvalidTimestamp:     true,
 	KindDuplicateSlug:        true,
 	KindDuplicateAlias:       true,
 	KindUnresolvedLink:       true,
@@ -226,7 +222,7 @@ func (s Service) checkOneNote(root, rel string, filter kindFilter) (issues []Dia
 	}

 	issues = s.checkNoteFields(note, rel, filter)
-	if s := note.EffectiveSlug(); s != "" {
+	if s := note.GetOrDeriveSlug(rel); s != "" {
 		slugs = append(slugs, s)
 	}
 	for _, a := range note.Aliases {
@@ -241,18 +237,10 @@ func (s Service) classifyParseError(parseErr error, rel string, filter kindFilte
 	var issues []DiagnosticIssue
 	var fieldErr *markdown.FrontmatterFieldError
 	if errors.As(parseErr, &fieldErr) {
-		if fieldErr.Kind == markdown.FieldErrKindInvalidTimestamp {
-			if filter.include(KindInvalidTimestamp) {
-				issues = append(issues, DiagnosticIssue{
-					Kind: KindInvalidTimestamp, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
-				})
-			}
-		} else {
-			if filter.include(KindInvalidFrontmatter) {
-				issues = append(issues, DiagnosticIssue{
-					Kind: KindInvalidFrontmatter, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
-				})
-			}
+		if filter.include(KindInvalidFrontmatter) {
+			issues = append(issues, DiagnosticIssue{
+				Kind: KindInvalidFrontmatter, Path: rel, Field: fieldErr.Field, Detail: parseErr.Error(),
+			})
 		}
 	} else {
 		if filter.include(KindInvalidFrontmatter) {
@@ -267,9 +255,10 @@ func (s Service) classifyParseError(parseErr error, rel string, filter kindFilte
 func (s Service) checkNoteFields(note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
 	issues := make([]DiagnosticIssue, 0, 7)
 	noteID := note.MnemonicNoteID
-	slug := note.EffectiveSlug()
+	slug := note.GetOrDeriveSlug(path)
+	title := note.GetOrDeriveTitle(path)

-	issues = append(issues, s.checkRequiredFields(noteID, slug, note.Title, path, filter)...)
+	issues = append(issues, s.checkRequiredFields(noteID, slug, title, path, filter)...)
 	issues = append(issues, s.checkContentFields(noteID, slug, note, path, filter)...)
 	return issues
 }
@@ -304,7 +293,6 @@ func (s Service) checkContentFields(noteID, slug string, note markdown.Note, pat
 			Kind: KindMissingSummary, NoteID: noteID, Slug: slug, Path: path,
 		})
 	}
-	issues = append(issues, s.checkTimestamps(noteID, slug, note, path, filter)...)
 	if len(strings.TrimSpace(string(note.Body))) == 0 && filter.include(KindEmptyBody) {
 		issues = append(issues, DiagnosticIssue{
 			Kind: KindEmptyBody, NoteID: noteID, Slug: slug, Path: path,
@@ -313,42 +301,6 @@ func (s Service) checkContentFields(noteID, slug string, note markdown.Note, pat
 	return issues
 }

-func (s Service) checkTimestamps(noteID, slug string, note markdown.Note, path string, filter kindFilter) []DiagnosticIssue {
-	var issues []DiagnosticIssue
-	addMissing := func(field string) {
-		if filter.include(KindMissingTimestamp) {
-			issues = append(issues, DiagnosticIssue{
-				Kind: KindMissingTimestamp, NoteID: noteID, Slug: slug, Path: path,
-				Field:  field,
-				Detail: "missing " + field,
-			})
-		}
-	}
-	addInvalid := func(detail, field string) {
-		if filter.include(KindInvalidTimestamp) {
-			issues = append(issues, DiagnosticIssue{
-				Kind: KindInvalidTimestamp, NoteID: noteID, Slug: slug, Path: path,
-				Field:  field,
-				Detail: detail,
-			})
-		}
-	}
-	if note.CreatedAt.IsZero() {
-		addMissing("created_at")
-	} else if note.CreatedAt.Unix() <= 0 {
-		addInvalid("created_at <= 0", "created_at")
-	}
-	if note.UpdatedAt.IsZero() {
-		addMissing("updated_at")
-	} else if note.UpdatedAt.Unix() <= 0 {
-		addInvalid("updated_at <= 0", "updated_at")
-	}
-	if !note.CreatedAt.IsZero() && !note.UpdatedAt.IsZero() && note.CreatedAt.After(note.UpdatedAt) {
-		addInvalid("created_at > updated_at", "created_at")
-	}
-	return issues
-}
-
 func (s Service) collectDuplicateSlugIssues(
 	seenSlugs map[string][]string, filter kindFilter,
 ) []DiagnosticIssue {
```

## Файл: `internal/service/indexsvc/service.go`

```go
diff --git a/internal/service/indexsvc/service.go b/internal/service/indexsvc/service.go
index df9589d..9bfbec9 100644
--- a/internal/service/indexsvc/service.go
+++ b/internal/service/indexsvc/service.go
@@ -216,7 +216,7 @@ func countNoteIDs(paths []string, root string, trashIgnored *int) (seenUUID, see
 		if note.MnemonicNoteID != "" {
 			seenUUID[note.MnemonicNoteID]++
 		}
-		if slug := note.EffectiveSlug(); slug != "" {
+		if slug := note.GetOrDeriveSlug(rel); slug != "" {
 			seenSlug[slug]++
 		}
 	}
```

## Файл: `internal/service/notesvc/service.go`

```go
diff --git a/internal/service/notesvc/service.go b/internal/service/notesvc/service.go
index 99e2db7..118b31d 100644
--- a/internal/service/notesvc/service.go
+++ b/internal/service/notesvc/service.go
@@ -57,7 +57,6 @@ type EditResult struct {
 	Slug        string `json:"slug"`
 	Path        string `json:"path"`
 	ContentHash string `json:"content_hash"`
-	CreatedAt   string `json:"created_at"`
 	UpdatedAt   string `json:"updated_at"`
 	IndexStatus string `json:"index_status"`
 	IndexError  string `json:"index_error,omitempty"`
@@ -185,7 +184,6 @@ func (s Service) Edit(input EditInput) (EditResult, error) {
 		Slug:        edited.Slug,
 		Path:        edited.Path,
 		ContentHash: edited.ContentHash,
-		CreatedAt:   edited.CreatedAt,
 		UpdatedAt:   edited.UpdatedAt,
 		IndexStatus: "skipped",
 	}
```

## Файл: `internal/store/markdownstore/store.go`

```go
diff --git a/internal/store/markdownstore/store.go b/internal/store/markdownstore/store.go
index 91815e0..3c974f4 100644
--- a/internal/store/markdownstore/store.go
+++ b/internal/store/markdownstore/store.go
@@ -72,7 +72,6 @@ type EditResult struct {
 	Slug        string `json:"slug"`
 	Path        string `json:"path"`
 	ContentHash string `json:"content_hash"`
-	CreatedAt   string `json:"created_at"`
 	UpdatedAt   string `json:"updated_at"`
 }

@@ -112,8 +111,9 @@ type ResolvedNote struct {
 // ShowResult describes a note show response.
 type ShowResult struct {
 	ResolvedNote
-	ContentHash string `json:"content_hash"`
-	RawMarkdown []byte `json:"-"`
+	ContentHash string    `json:"content_hash"`
+	RawMarkdown []byte    `json:"-"`
+	UpdatedAt   time.Time // System-derived modification timestamp
 }

 // Create writes a new markdown note beneath the store root.
@@ -137,23 +137,16 @@ func (s Store) Create(input CreateInput) (CreateResult, error) {
 		return CreateResult{}, err
 	}

-	now := input.Now
-	if now == nil {
-		now = clock.NowUTC
-	}
 	uuidFn := input.UUID
 	if uuidFn == nil {
 		uuidFn = idgen.NewUUID
 	}

-	timestamp := now().UTC()
 	note := markdown.Note{
 		MnemonicNoteID: uuidFn(),
 		Title:          input.Title,
 		Slug:           slugValue,
 		Tags:           dedupeTags(input.Tags),
-		CreatedAt:      timestamp,
-		UpdatedAt:      timestamp,
 		Body:           append([]byte(nil), input.Body...),
 	}

@@ -192,16 +185,11 @@ func applyEditInput(edited *markdown.Note, input EditInput) error {
 	if input.Aliases != nil {
 		edited.Aliases = *input.Aliases
 	}
-	now := input.Now
-	if now == nil {
-		now = clock.NowUTC
-	}
 	if input.HasBody {
 		edited.Body = append([]byte(nil), input.Body...)
 	} else {
 		edited.Body = append(append([]byte(nil), edited.Body...), input.Append...)
 	}
-	edited.UpdatedAt = now().UTC()
 	return nil
 }

@@ -243,6 +231,11 @@ func (s Store) Edit(input EditInput) (EditResult, error) {
 		}
 	}

+	now := input.Now
+	if now == nil {
+		now = clock.NowUTC
+	}
+
 	if err = applyEditInput(&edited, input); err != nil {
 		return EditResult{}, err
 	}
@@ -261,8 +254,7 @@ func (s Store) Edit(input EditInput) (EditResult, error) {
 		Slug:        edited.EffectiveSlug(),
 		Path:        resolved.Path,
 		ContentHash: HashBytes(rendered),
-		CreatedAt:   edited.CreatedAt.UTC().Format(time.RFC3339),
-		UpdatedAt:   edited.UpdatedAt.UTC().Format(time.RFC3339),
+		UpdatedAt:   now().UTC().Format(time.RFC3339),
 	}, nil
 }

@@ -360,16 +352,18 @@ func (s Store) List() ([]NoteSummary, error) {
 		if note.MnemonicNoteID == "" {
 			return NoteSummary{}, fmt.Errorf("note %q is missing mnemonic_note_id", relPath)
 		}
-		if note.UpdatedAt.IsZero() {
-			return NoteSummary{}, fmt.Errorf("note %q is missing updated_at", relPath)
+
+		st, err := os.Stat(absPath)
+		if err != nil {
+			return NoteSummary{}, fmt.Errorf("stat note %q: %w", relPath, err)
 		}

 		return NoteSummary{
 			NoteID:      note.MnemonicNoteID,
-			Slug:        note.EffectiveSlug(),
-			Title:       note.Title,
+			Slug:        note.GetOrDeriveSlug(relPath),
+			Title:       note.GetOrDeriveTitle(relPath),
 			Path:        relPath,
-			UpdatedAt:   note.UpdatedAt.UTC().Format(time.RFC3339),
+			UpdatedAt:   st.ModTime().UTC().Format(time.RFC3339),
 			ContentHash: HashBytes(data),
 		}, nil
 	})
@@ -390,7 +384,8 @@ func (s Store) Show(selector string) (ShowResult, error) {
 		return ShowResult{}, err
 	}

-	data, err := os.ReadFile(filepath.Join(s.rootDir(), filepath.FromSlash(resolved.Path)))
+	absPath := filepath.Join(s.rootDir(), filepath.FromSlash(resolved.Path))
+	data, err := os.ReadFile(absPath)
 	if err != nil {
 		return ShowResult{}, apperr.IO(fmt.Sprintf("read note %q", resolved.Path), err)
 	}
@@ -400,6 +395,11 @@ func (s Store) Show(selector string) (ShowResult, error) {
 		return ShowResult{}, apperr.Corrupted(fmt.Sprintf("parse note %q", resolved.Path), err)
 	}

+	st, err := os.Stat(absPath)
+	if err != nil {
+		return ShowResult{}, apperr.IO(fmt.Sprintf("stat note %q", resolved.Path), err)
+	}
+
 	return ShowResult{
 		ResolvedNote: ResolvedNote{
 			Note: note,
@@ -407,6 +407,7 @@ func (s Store) Show(selector string) (ShowResult, error) {
 		},
 		ContentHash: HashBytes(data),
 		RawMarkdown: append([]byte(nil), data...),
+		UpdatedAt:   st.ModTime().UTC(),
 	}, nil
 }

@@ -566,10 +567,6 @@ func (s Store) Hydrate(input HydrateInput) (HydrateResult, error) {
 		return HydrateResult{}, errors.New("root directory is required")
 	}

-	now := input.Now
-	if now == nil {
-		now = clock.NowUTC
-	}
 	uuidFn := input.UUID
 	if uuidFn == nil {
 		uuidFn = idgen.NewUUID
@@ -592,7 +589,7 @@ func (s Store) Hydrate(input HydrateInput) (HydrateResult, error) {

 	result := HydrateResult{DryRun: input.DryRun}
 	for _, relPath := range relPaths {
-		entry, skip, err := s.hydrateOne(root, relPath, input.DryRun, now, uuidFn)
+		entry, skip, err := s.hydrateOne(root, relPath, input.DryRun, uuidFn)
 		if err != nil {
 			return HydrateResult{}, err
 		}
@@ -607,9 +604,10 @@ func (s Store) Hydrate(input HydrateInput) (HydrateResult, error) {
 }

 // hydrateOne processes a single note file: reads, parses, and either reports
-// it as skipped (when it already has a mnemonic_note_id) or hydrates missing
-// metadata and writes it back (unless dry-run).
-func (s Store) hydrateOne(root, relPath string, dryRun bool, now func() time.Time, uuidFn func() string) (HydratedNote, bool, error) {
+// it as skipped (when it already has a mnemonic_note_id) or hydrates the
+// mnemonic_note_id and writes it back (unless dry-run). Title and slug in
+// the result are derived dynamically for reporting purposes.
+func (s Store) hydrateOne(root, relPath string, dryRun bool, uuidFn func() string) (HydratedNote, bool, error) {
 	absPath := filepath.Join(root, filepath.FromSlash(relPath))
 	data, err := os.ReadFile(absPath)
 	if err != nil {
@@ -625,7 +623,7 @@ func (s Store) hydrateOne(root, relPath string, dryRun bool, now func() time.Tim
 		return HydratedNote{}, true, nil
 	}

-	hydrated, err := s.hydrateNote(relPath, absPath, note, now, uuidFn)
+	hydrated, err := s.hydrateNote(note, uuidFn)
 	if err != nil {
 		return HydratedNote{}, false, fmt.Errorf("hydrate note %q: %w", relPath, err)
 	}
@@ -643,8 +641,8 @@ func (s Store) hydrateOne(root, relPath string, dryRun bool, now func() time.Tim
 	return HydratedNote{
 		Path:   relPath,
 		NoteID: hydrated.note.MnemonicNoteID,
-		Title:  hydrated.note.Title,
-		Slug:   hydrated.note.EffectiveSlug(),
+		Title:  hydrated.note.GetOrDeriveTitle(relPath),
+		Slug:   hydrated.note.GetOrDeriveSlug(relPath),
 	}, false, nil
 }

@@ -707,75 +705,17 @@ type hydratedNote struct {
 	note markdown.Note
 }

-// hydrateNote fills missing canonical fields on a parsed note using the
-// supplied time and UUID providers. It never overwrites fields that already
-// carry a value.
-func (s Store) hydrateNote(relPath, absPath string, note markdown.Note, now func() time.Time, uuidFn func() string) (hydratedNote, error) {
+// hydrateNote fills the mnemonic_note_id on a parsed note when missing. Per
+// the minimal frontmatter model, no other canonical fields (title, slug,
+// timestamps) are written to disk; they are resolved dynamically at runtime.
+func (s Store) hydrateNote(note markdown.Note, uuidFn func() string) (hydratedNote, error) {
 	if note.MnemonicNoteID == "" {
 		note.MnemonicNoteID = uuidFn()
 	}

-	if note.Title == "" {
-		note.Title = deriveNoteTitle(note.Body, relPath)
-	}
-
-	if note.EffectiveSlug() == "" {
-		slugValue, err := slug.Slugify(note.Title)
-		if err != nil || slugValue == "" {
-			slugValue = slugFromBaseName(relPath)
-		}
-		note.Slug = slugValue
-	}
-
-	mtime := fileMtime(absPath, now)
-	if note.CreatedAt.IsZero() {
-		note.CreatedAt = mtime
-	}
-	if note.UpdatedAt.IsZero() {
-		note.UpdatedAt = mtime
-	}
-
 	return hydratedNote{note: note}, nil
 }

-// deriveNoteTitle resolves a title from the note body's first H1 heading,
-// falling back to the file's base name (without extension) when no heading
-// is present.
-func deriveNoteTitle(body []byte, relPath string) string {
-	if title := markdown.ExtractH1Title(body); title != "" {
-		return title
-	}
-	return strings.TrimSuffix(filepath.Base(relPath), ".md")
-}
-
-// slugFromBaseName produces a best-effort slug from a file's base name by
-// stripping the extension and lowercasing. Used when Slugify rejects the
-// title (e.g. non-ASCII input).
-func slugFromBaseName(relPath string) string {
-	base := strings.TrimSuffix(filepath.Base(relPath), ".md")
-	base = strings.ToLower(base)
-	var b strings.Builder
-	for _, r := range base {
-		switch {
-		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
-			b.WriteRune(r)
-		default:
-			b.WriteByte('-')
-		}
-	}
-	return strings.Trim(b.String(), "-")
-}
-
-// fileMtime returns the file's modification time in UTC, falling back to the
-// supplied now() provider when the stat fails or the time is zero.
-func fileMtime(absPath string, now func() time.Time) time.Time {
-	info, err := os.Stat(absPath)
-	if err != nil || info.ModTime().IsZero() {
-		return now().UTC()
-	}
-	return info.ModTime().UTC()
-}
-
 type TrashPathInput struct {
 	RootDir      string
 	OriginalPath string
```

## Файл: `internal/store/sqliteindex/scan.go`

```go
diff --git a/internal/store/sqliteindex/scan.go b/internal/store/sqliteindex/scan.go
index a12afc0..d368223 100644
--- a/internal/store/sqliteindex/scan.go
+++ b/internal/store/sqliteindex/scan.go
@@ -7,8 +7,10 @@ import (
 	"path/filepath"
 	"runtime"
 	"strings"
+	"time"

 	"github.com/ilyachch/mnemonic/internal/format/markdown"
+	mnemonicfs "github.com/ilyachch/mnemonic/internal/platform/fs"
 	"github.com/ilyachch/mnemonic/internal/platform/parallel"
 	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
 )
@@ -91,9 +93,9 @@ func scanOneNote(root, relPath string) (NoteDoc, error) {
 	if err != nil {
 		return NoteDoc{}, fmt.Errorf("%s: read note: %w", relPath, err)
 	}
-	st, err := os.Stat(abs)
+	btime, mtime, err := mnemonicfs.GetFileTimes(abs)
 	if err != nil {
-		return NoteDoc{}, fmt.Errorf("%s: stat note: %w", relPath, err)
+		return NoteDoc{}, fmt.Errorf("%s: stat note times: %w", relPath, err)
 	}
 	note, err := markdown.ParseNote(data)
 	if err != nil {
@@ -101,28 +103,38 @@ func scanOneNote(root, relPath string) (NoteDoc, error) {
 	}
 	info := NoteDoc{
 		NoteID:       note.MnemonicNoteID,
-		Slug:         note.EffectiveSlug(),
-		Title:        note.Title,
+		Slug:         note.GetOrDeriveSlug(relPath),
+		Title:        note.GetOrDeriveTitle(relPath),
 		RelPath:      relPath,
 		Frontmatter:  note.Frontmatter,
 		BodyMarkdown: string(note.Body),
 		BodyText:     string(note.Body),
 		ContentHash:  markdownstore.HashBytes(data),
-		FileMTimeNS:  st.ModTime().UnixNano(),
-		FileSize:     st.Size(),
+		FileMTimeNS:  mtime.UnixNano(),
+		FileSize:     int64(len(data)),
 	}
-	populateNoteDocData(&info, note, data)
+	populateNoteDocData(&info, note, data, btime, mtime)
 	return info, nil
 }

-func populateNoteDocData(info *NoteDoc, note markdown.Note, data []byte) {
+func populateNoteDocData(info *NoteDoc, note markdown.Note, data []byte, btime, mtime time.Time) {
 	info.Summary = note.Summary
 	info.Aliases = note.Aliases
-	if !note.CreatedAt.IsZero() {
-		info.CreatedAt = note.CreatedAt.Unix()
+
+	// CreatedAt: YAML value -> system btime -> system mtime.
+	if ca, ok := frontmatterUnixTime(note.Frontmatter, "created_at"); ok && ca > 0 {
+		info.CreatedAt = ca
+	} else if !btime.IsZero() {
+		info.CreatedAt = btime.Unix()
+	} else {
+		info.CreatedAt = mtime.Unix()
 	}
-	if !note.UpdatedAt.IsZero() {
-		info.UpdatedAt = note.UpdatedAt.Unix()
+
+	// UpdatedAt: YAML value -> system mtime.
+	if ua, ok := frontmatterUnixTime(note.Frontmatter, "updated_at"); ok && ua > 0 {
+		info.UpdatedAt = ua
+	} else {
+		info.UpdatedAt = mtime.Unix()
 	}
 	info.Tags = append(info.Tags, tagRow{Source: "frontmatter"})
 	for _, t := range note.Tags {
@@ -178,3 +190,22 @@ func normalizeTitleSlug(s string) string {
 	}
 	return strings.Trim(b.String(), "-")
 }
+
+// frontmatterUnixTime extracts a Unix epoch integer from raw frontmatter.
+// YAML unmarshals integers as `int` (or `int64`); both are accepted.
+func frontmatterUnixTime(fm map[string]any, key string) (int64, bool) {
+	if fm == nil {
+		return 0, false
+	}
+	value, ok := fm[key]
+	if !ok || value == nil {
+		return 0, false
+	}
+	switch v := value.(type) {
+	case int:
+		return int64(v), true
+	case int64:
+		return v, true
+	}
+	return 0, false
+}
```
