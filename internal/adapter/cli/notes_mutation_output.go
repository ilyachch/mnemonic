package cli

import "github.com/ilyachch/mnemonic/internal/service/notesvc"

type notesCreateOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

type notesEditOutput struct {
	NoteID      string `json:"note_id"`
	Slug        string `json:"slug"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

type notesDeleteOutput struct {
	Mode        string `json:"mode"`
	Path        string `json:"path,omitempty"`
	TrashPath   string `json:"trash_path,omitempty"`
	IndexStatus string `json:"index_status"`
	IndexError  string `json:"index_error,omitempty"`
}

func newNotesCreateOutput(result notesvc.CreateResult) notesCreateOutput {
	return notesCreateOutput{
		NoteID:      result.NoteID,
		Slug:        result.Slug,
		Path:        result.Path,
		ContentHash: result.ContentHash,
		IndexStatus: result.IndexStatus,
		IndexError:  result.IndexError,
	}
}

func newNotesEditOutput(result notesvc.EditResult) notesEditOutput {
	return notesEditOutput{
		NoteID:      result.NoteID,
		Slug:        result.Slug,
		Path:        result.Path,
		ContentHash: result.ContentHash,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
		IndexStatus: result.IndexStatus,
		IndexError:  result.IndexError,
	}
}

func newNotesDeleteOutput(result notesvc.DeleteResult) notesDeleteOutput {
	return notesDeleteOutput{
		Mode:        result.Mode,
		Path:        result.Path,
		TrashPath:   result.TrashPath,
		IndexStatus: result.IndexStatus,
		IndexError:  result.IndexError,
	}
}
