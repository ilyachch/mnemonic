package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyachch/mnemonic/internal/app"
	mnemonicfs "github.com/ilyachch/mnemonic/internal/fs"
	"github.com/ilyachch/mnemonic/internal/markdown"
	"github.com/ilyachch/mnemonic/internal/project"
)

// CreateInput configures note creation.
type CreateInput struct {
	RootDir string
	Title   string
	Body    []byte
	Tags    []string
	Now     func() time.Time
	UUID    func() string
}

// CreateResult describes the created note for CLI output.
type CreateResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

// Create writes a new markdown note beneath rootDir.
func Create(input CreateInput) (CreateResult, error) {
	if input.RootDir == "" {
		return CreateResult{}, fmt.Errorf("root directory is required")
	}
	if input.Title == "" {
		return CreateResult{}, app.NewCLIUsageError("note title is required", nil)
	}

	guard, err := acquireWriteLock(input.RootDir)
	if err != nil {
		return CreateResult{}, err
	}
	defer func() { _ = guard.Release() }()

	slug, err := project.Slugify(input.Title)
	if err != nil {
		return CreateResult{}, err
	}

	now := input.Now
	if now == nil {
		now = project.NowUTC
	}
	uuidFn := input.UUID
	if uuidFn == nil {
		uuidFn = project.NewUUID
	}

	timestamp := now().UTC()
	note := markdown.Note{
		MnemonicNoteID: uuidFn(),
		Title:          input.Title,
		Slug:           slug,
		Tags:           dedupeTags(input.Tags),
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		Body:           append([]byte(nil), input.Body...),
	}

	rendered, err := markdown.RenderNote(note)
	if err != nil {
		return CreateResult{}, err
	}

	relPath := slug + ".md"
	absPath := filepath.Join(input.RootDir, relPath)
	if _, err := os.Stat(absPath); err == nil {
		return CreateResult{}, app.NewAmbiguousError(fmt.Sprintf("note slug %q already exists", slug), nil)
	} else if !os.IsNotExist(err) {
		return CreateResult{}, fmt.Errorf("check note path %q: %w", absPath, err)
	}

	if err := mnemonicfs.AtomicWriteFile(absPath, rendered, 0o644); err != nil {
		return CreateResult{}, err
	}

	return CreateResult{
		NoteID:      note.MnemonicNoteID,
		Slug:        slug,
		Path:        filepath.ToSlash(relPath),
		ContentHash: HashBytes(rendered),
	}, nil
}

func dedupeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}

	return out
}
