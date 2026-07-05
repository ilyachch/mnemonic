package indexsvc

import (
	"context"
	"log/slog"
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
	KB     kb.KnowledgeBase
	Notes  markdownstore.Store
	Index  sqliteindex.Store
	Logger *slog.Logger
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
func New(k kb.KnowledgeBase, logger *slog.Logger) *Service {
	return &Service{
		KB: k,
		Notes: markdownstore.Store{
			RootDir:  k.RootDir,
			StateDir: k.StateDir,
		},
		Index: sqliteindex.Store{
			IndexPath: k.IndexPath,
			RootDir:   k.RootDir,
			StateDir:  k.StateDir,
			KBID:      k.ID,
		},
		Logger: logger,
	}
}

// Rebuild rebuilds the bound index database.
func (s Service) Rebuild(ctx context.Context) (RebuildOutput, error) {
	_ = ctx
	result, err := s.Index.Rebuild()
	if err != nil {
		return RebuildOutput{}, err
	}

	if s.Logger != nil {
		s.Logger.Info("index rebuilt",
			"kb_id", result.KBID,
			"notes_seen", result.NotesSeen,
			"notes_indexed", result.NotesIndexed,
		)
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

	if err = s.Index.QuickCheck(); err != nil {
		return DoctorOutput{}, err
	}
	result.addCheck(DoctorCheck{Name: "index quick_check", Status: "ok"})

	schemaStatus, err := s.Index.SchemaStatus()
	if err != nil {
		return DoctorOutput{}, err
	}
	if schemaStatus != sqliteindex.SchemaStatusOK {
		result.Status = "needs_reindex"
		result.addCheck(DoctorCheck{Name: "index schema", Status: "needs_reindex"})
		return result, nil
	}
	result.addCheck(DoctorCheck{Name: "index schema", Status: "ok"})

	dupUUIDs, dupSlugs, unresolved, trashIgnored, err := doctorNoteChecks(root, s.Index)
	if err != nil {
		return DoctorOutput{}, err
	}
	result.addCheck(DoctorCheck{Name: "duplicate note UUIDs", Status: countStatus(dupUUIDs), Count: dupUUIDs})
	result.addCheck(DoctorCheck{Name: "duplicate note slugs", Status: countStatus(dupSlugs), Count: dupSlugs})
	result.addCheck(DoctorCheck{Name: "unresolved link count", Status: countStatus(unresolved), Count: unresolved})
	result.addCheck(DoctorCheck{Name: ".trash ignored", Status: countStatus(trashIgnored), Count: trashIgnored})
	result.addCheck(doctorStaleTempCheck(root))

	if s.Logger != nil {
		for _, check := range result.Checks {
			switch check.Status {
			case "warning":
				s.Logger.Warn("doctor check warning",
					"kb_slug", s.KB.Slug,
					"check", check.Name,
					"count", check.Count,
					"detail", check.Detail,
				)
			case "missing", "error":
				s.Logger.Warn("doctor check issue",
					"kb_slug", s.KB.Slug,
					"check", check.Name,
					"status", check.Status,
					"detail", check.Detail,
				)
			}
		}
	}

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

func doctorNoteChecks(root string, index sqliteindex.Store) (dupUUIDs int, dupSlugs int, unresolved int, trashIgnored int, err error) {
	paths, err := (markdownstore.Store{RootDir: root}).Walk()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	seenUUID, seenSlug, err := countNoteIDs(paths, root, &trashIgnored)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	dupUUIDs = countDups(seenUUID)
	dupSlugs = countDups(seenSlug)
	unresolved, err = index.UnresolvedLinkCount()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return dupUUIDs, dupSlugs, unresolved, trashIgnored, nil
}

func countNoteIDs(paths []string, root string, trashIgnored *int) (seenUUID, seenSlug map[string]int, err error) {
	seenUUID = map[string]int{}
	seenSlug = map[string]int{}
	for _, rel := range paths {
		if isTrashNote(rel) {
			*trashIgnored++
		}
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			return nil, nil, readErr
		}
		note, parseErr := markdown.ParseNote(data)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if note.MnemonicNoteID != "" {
			seenUUID[note.MnemonicNoteID]++
		}
		if slug := note.EffectiveSlug(); slug != "" {
			seenSlug[slug]++
		}
	}
	return
}

func isTrashNote(rel string) bool {
	return strings.Contains(rel, ".trash/") || strings.HasPrefix(rel, ".trash/") || rel == ".trash"
}

func countDups(m map[string]int) int {
	n := 0
	for _, count := range m {
		if count > 1 {
			n++
		}
	}
	return n
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
