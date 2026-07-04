package notesvc

import (
	"log/slog"
	"time"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
)

// Service owns runtime note operations for one selected knowledge base.
type Service struct {
	Notes  markdownstore.Store
	Index  sqliteindex.Store
	Logger *slog.Logger
}

// CreateInput configures note creation.
type CreateInput struct {
	Title string
	Body  []byte
	Tags  []string
	Now   func() time.Time
	UUID  func() string
}

// CreateResult describes the created note and index status.
type CreateResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

// EditInput configures note editing.
type EditInput struct {
	Selector string
	Append   []byte
	Body     []byte
	HasBody  bool
	Set      map[string]string
	IfMatch  string
	Now      func() time.Time
}

// EditResult describes the edited note and index status.
type EditResult struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

// DeleteInput configures note deletion.
type DeleteInput struct {
	Selector string
	DryRun   bool
	Hard     bool
	TrashDir string
	Yes      bool
	Now      func() time.Time
}

// DeleteResult describes the outcome of a note deletion and index status.
type DeleteResult struct {
	Mode        string `json:"mode"`
	Path        string `json:"path,omitempty"`
	TrashPath   string `json:"trash_path,omitempty"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

// NoteSummary aliases the markdownstore note summary type.
type NoteSummary = markdownstore.NoteSummary

// ShowResult aliases the markdownstore show result type.
type ShowResult = markdownstore.ShowResult

// ReadManyItem holds one resolved note from a batch read.
type ReadManyItem struct {
	ShowResult
	ID string
}

// ReadManyInput configures a batch note read operation.
type ReadManyInput struct {
	Selectors []string
}

// ReadManyOutput is the result of a batch read including missing selectors.
type ReadManyOutput struct {
	Notes   []ReadManyItem
	Missing []string
}

// New constructs the runtime notes service for one knowledge base.
func New(k kb.KnowledgeBase) *Service {
	return &Service{
		Notes: markdownstore.Store{RootDir: k.RootDir, StateDir: k.StateDir, IndexPath: k.IndexPath},
		Index: sqliteindex.Store{
			IndexPath: k.IndexPath,
			RootDir:   k.RootDir,
			StateDir:  k.StateDir,
			KBID:      k.ID,
		},
	}
}

// Create writes a note and rebuilds the index.
func (s Service) Create(input CreateInput) (CreateResult, error) {
	created, err := s.Notes.Create(markdownstore.CreateInput{
		Title: input.Title,
		Body:  input.Body,
		Tags:  input.Tags,
		Now:   input.Now,
		UUID:  input.UUID,
	})
	if err != nil {
		return CreateResult{}, err
	}

	result := CreateResult{
		NoteID:      created.NoteID,
		Slug:        created.Slug,
		Path:        created.Path,
		ContentHash: created.ContentHash,
		IndexStatus: "skipped",
	}
	if err := s.rebuildIndex(); err != nil {
		result.IndexStatus = "stale"
		result.IndexError = err.Error()
		if s.Logger != nil {
			s.Logger.Warn("note created but index rebuild failed", "note_id", created.NoteID, "path", created.Path, "error", err)
		}
		return result, nil
	}
	result.IndexStatus = "ok"
	if s.Logger != nil {
		s.Logger.Info("note created", "note_id", created.NoteID, "slug", created.Slug, "path", created.Path)
	}
	return result, nil
}

// Edit updates a note and rebuilds the index.
func (s Service) Edit(input EditInput) (EditResult, error) {
	edited, err := s.Notes.Edit(markdownstore.EditInput{
		Selector: input.Selector,
		Append:   input.Append,
		Body:     input.Body,
		HasBody:  input.HasBody,
		Set:      input.Set,
		IfMatch:  input.IfMatch,
		Now:      input.Now,
	})
	if err != nil {
		return EditResult{}, err
	}

	result := EditResult{
		NoteID:      edited.NoteID,
		Slug:        edited.Slug,
		Path:        edited.Path,
		ContentHash: edited.ContentHash,
		CreatedAt:   edited.CreatedAt,
		UpdatedAt:   edited.UpdatedAt,
		IndexStatus: "skipped",
	}
	if err := s.rebuildIndex(); err != nil {
		result.IndexStatus = "stale"
		result.IndexError = err.Error()
		if s.Logger != nil {
			s.Logger.Warn("note edited but index rebuild failed", "note_id", edited.NoteID, "error", err)
		}
		return result, nil
	}
	result.IndexStatus = "ok"
	if s.Logger != nil {
		s.Logger.Info("note edited", "note_id", edited.NoteID, "slug", edited.Slug, "path", edited.Path)
	}
	return result, nil
}

// Delete removes or trashes a note and rebuilds the index.
func (s Service) Delete(input DeleteInput) (DeleteResult, error) {
	deleted, err := s.Notes.Delete(markdownstore.DeleteInput{
		Selector: input.Selector,
		DryRun:   input.DryRun,
		Hard:     input.Hard,
		TrashDir: input.TrashDir,
		Yes:      input.Yes,
		Now:      input.Now,
	})
	if err != nil {
		return DeleteResult{}, err
	}

	result := DeleteResult{
		Mode:        deleted.Mode,
		Path:        deleted.Path,
		TrashPath:   deleted.TrashPath,
		IndexStatus: "skipped",
	}
	if input.DryRun {
		return result, nil
	}
	if err := s.rebuildIndex(); err != nil {
		result.IndexStatus = "stale"
		result.IndexError = err.Error()
		if s.Logger != nil {
			s.Logger.Warn("note deleted but index rebuild failed", "path", deleted.Path, "error", err)
		}
		return result, nil
	}
	result.IndexStatus = "ok"
	if s.Logger != nil {
		s.Logger.Info("note deleted", "path", deleted.Path, "mode", deleted.Mode)
	}
	return result, nil
}

// List returns note summaries from the bound store.
func (s Service) List() ([]NoteSummary, error) {
	return s.Notes.List()
}

// Show returns a parsed note from the bound store.
func (s Service) Show(selector string) (ShowResult, error) {
	result, err := s.Notes.Show(selector)
	if s.Logger != nil {
		if err != nil {
			s.Logger.Debug("note read failed", "selector", selector, "error", err)
		} else {
			s.Logger.Debug("note read", "note_id", result.Note.MnemonicNoteID, "selector", selector)
		}
	}
	return result, err
}

// ShowMany resolves multiple selectors and returns found notes plus missing identifiers.
func (s Service) ShowMany(input ReadManyInput) (ReadManyOutput, error) {
	var notes []ReadManyItem
	var missing []string
	for _, identifier := range input.Selectors {
		resolved, err := s.Show(identifier)
		if err != nil {
			missing = append(missing, identifier)
			continue
		}
		notes = append(notes, ReadManyItem{
			ShowResult: resolved,
			ID:         identifier,
		})
	}
	return ReadManyOutput{Notes: notes, Missing: missing}, nil
}

// HydrateResult aliases the markdownstore hydration payload.
type HydrateResult = markdownstore.HydrateResult

// Hydrate fills missing canonical frontmatter for raw notes via the bound
// store. See markdownstore.Store.Hydrate for details.
func (s Service) Hydrate(input markdownstore.HydrateInput) (HydrateResult, error) {
	return s.Notes.Hydrate(input)
}

func (s Service) rebuildIndex() error {
	_, err := s.Index.Rebuild()
	return err
}
