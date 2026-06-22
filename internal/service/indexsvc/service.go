package indexsvc

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilyachch/mnemonic/internal/domain/kb"
	manifest "github.com/ilyachch/mnemonic/internal/format/manifest"
	"github.com/ilyachch/mnemonic/internal/format/markdown"
	"github.com/ilyachch/mnemonic/internal/store/markdownstore"
	"github.com/ilyachch/mnemonic/internal/store/sqliteindex"
)

// Service owns runtime index operations for one selected knowledge base.
type Service struct {
	KB    kb.KnowledgeBase
	Notes markdownstore.Store
	Index sqliteindex.Store
}

// RebuildOutput mirrors the runtime rebuild payload.
type RebuildOutput struct {
	KBID         string `json:"kb_id"`
	NotesSeen    int    `json:"notes_seen"`
	NotesIndexed int    `json:"notes_indexed"`
	Status       string `json:"status"`
}

// DoctorOutput mirrors the runtime health payload.
type DoctorOutput struct {
	Status string        `json:"status"`
	Checks []DoctorCheck `json:"checks"`
}

// DoctorCheck is one health check entry.
type DoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Count  int    `json:"count,omitempty"`
}

// New constructs the runtime index service for one knowledge base.
func New(k kb.KnowledgeBase) *Service {
	return &Service{
		KB: k,
		Notes: markdownstore.Store{
			RootDir: k.RootDir,
		},
		Index: sqliteindex.Store{
			IndexPath: k.IndexPath,
			RootDir:   k.RootDir,
			KBID:      k.ID,
		},
	}
}

// Rebuild rebuilds the bound index database.
func (s Service) Rebuild(ctx context.Context) (RebuildOutput, error) {
	_ = ctx
	result, err := s.Index.Rebuild()
	if err != nil {
		return RebuildOutput{}, err
	}
	return RebuildOutput{
		KBID:         result.KBID,
		NotesSeen:    result.NotesSeen,
		NotesIndexed: result.NotesIndexed,
		Status:       result.Status,
	}, nil
}

// Doctor runs health checks for the bound knowledge base.
func (s Service) Doctor(ctx context.Context) (DoctorOutput, error) {
	_ = ctx

	result := DoctorOutput{Status: "ok"}
	root := s.KB.RootDir

	result.addCheck(DoctorCheck{Name: "project path exists", Status: pathStatus(root)})
	if s.KB.ManifestPath != "" {
		result.addCheck(doctorParseCheck("mnemonic.toml", s.KB.ManifestPath))
	}

	exists, err := s.Index.Exists()
	if err != nil {
		return DoctorOutput{}, err
	}
	if !exists {
		result.Status = "needs_reindex"
		result.addCheck(DoctorCheck{Name: "index exists", Status: "missing"})
		return result, nil
	}
	result.addCheck(DoctorCheck{Name: "index exists", Status: "ok"})

	if err := s.Index.QuickCheck(); err != nil {
		return DoctorOutput{}, err
	}
	result.addCheck(DoctorCheck{Name: "index quick_check", Status: "ok"})

	db, err := s.Index.OpenReadonly()
	if err != nil {
		return DoctorOutput{}, err
	}
	defer func() { _ = db.Close() }()

	schemaStatus, err := s.Index.CheckSchemaStatus(db)
	if err != nil {
		return DoctorOutput{}, err
	}
	if schemaStatus != sqliteindex.SchemaStatusOK {
		result.Status = "needs_reindex"
		result.addCheck(DoctorCheck{Name: "index schema", Status: "needs_reindex"})
		return result, nil
	}
	result.addCheck(DoctorCheck{Name: "index schema", Status: "ok"})

	dupUUIDs, dupSlugs, unresolved, trashIgnored, err := doctorNoteChecks(root, db)
	if err != nil {
		return DoctorOutput{}, err
	}
	result.addCheck(DoctorCheck{Name: "duplicate note UUIDs", Status: countStatus(dupUUIDs), Count: dupUUIDs})
	result.addCheck(DoctorCheck{Name: "duplicate note slugs", Status: countStatus(dupSlugs), Count: dupSlugs})
	result.addCheck(DoctorCheck{Name: "unresolved link count", Status: countStatus(unresolved), Count: unresolved})
	result.addCheck(DoctorCheck{Name: ".trash ignored", Status: countStatus(trashIgnored), Count: trashIgnored})
	result.addCheck(doctorStaleTempCheck(root))

	return result, nil
}

func (r *DoctorOutput) addCheck(check DoctorCheck) {
	r.Checks = append(r.Checks, check)
	if r.Status == "ok" && check.Status == "warning" {
		r.Status = "warning"
	}
}

func countStatus(count int) string {
	if count == 0 {
		return "ok"
	}
	return "warning"
}

func pathStatus(path string) string {
	if _, err := os.Stat(path); err == nil {
		return "ok"
	}
	return "missing"
}

func doctorParseCheck(name, path string) DoctorCheck {
	data, err := os.ReadFile(path)
	if err != nil {
		return DoctorCheck{Name: name, Status: "missing", Detail: err.Error()}
	}
	if _, err := manifest.ParseMnemonicManifest(data); err != nil {
		return DoctorCheck{Name: name, Status: "error", Detail: err.Error()}
	}
	return DoctorCheck{Name: name, Status: "ok"}
}

func doctorNoteChecks(root string, db *sql.DB) (dupUUIDs int, dupSlugs int, unresolved int, trashIgnored int, err error) {
	paths, err := (markdownstore.Store{RootDir: root}).Walk()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	seenUUID := map[string]int{}
	seenSlug := map[string]int{}
	for _, rel := range paths {
		if strings.Contains(rel, ".trash/") || strings.HasPrefix(rel, ".trash/") || rel == ".trash" {
			trashIgnored++
		}
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			return 0, 0, 0, 0, readErr
		}
		note, parseErr := markdown.ParseNote(data)
		if parseErr != nil {
			return 0, 0, 0, 0, parseErr
		}
		if note.MnemonicNoteID != "" {
			seenUUID[note.MnemonicNoteID]++
		}
		if slug := note.EffectiveSlug(); slug != "" {
			seenSlug[slug]++
		}
	}
	for _, count := range seenUUID {
		if count > 1 {
			dupUUIDs++
		}
	}
	for _, count := range seenSlug {
		if count > 1 {
			dupSlugs++
		}
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM links WHERE to_note_id IS NULL`).Scan(&unresolved); err != nil {
		return 0, 0, 0, 0, err
	}
	return dupUUIDs, dupSlugs, unresolved, trashIgnored, nil
}

func doctorStaleTempCheck(root string) DoctorCheck {
	entries, err := os.ReadDir(root)
	if err != nil {
		return DoctorCheck{Name: "stale temp files", Status: "error", Detail: err.Error()}
	}
	count := 0
	firstPath := ""
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".tmp") {
			continue
		}
		count++
		if firstPath == "" {
			firstPath = filepath.Join(root, name)
		}
	}
	check := DoctorCheck{Name: "stale temp files", Status: countStatus(count), Count: count}
	if firstPath != "" {
		check.Detail = firstPath
	}
	return check
}
